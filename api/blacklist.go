package api

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gotify/server/v2/auth"
	"github.com/gotify/server/v2/model"
)

type BlacklistAPI struct {
	Blacklist *auth.AuthBlacklist
}

func (b *BlacklistAPI) validateIP(ctx *gin.Context, ip string) bool {
	if ip == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "IP parameter is required",
		})
		return false
	}

	if !auth.IsValidIP(ip) {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid IP parameter format",
		})
		return false
	}

	return true
}

// GetBlacklist returns the current blacklist
func (b *BlacklistAPI) GetBlacklist(ctx *gin.Context) {
	if !b.Blacklist.GetConfig().Enabled {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Blacklist feature is not enabled",
		})
		return
	}

	allBlockedIPs := b.Blacklist.GetAllBlockedIPs()
	now := time.Now()
	var blockedIPInfos []*model.BlockedIPInfo
	for _, info := range allBlockedIPs {
		isExpired := now.After(info.ExpiresAt)
		blockedIPInfos = append(blockedIPInfos, &model.BlockedIPInfo{
			IP:        info.IP,
			BlockedAt: info.BlockedAt.Format(time.RFC3339),
			ExpiresAt: info.ExpiresAt.Format(time.RFC3339),
			Reason:    info.Reason,
			Expired:   isExpired,
		})
	}

	ctx.JSON(http.StatusOK, model.BlacklistList{
		BlockedCount: len(blockedIPInfos),
		BlockedIPs:   blockedIPInfos,
	})
}

// GetIPStatus returns the status of a specific IP
func (b *BlacklistAPI) GetIPStatus(ctx *gin.Context) {
	if !b.Blacklist.GetConfig().Enabled {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Blacklist feature is not enabled",
		})
		return
	}

	ip := ctx.Param("ip")
	if !b.validateIP(ctx, ip) {
		return
	}

	isBlocked := b.Blacklist.IsBlocked(ip)
	isWhitelisted := b.Blacklist.IsWhitelisted(ip)
	failureCount := b.Blacklist.GetFailureCount(ip)

	response := model.IPStatus{
		IP:            ip,
		IsBlocked:     isBlocked,
		IsWhitelisted: isWhitelisted,
		FailureCount:  failureCount,
	}

	if isBlocked {
		blockedInfo := b.Blacklist.GetBlockedInfo(ip)
		response.ExpiresAt = blockedInfo.ExpiresAt.Format(time.RFC3339)
	}

	ctx.JSON(http.StatusOK, response)
}

// UnblockIP manually unblocks an IP
func (b *BlacklistAPI) UnblockIP(ctx *gin.Context) {
	if !b.Blacklist.GetConfig().Enabled {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Blacklist feature is not enabled",
		})
		return
	}

	ip := ctx.Param("ip")
	if !b.validateIP(ctx, ip) {
		return
	}

	if b.Blacklist.UnblockIP(ip) {
		log.Printf("[Blacklist] Admin unblocked IP %s", ip)
		ctx.JSON(http.StatusOK, model.SuccessResponse{
			Success: true,
			Message: "IP successfully unblocked",
		})
	} else {
		ctx.JSON(http.StatusNotFound, gin.H{
			"error": "IP not found in blacklist",
		})
	}
}

// ClearBlacklist clears all entries from the blacklist
func (b *BlacklistAPI) ClearBlacklist(ctx *gin.Context) {
	if !b.Blacklist.GetConfig().Enabled {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Blacklist feature is not enabled",
		})
		return
	}

	count := b.Blacklist.ClearAll()
	log.Printf("[Blacklist] Admin cleared blacklist (%d entries removed)", count)
	ctx.JSON(http.StatusOK, model.SuccessResponse{
		Success:      true,
		Message:      "Blacklist cleared successfully",
		ClearedCount: count,
	})
}

// GetWhitelist returns the current whitelist
func (b *BlacklistAPI) GetWhitelist(ctx *gin.Context) {
	whitelist := b.Blacklist.GetWhitelist()

	ctx.JSON(http.StatusOK, model.WhitelistInfo{
		Entries: whitelist,
		Count:   len(whitelist),
	})
}

// AddToWhitelist adds an IP or CIDR to the whitelist
func (b *BlacklistAPI) AddToWhitelist(ctx *gin.Context) {
	var req struct {
		Entry string `json:"entry" binding:"required"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request: " + err.Error(),
		})
		return
	}

	if err := b.Blacklist.AddToWhitelist(req.Entry); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid whitelist entry: " + err.Error(),
		})
		return
	}

	log.Printf("[Blacklist] Admin added %s to whitelist", req.Entry)
	ctx.JSON(http.StatusOK, model.SuccessResponse{
		Success: true,
		Message: "Entry added to whitelist",
	})
}

// RemoveFromWhitelist removes an IP or CIDR from the whitelist
func (b *BlacklistAPI) RemoveFromWhitelist(ctx *gin.Context) {
	ip := ctx.Param("ip")
	if !b.validateIP(ctx, ip) {
		return
	}

	b.Blacklist.RemoveFromWhitelist(ip)
	log.Printf("[Blacklist] Admin removed %s from whitelist", ip)

	ctx.JSON(http.StatusOK, model.SuccessResponse{
		Success: true,
		Message: "Entry removed from whitelist",
	})
}
