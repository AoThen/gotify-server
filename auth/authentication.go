package auth

import (
	"errors"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gotify/server/v2/auth/password"
	"github.com/gotify/server/v2/model"
)

const (
	headerName = "X-Gotify-Key"
)

// The Database interface for encapsulating database access.
type Database interface {
	GetApplicationByToken(token string) (*model.Application, error)
	GetClientByToken(token string) (*model.Client, error)
	GetPluginConfByToken(token string) (*model.PluginConf, error)
	GetUserByName(name string) (*model.User, error)
	GetUserByID(id uint) (*model.User, error)
	UpdateClientTokensLastUsed(tokens []string, t *time.Time) error
	UpdateApplicationTokenLastUsed(token string, t *time.Time) error
}

// Auth is the provider for authentication middleware.
type Auth struct {
	DB        Database
	Blacklist *AuthBlacklist
}

// SetBlacklist sets the blacklist for authentication
func (a *Auth) SetBlacklist(blacklist *AuthBlacklist) {
	a.Blacklist = blacklist
}

type authenticate func(tokenID string, user *model.User) (authenticated, success bool, userId uint, err error)

// getClientIP 获取客户端IP
func getClientIP(ctx *gin.Context) string {
	// 先检查 X-Forwarded-For 头
	if forwardedFor := ctx.GetHeader("X-Forwarded-For"); forwardedFor != "" {
		// X-Forwarded-For 可能包含多个IP，第一个是真实客户端IP
		parts := strings.Split(forwardedFor, ",")
		return strings.TrimSpace(parts[0])
	}
	// 检查 X-Real-IP 头
	if realIP := ctx.GetHeader("X-Real-IP"); realIP != "" {
		return realIP
	}
	// 从 RemoteAddr 获取
	return ctx.ClientIP()
}

// RequireAdmin returns a gin middleware which requires a client token or basic authentication header to be supplied
// with the request. Also the authenticated user must be an administrator.
func (a *Auth) RequireAdmin() gin.HandlerFunc {
	return a.requireToken(func(tokenID string, user *model.User) (bool, bool, uint, error) {
		if user != nil {
			return true, user.Admin, user.ID, nil
		}
		if token, err := a.DB.GetClientByToken(tokenID); err != nil {
			return false, false, 0, err
		} else if token != nil {
			user, err := a.DB.GetUserByID(token.UserID)
			if err != nil {
				return false, false, token.UserID, err
			}
			return true, user.Admin, token.UserID, nil
		}
		return false, false, 0, errors.New("invalid or unknown token")
	})
}

// RequireClient returns a gin middleware which requires a client token or basic authentication header to be supplied
// with the request.
func (a *Auth) RequireClient() gin.HandlerFunc {
	return a.requireToken(func(tokenID string, user *model.User) (bool, bool, uint, error) {
		if user != nil {
			return true, true, user.ID, nil
		}
		if client, err := a.DB.GetClientByToken(tokenID); err != nil {
			return false, false, 0, err
		} else if client != nil {
			if client.ExpiresAt != nil && client.ExpiresAt.Before(time.Now()) {
				return false, false, 0, errors.New("token has expired")
			}

			now := time.Now()
			if client.LastUsed == nil || client.LastUsed.Add(5*time.Minute).Before(now) {
				if err := a.DB.UpdateClientTokensLastUsed([]string{tokenID}, &now); err != nil {
					return false, false, 0, err
				}
			}
			return true, true, client.UserID, nil
		}
		return false, false, 0, errors.New("invalid or unknown token")
	})
}

// RequireApplicationToken returns a gin middleware which requires an application token to be supplied with the request.
func (a *Auth) RequireApplicationToken() gin.HandlerFunc {
	return a.requireToken(func(tokenID string, user *model.User) (bool, bool, uint, error) {
		if user != nil {
			return true, false, 0, nil
		}
		if app, err := a.DB.GetApplicationByToken(tokenID); err != nil {
			return false, false, 0, err
		} else if app != nil {
			if app.ExpiresAt != nil && app.ExpiresAt.Before(time.Now()) {
				return false, false, 0, errors.New("token has expired")
			}

			now := time.Now()
			if app.LastUsed == nil || app.LastUsed.Add(5*time.Minute).Before(now) {
				if err := a.DB.UpdateApplicationTokenLastUsed(tokenID, &now); err != nil {
					return false, false, 0, err
				}
			}
			return true, true, app.UserID, nil
		}
		return false, false, 0, errors.New("invalid or unknown token")
	})
}

func (a *Auth) tokenFromQueryOrHeader(ctx *gin.Context) string {
	if token := a.tokenFromQuery(ctx); token != "" {
		return token
	} else if token := a.tokenFromXGotifyHeader(ctx); token != "" {
		return token
	} else if token := a.tokenFromAuthorizationHeader(ctx); token != "" {
		return token
	}
	return ""
}

func (a *Auth) tokenFromQuery(ctx *gin.Context) string {
	return ctx.Request.URL.Query().Get("token")
}

func (a *Auth) tokenFromXGotifyHeader(ctx *gin.Context) string {
	return ctx.Request.Header.Get(headerName)
}

