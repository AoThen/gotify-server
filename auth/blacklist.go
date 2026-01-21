package auth

import (
	"net"
	"sync"
	"time"
)

type AuthBlacklist struct {
	mu        sync.RWMutex
	blocked   map[string]*BlockedIP
	whitelist []*net.IPNet
	failures  map[string]*AuthFailure
	config    *Config
	stopCh    chan struct{}
}

type BlockedIP struct {
	IP        string
	BlockedAt time.Time
	ExpiresAt time.Time
	Reason    string
}

type AuthFailure struct {
	IP        string
	Count     int
	FirstFail time.Time
	LastFail  time.Time
}

type Config struct {
	Enabled         bool
	MaxFailures     int
	WindowSeconds   int
	BlockDuration   int
	Whitelist       []string
	CleanupInterval int
}

func NewAuthBlacklist(conf Config) *AuthBlacklist {
	ab := &AuthBlacklist{
		blocked:  make(map[string]*BlockedIP),
		failures: make(map[string]*AuthFailure),
		config:   &conf,
		stopCh:   make(chan struct{}),
	}

	whitelist, err := ab.parseWhitelist(conf.Whitelist)
	if err != nil {
		panic(err.Error())
	}
	ab.whitelist = whitelist

	if conf.Enabled {
		go ab.cleanupLoop()
	}

	return ab
}

func (ab *AuthBlacklist) GetConfig() *Config {
	return ab.config
}

func (ab *AuthBlacklist) parseWhitelist(whitelist []string) ([]*net.IPNet, error) {
	var networks []*net.IPNet
	for _, entry := range whitelist {
		_, ipNet, err := net.ParseCIDR(entry)
		if err != nil {
			ip := net.ParseIP(entry)
			if ip != nil {
				if ip.To4() != nil {
					ipNet = &net.IPNet{IP: ip, Mask: net.CIDRMask(32, 32)}
				} else {
					ipNet = &net.IPNet{IP: ip, Mask: net.CIDRMask(128, 128)}
				}
			} else {
				return nil, err
			}
		}
		networks = append(networks, ipNet)
	}
	return networks, nil
}

func (ab *AuthBlacklist) IsWhitelisted(ip string) bool {
	ab.mu.RLock()
	defer ab.mu.RUnlock()

	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	for _, ipNet := range ab.whitelist {
		if ipNet.Contains(parsedIP) {
			return true
		}
	}
	return false
}

func (ab *AuthBlacklist) IsBlocked(ip string) bool {
	ab.mu.RLock()
	defer ab.mu.RUnlock()

	blockedInfo, exists := ab.blocked[ip]
	if !exists {
		return false
	}

	if time.Now().Before(blockedInfo.ExpiresAt) {
		return true
	}

	return false
}

func (ab *AuthBlacklist) GetBlockedInfo(ip string) *BlockedIP {
	ab.mu.RLock()
	defer ab.mu.RUnlock()
	return ab.blocked[ip]
}

func (ab *AuthBlacklist) RecordFailure(ip string) {
	if !ab.config.Enabled {
		return
	}

	ab.mu.Lock()
	defer ab.mu.Unlock()

	now := time.Now()
	failure, exists := ab.failures[ip]

	if !exists {
		ab.failures[ip] = &AuthFailure{
			IP:        ip,
			Count:     1,
			FirstFail: now,
			LastFail:  now,
		}
		return
	}

	windowStart := now.Add(-time.Duration(ab.config.WindowSeconds) * time.Second)
	if failure.FirstFail.Before(windowStart) {
		failure.Count = 1
		failure.FirstFail = now
		failure.LastFail = now
	} else {
		failure.Count++
		failure.LastFail = now
	}

	if failure.Count >= ab.config.MaxFailures {
		ab.blockIP(ip, "Too many authentication failures")
	}
}

func (ab *AuthBlacklist) ClearFailures(ip string) {
	ab.mu.Lock()
	defer ab.mu.Unlock()
	delete(ab.failures, ip)
	delete(ab.blocked, ip)
}

func (ab *AuthBlacklist) blockIP(ip string, reason string) {
	now := time.Now()
	ab.blocked[ip] = &BlockedIP{
		IP:        ip,
		BlockedAt: now,
		ExpiresAt: now.Add(time.Duration(ab.config.BlockDuration) * time.Second),
		Reason:    reason,
	}
	delete(ab.failures, ip)
}

func (ab *AuthBlacklist) UnblockIP(ip string) bool {
	ab.mu.Lock()
	defer ab.mu.Unlock()

	delete(ab.blocked, ip)
	delete(ab.failures, ip)
	return true
}

func (ab *AuthBlacklist) ClearAll() int {
	ab.mu.Lock()
	defer ab.mu.Unlock()

	count := len(ab.blocked)
	ab.blocked = make(map[string]*BlockedIP)
	ab.failures = make(map[string]*AuthFailure)
	return count
}

func (ab *AuthBlacklist) GetBlockedIPs() []*BlockedIP {
	ab.mu.RLock()
	defer ab.mu.RUnlock()

	now := time.Now()
	var blocked []*BlockedIP
	for _, info := range ab.blocked {
		if now.Before(info.ExpiresAt) {
			blocked = append(blocked, info)
		}
	}
	return blocked
}

func (ab *AuthBlacklist) GetFailureCount(ip string) int {
	ab.mu.RLock()
	defer ab.mu.RUnlock()

	if failure, exists := ab.failures[ip]; exists {
		return failure.Count
	}
	return 0
}

func (ab *AuthBlacklist) GetWhitelist() []string {
	ab.mu.RLock()
	defer ab.mu.RUnlock()

	var list []string
	for _, ipNet := range ab.whitelist {
		list = append(list, ipNet.String())
	}
	return list
}

func (ab *AuthBlacklist) AddToWhitelist(entry string) error {
	ab.mu.Lock()
	defer ab.mu.Unlock()

	networks, err := ab.parseWhitelist([]string{entry})
	if err != nil {
		return err
	}

	ab.whitelist = append(ab.whitelist, networks[0])
	return nil
}

func (ab *AuthBlacklist) RemoveFromWhitelist(entry string) {
	ab.mu.Lock()
	defer ab.mu.Unlock()

	ip := net.ParseIP(entry)
	var newWhitelist []*net.IPNet
	for _, ipNet := range ab.whitelist {
		if ip == nil || !ipNet.Contains(ip) {
			newWhitelist = append(newWhitelist, ipNet)
		}
	}
	ab.whitelist = newWhitelist
}

func (ab *AuthBlacklist) Close() {
	close(ab.stopCh)
}

func (ab *AuthBlacklist) cleanupLoop() {
	ticker := time.NewTicker(time.Duration(ab.config.CleanupInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ab.cleanup()
		case <-ab.stopCh:
			return
		}
	}
}

func (ab *AuthBlacklist) cleanup() {
	ab.mu.Lock()
	defer ab.mu.Unlock()

	now := time.Now()
	for ip, blockedInfo := range ab.blocked {
		if now.After(blockedInfo.ExpiresAt) {
			delete(ab.blocked, ip)
		}
	}

	for ip, failure := range ab.failures {
		windowStart := now.Add(-time.Duration(ab.config.WindowSeconds) * time.Second)
		if failure.FirstFail.Before(windowStart) {
			delete(ab.failures, ip)
		}
	}
}
