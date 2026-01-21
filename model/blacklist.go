package model

// BlockedIPInfo represents a blocked IP entry
type BlockedIPInfo struct {
	IP        string `json:"ip"`
	BlockedAt string `json:"blockedAt"`
	ExpiresAt string `json:"expiresAt"`
	Reason    string `json:"reason"`
}

// IPStatus represents the current status of an IP
type IPStatus struct {
	IP            string `json:"ip"`
	IsBlocked     bool   `json:"isBlocked"`
	IsWhitelisted bool  `json:"isWhitelisted"`
	FailureCount  int    `json:"failureCount,omitempty"`
	ExpiresAt     string `json:"expiresAt,omitempty"`
}

// BlacklistList represents a list of blocked IPs
type BlacklistList struct {
	BlockedCount int              `json:"blockedCount"`
	BlockedIPs   []*BlockedIPInfo `json:"blockedIPs"`
}

// SuccessResponse represents a generic success response
type SuccessResponse struct {
	Success      bool   `json:"success"`
	Message      string `json:"message,omitempty"`
	ClearedCount int    `json:"clearedCount,omitempty"`
}

// WhitelistInfo represents whitelist entries
type WhitelistInfo struct {
	Entries []string `json:"entries"`
	Count   int      `json:"count"`
}
