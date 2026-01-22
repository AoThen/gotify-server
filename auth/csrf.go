package auth

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type CSRFMiddleware struct {
	secret     string
	tokenStore map[string]*csrfToken
	mu         sync.RWMutex
	maxAge     time.Duration
}

type csrfToken struct {
	token     string
	expiresAt time.Time
	userID    uint
}

const (
	csrfCookieName  = "csrf_token"
	csrfHeaderName  = "X-CSRF-Token"
	csrfTokenLength = 32
)

func NewCSRFMiddleware(secret string) *CSRFMiddleware {
	if secret == "" {
		secretBytes := make([]byte, 32)
		rand.Read(secretBytes)
		secret = base64.StdEncoding.EncodeToString(secretBytes)
	}

	return &CSRFMiddleware{
		secret:     secret,
		tokenStore: make(map[string]*csrfToken),
		maxAge:     24 * time.Hour,
	}
}

func (m *CSRFMiddleware) Middleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if ctx.Request.Method == http.MethodGet || ctx.Request.Method == http.MethodHead || ctx.Request.Method == http.MethodOptions {
			cookie, _ := ctx.Cookie(csrfCookieName)
			if cookie == "" {
				m.setCSRFCookie(ctx)
			}
			ctx.Next()
			return
		}

		if ctx.GetHeader("X-Requested-With") == "XMLHttpRequest" {
			expectedToken, _ := ctx.Cookie(csrfCookieName)
			providedToken := ctx.GetHeader(csrfHeaderName)

			if expectedToken == "" || providedToken == "" {
				ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"error":            "Missing CSRF token",
					"errorCode":        403,
					"errorDescription": "CSRF token is missing or invalid",
				})
				return
			}

			if !m.validateToken(expectedToken, providedToken) {
				ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"error":            "Invalid CSRF token",
					"errorCode":        403,
					"errorDescription": "CSRF token is invalid",
				})
				return
			}
		}

		m.setCSRFCookie(ctx)
		ctx.Next()
	}
}

func (m *CSRFMiddleware) setCSRFCookie(ctx *gin.Context) {
	token := m.generateToken()
	userID := GetUserID(ctx)

	m.mu.Lock()
	m.tokenStore[token] = &csrfToken{
		token:     token,
		expiresAt: time.Now().Add(m.maxAge),
		userID:    userID,
	}
	m.mu.Unlock()

	ctx.SetCookie(csrfCookieName, token, int(m.maxAge.Seconds()), "/", "", false, true)
}

func (m *CSRFMiddleware) generateToken() string {
	tokenBytes := make([]byte, csrfTokenLength)
	rand.Read(tokenBytes)
	return base64.StdEncoding.EncodeToString(tokenBytes)
}

func (m *CSRFMiddleware) validateToken(cookieToken, headerToken string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	now := time.Now()
	for token, entry := range m.tokenStore {
		if now.After(entry.expiresAt) {
			delete(m.tokenStore, token)
			continue
		}
		if token == cookieToken && token == headerToken {
			return true
		}
	}
	return false
}

func (m *CSRFMiddleware) Cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for token, entry := range m.tokenStore {
		if now.After(entry.expiresAt) {
			delete(m.tokenStore, token)
		}
	}
}

func (m *CSRFMiddleware) StartCleanupLoop(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			m.Cleanup()
		}
	}()
}

func IsValidIP(ip string) bool {
	if len(ip) == 0 || len(ip) > 45 {
		return false
	}

	if strings.Contains(ip, "/") || strings.Contains(ip, "\\") {
		return false
	}

	parts := strings.Split(ip, ".")
	if len(parts) == 4 {
		for _, part := range parts {
			if len(part) == 0 || len(part) > 3 {
				return false
			}
			for _, c := range part {
				if c < '0' || c > '9' {
					return false
				}
			}
		}
	}

	return true
}