func (a *Auth) tokenFromAuthorizationHeader(ctx *gin.Context) string {
	const prefix = "Bearer "

	authHeader := ctx.Request.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	if len(authHeader) < len(prefix) || !strings.EqualFold(prefix, authHeader[:len(prefix)]) {
		return ""
	}

	return authHeader[len(prefix):]
}

func (a *Auth) userFromBasicAuth(ctx *gin.Context) (*model.User, bool) {
	if name, pass, ok := ctx.Request.BasicAuth(); ok {
		if user, err := a.DB.GetUserByName(name); err != nil {
			return nil, false
		} else if user != nil {
			if password.ComparePassword(user.Pass, []byte(pass)) {
				return user, true
			}
			return nil, false
		}
	}
	return nil, false
}

func (a *Auth) requireToken(auth authenticate) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		clientIP := getClientIP(ctx)

		if a.Blacklist != nil && a.Blacklist.GetConfig().Enabled {
			if !a.Blacklist.IsWhitelisted(clientIP) {
				if a.Blacklist.IsBlocked(clientIP) {
					blockedInfo := a.Blacklist.GetBlockedInfo(clientIP)
					ctx.AbortWithStatusJSON(429, gin.H{
						"error":            "Too Many Requests",
						"errorDescription": "IP is temporarily blocked due to too many authentication failures",
						"blockedUntil":     blockedInfo.ExpiresAt.Format(time.RFC3339),
						"blockedIP":        clientIP,
						"blockedAt":        blockedInfo.BlockedAt.Format(time.RFC3339),
						"blockedReason":    blockedInfo.Reason,
					})
					return
				}
			}
		}

		token := a.tokenFromQueryOrHeader(ctx)
		user, authValid := a.userFromBasicAuth(ctx)

		if !authValid && token == "" && (user != nil || token != "") {
			if a.Blacklist != nil && a.Blacklist.GetConfig().Enabled {
				if !a.Blacklist.IsWhitelisted(clientIP) {
					a.Blacklist.RecordFailure(clientIP)
				}
			}
		}

		if user != nil || token != "" {
			authenticated, ok, userID, err := auth(token, user)
			if err != nil {
				if a.Blacklist != nil && a.Blacklist.GetConfig().Enabled {
					if !a.Blacklist.IsWhitelisted(clientIP) {
						a.Blacklist.RecordFailure(clientIP)
					}
				}
				ctx.AbortWithError(401, errors.New("invalid token or credentials"))
				return
			} else if ok {
				if a.Blacklist != nil {
					a.Blacklist.ClearFailures(clientIP)
				}
				RegisterAuthentication(ctx, user, userID, token)
				ctx.Next()
				return
			} else if authenticated {
				if a.Blacklist != nil && a.Blacklist.GetConfig().Enabled {
					if !a.Blacklist.IsWhitelisted(clientIP) {
						a.Blacklist.RecordFailure(clientIP)
					}
				}
				ctx.AbortWithError(403, errors.New("you are not allowed to access this api"))
				return
			}
		}

		if a.Blacklist != nil && a.Blacklist.GetConfig().Enabled {
			if !a.Blacklist.IsWhitelisted(clientIP) {
				a.Blacklist.RecordFailure(clientIP)
			}
		}
		ctx.AbortWithError(401, errors.New("you need to provide a valid access token or user credentials to access this api"))
	}
}

func (a *Auth) Optional() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		clientIP := getClientIP(ctx)

		if a.Blacklist != nil && a.Blacklist.GetConfig().Enabled {
			if !a.Blacklist.IsWhitelisted(clientIP) {
				if a.Blacklist.IsBlocked(clientIP) {
					blockedInfo := a.Blacklist.GetBlockedInfo(clientIP)
					ctx.AbortWithStatusJSON(429, gin.H{
						"error":            "Too Many Requests",
						"errorDescription": "IP is temporarily blocked due to too many authentication failures",
						"blockedUntil":     blockedInfo.ExpiresAt.Format(time.RFC3339),
						"blockedIP":        clientIP,
						"blockedAt":        blockedInfo.BlockedAt.Format(time.RFC3339),
						"blockedReason":    blockedInfo.Reason,
					})
					return
				}
			}
		}

		token := a.tokenFromQueryOrHeader(ctx)
		user, authValid := a.userFromBasicAuth(ctx)

		if user != nil && authValid {
			if a.Blacklist != nil {
				a.Blacklist.ClearFailures(clientIP)
			}
			RegisterAuthentication(ctx, user, user.ID, token)
			ctx.Next()
			return
		} else if token != "" {
			if tokenClient, err := a.DB.GetClientByToken(token); err == nil && tokenClient != nil {
				if a.Blacklist != nil {
					a.Blacklist.ClearFailures(clientIP)
				}
				RegisterAuthentication(ctx, user, tokenClient.UserID, token)
				ctx.Next()
				return
			}
			if a.Blacklist != nil && a.Blacklist.GetConfig().Enabled {
				if !a.Blacklist.IsWhitelisted(clientIP) {
					a.Blacklist.RecordFailure(clientIP)
				}
			}
		}
		RegisterAuthentication(ctx, nil, 0, "")
		ctx.Next()
	}
}
