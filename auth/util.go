package auth

import (
	"github.com/gin-gonic/gin"
	"github.com/gotify/server/v2/model"
)

// RegisterAuthentication registers the user id, user and or token.
func RegisterAuthentication(ctx *gin.Context, user *model.User, userID uint, tokenID string) {
	ctx.Set("user", user)
	ctx.Set("userid", userID)
	ctx.Set("tokenid", tokenID)
}

// GetUserID returns the user id which was previously registered by RegisterAuthentication.
func GetUserID(ctx *gin.Context) uint {
	id := TryGetUserID(ctx)
	if id == nil {
		panic("token and user may not be null")
	}
	return *id
}

// TryGetUserID returns the user id or nil if one is not set.
func TryGetUserID(ctx *gin.Context) *uint {
	userValue, exists := ctx.Get("user")
	if exists {
		user, ok := userValue.(*model.User)
		if ok && user != nil {
			return &user.ID
		}
	}

	userIDValue, exists := ctx.Get("userid")
	if exists {
		userID, ok := userIDValue.(uint)
		if ok && userID != 0 {
			return &userID
		}
	}

	return nil
}

// GetTokenID returns the tokenID.
func GetTokenID(ctx *gin.Context) string {
	return ctx.MustGet("tokenid").(string)
}
