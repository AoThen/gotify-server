package router

import (
	"fmt"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gotify/location"
	"github.com/gotify/server/v2/api"
	"github.com/gotify/server/v2/api/stream"
	"github.com/gotify/server/v2/auth"
	"github.com/gotify/server/v2/config"
	"github.com/gotify/server/v2/database"
	"github.com/gotify/server/v2/docs"
	gerror "github.com/gotify/server/v2/error"
	"github.com/gotify/server/v2/model"
	"github.com/gotify/server/v2/plugin"
	"github.com/gotify/server/v2/ui"
	"golang.org/x/time/rate"
)

func Create(db *database.GormDatabase, vInfo *model.VersionInfo, conf *config.Configuration) (*gin.Engine, func()) {
	g := gin.New()

	g.RemoveExtraSlash = true
	g.RemoteIPHeaders = []string{"X-Forwarded-For"}
	g.SetTrustedProxies(conf.Server.TrustedProxies)
	g.ForwardedByClientIP = true

	g.Use(func(ctx *gin.Context) {
		if ctx.Request.RemoteAddr == "@" {
			ctx.Request.RemoteAddr = "127.0.0.1:65535"
		}
	})

	g.Use(gin.LoggerWithFormatter(logFormatter), gin.Recovery(), gerror.Handler(), location.Default())
	g.NoRoute(gerror.NotFound())

	g.Use(func(ctx *gin.Context) {
		ctx.Header("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'")
		ctx.Header("X-Content-Type-Options", "nosniff")
		ctx.Header("X-Frame-Options", "DENY")
		ctx.Header("X-XSS-Protection", "1; mode=block")
		ctx.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		if conf.Server.SSL.Enabled {
			ctx.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		ctx.Next()
	})

	if conf.Server.SSL.Enabled && conf.Server.SSL.RedirectToHTTPS {
		g.Use(func(ctx *gin.Context) {
			if ctx.Request.TLS != nil {
				ctx.Next()
				return
			}
			if ctx.Request.Method != http.MethodGet && ctx.Request.Method != http.MethodHead {
				ctx.Data(http.StatusBadRequest, "text/plain; charset=utf-8", []byte("Use HTTPS"))
				ctx.Abort()
				return
			}
			host := ctx.Request.Host
			if idx := strings.LastIndex(host, ":"); idx != -1 {
				host = host[:idx]
			}
			if conf.Server.SSL.Port != 443 {
				host = fmt.Sprintf("%s:%d", host, conf.Server.SSL.Port)
			}
			ctx.Redirect(http.StatusFound, fmt.Sprintf("https://%s%s", host, ctx.Request.RequestURI))
			ctx.Abort()
		})
	}
	streamHandler := stream.New(
		time.Duration(conf.Server.Stream.PingPeriodSeconds)*time.Second, 15*time.Second, conf.Server.Stream.AllowedOrigins)
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		for range ticker.C {
			connectedTokens := streamHandler.CollectConnectedClientTokens()
			now := time.Now()
			db.UpdateClientTokensLastUsed(connectedTokens, &now)
		}
	}()
	authentication := auth.Auth{DB: db}
	messageHandler := api.MessageAPI{Notifier: streamHandler, DB: db}
	healthHandler := api.HealthAPI{DB: db}
	clientHandler := api.ClientAPI{
		DB:            db,
		ImageDir:      conf.UploadedImagesDir,
		NotifyDeleted: streamHandler.NotifyDeletedClient,
	}
	applicationHandler := api.ApplicationAPI{
		DB:       db,
		ImageDir: conf.UploadedImagesDir,
	}
	userChangeNotifier := new(api.UserChangeNotifier)
	userHandler := api.UserAPI{DB: db, PasswordStrength: conf.PassStrength, UserChangeNotifier: userChangeNotifier, Registration: conf.Registration}

	pluginManager, err := plugin.NewManager(db, conf.PluginsDir, g.Group("/plugin/:id/custom/"), streamHandler)
	if err != nil {
		panic(err)
	}
	pluginHandler := api.PluginAPI{
		Manager:  pluginManager,
		Notifier: streamHandler,
		DB:       db,
	}

	userChangeNotifier.OnUserDeleted(streamHandler.NotifyDeletedUser)
	userChangeNotifier.OnUserDeleted(pluginManager.RemoveUser)
	userChangeNotifier.OnUserAdded(pluginManager.InitializeForUserID)

	// Tiered rate limiters
	var globalRateLimiter, authRateLimiter, messageRateLimiter, adminRateLimiter *auth.RateLimiter

	// Level 1: Global rate limiter (all requests)
	if conf.Server.RateLimit.Global.Enabled {
		r := rate.Limit(float64(conf.Server.RateLimit.Global.RequestsPerSecond))
		globalRateLimiter = auth.NewRateLimiter(r, conf.Server.RateLimit.Global.Burst)
	}

	// Level 2: Auth API rate limiter (Client Token, Admin)
	if conf.Server.RateLimit.Auth.Enabled {
		r := rate.Limit(float64(conf.Server.RateLimit.Auth.RequestsPerSecond))
		authRateLimiter = auth.NewRateLimiter(r, conf.Server.RateLimit.Auth.Burst)
	}

	// Level 3: Message sending rate limiter (Application Token)
	if conf.Server.RateLimit.Message.Enabled {
		r := rate.Limit(float64(conf.Server.RateLimit.Message.RequestsPerSecond))
		messageRateLimiter = auth.NewRateLimiter(r, conf.Server.RateLimit.Message.Burst)
	}

	// Level 4: Admin API rate limiter
	if conf.Server.RateLimit.Admin.Enabled {
		r := rate.Limit(float64(conf.Server.RateLimit.Admin.RequestsPerSecond))
		adminRateLimiter = auth.NewRateLimiter(r, conf.Server.RateLimit.Admin.Burst)
	}

	// Initialize blacklist manager
	var authBlacklist *auth.AuthBlacklist
	if conf.Server.AuthBlacklist.Enabled {
		blacklistConfig := auth.Config{
			Enabled:         conf.Server.AuthBlacklist.Enabled,
			MaxFailures:     conf.Server.AuthBlacklist.MaxFailures,
			WindowSeconds:   conf.Server.AuthBlacklist.WindowSeconds,
			BlockDuration:   conf.Server.AuthBlacklist.BlockDuration,
			Whitelist:       conf.Server.AuthBlacklist.Whitelist,
			CleanupInterval: conf.Server.AuthBlacklist.CleanupInterval,
		}
		authBlacklist = auth.NewAuthBlacklist(blacklistConfig)
	}

	ui.Register(g, *vInfo, conf.Registration)

	// Apply global rate limit to all requests
	if globalRateLimiter != nil {
		g.Use(globalRateLimiter.RateLimit())
	}

	g.Match([]string{"GET", "HEAD"}, "/health", healthHandler.Health)
	g.GET("/swagger", docs.Serve)
	g.StaticFS("/image", &onlyImageFS{inner: gin.Dir(conf.UploadedImagesDir, false)})

	g.GET("/docs", docs.UI)

	g.Use(func(ctx *gin.Context) {
		ctx.Header("Content-Type", "application/json")
		for header, value := range conf.Server.ResponseHeaders {
			ctx.Header(header, value)
		}
	})
	g.Use(cors.New(auth.CorsConfig(conf)))

	{
		// Plugin API with auth rate limiting
		pluginGroup := g.Group("/plugin/")
		{
			if authRateLimiter != nil {
				pluginGroup.Use(authRateLimiter.RateLimit())
			}
			g.GET("/plugin", authentication.RequireClient(), pluginHandler.GetPlugins)
			pluginRoute := pluginGroup.Use(authentication.RequireClient())
			{
				pluginRoute.GET("/:id/config", pluginHandler.GetConfig)
				pluginRoute.POST("/:id/config", pluginHandler.UpdateConfig)
				pluginRoute.GET("/:id/display", pluginHandler.GetDisplay)
				pluginRoute.POST("/:id/enable", pluginHandler.EnablePlugin)
				pluginRoute.POST("/:id/disable", pluginHandler.DisablePlugin)
			}
		}

		// User registration with auth rate limiting
		regRoute := g.Group("/user")
		{
			if authRateLimiter != nil {
				regRoute.Use(authRateLimiter.RateLimit())
			}
			regRoute.Use(authentication.Optional())
			regRoute.POST("", userHandler.CreateUser)
		}
	}

	g.OPTIONS("/*any")

	// swagger:operation GET /version version getVersion
	g.GET("version", func(ctx *gin.Context) {
		ctx.JSON(200, vInfo)
	})

	// Application Token routes with message rate limiting
	appAuth := g.Group("")
	{
		if messageRateLimiter != nil {
			appAuth.Use(messageRateLimiter.RateLimit())
		}
		appAuth.Use(authentication.RequireApplicationToken())
		appAuth.POST("/message", messageHandler.CreateMessage)
	}

	// Client Token routes with auth rate limiting
	clientAuth := g.Group("")
	{
		if authRateLimiter != nil {
			clientAuth.Use(authRateLimiter.RateLimit())
		}
		clientAuth.Use(authentication.RequireClient())
		app := clientAuth.Group("/application")
		{
			app.GET("", applicationHandler.GetApplications)
			app.POST("", applicationHandler.CreateApplication)
			app.POST("/:id/image", applicationHandler.UploadApplicationImage)
			app.DELETE("/:id/image", applicationHandler.RemoveApplicationImage)
			app.PUT("/:id", applicationHandler.UpdateApplication)
			app.DELETE("/:id", applicationHandler.DeleteApplication)

			tokenMessage := app.Group("/:id/message")
			{
				tokenMessage.GET("", messageHandler.GetMessagesWithApplication)
				tokenMessage.DELETE("", messageHandler.DeleteMessageWithApplication)
			}
		}

		client := clientAuth.Group("/client")
		{
			client.GET("", clientHandler.GetClients)
			client.POST("", clientHandler.CreateClient)
			client.DELETE("/:id", clientHandler.DeleteClient)
			client.PUT("/:id", clientHandler.UpdateClient)
		}

		message := clientAuth.Group("/message")
		{
			message.GET("", messageHandler.GetMessages)
			message.DELETE("", messageHandler.DeleteMessages)
			message.DELETE("/:id", messageHandler.DeleteMessage)
		}

		clientAuth.GET("/stream", streamHandler.Handle)
		clientAuth.GET("current/user", userHandler.GetCurrentUser)
		clientAuth.POST("current/user/password", userHandler.ChangePassword)
	}

	// Admin routes with admin rate limiting
	authAdmin := g.Group("/user")
	{
		if adminRateLimiter != nil {
			authAdmin.Use(adminRateLimiter.RateLimit())
		}
		authAdmin.Use(authentication.RequireAdmin())

		authAdmin.GET("", userHandler.GetUsers)
		authAdmin.DELETE("/:id", userHandler.DeleteUserByID)
		authAdmin.GET("/:id", userHandler.GetUserByID)
		authAdmin.POST("/:id", userHandler.UpdateUserByID)
	}

	// Blacklist management API (Phase 2)
	if authBlacklist != nil {
		blacklistAPI := &api.BlacklistAPI{Blacklist: authBlacklist}
		blacklistGroup := g.Group("/admin/blacklist")
		{
			blacklistGroup.Use(authentication.RequireAdmin())
			blacklistGroup.GET("", blacklistAPI.GetBlacklist)
			blacklistGroup.GET("/:ip", blacklistAPI.GetIPStatus)
			blacklistGroup.DELETE("/:ip", blacklistAPI.UnblockIP)
			blacklistGroup.POST("/clear-all", blacklistAPI.ClearBlacklist)
		}

		whitelistGroup := g.Group("/admin/whitelist")
		{
			whitelistGroup.Use(authentication.RequireAdmin())
			whitelistGroup.GET("", blacklistAPI.GetWhitelist)
			whitelistGroup.POST("", blacklistAPI.AddToWhitelist)
			whitelistGroup.DELETE("/:ip", blacklistAPI.RemoveFromWhitelist)
		}
	}

	return g, streamHandler.Close
}

var tokenRegexp = regexp.MustCompile("token=[^&]+")

func logFormatter(param gin.LogFormatterParams) string {
	if (param.ClientIP == "127.0.0.1" || param.ClientIP == "::1") && param.Path == "/health" {
		return ""
	}

	var statusColor, methodColor, resetColor string
	if param.IsOutputColor() {
		statusColor = param.StatusCodeColor()
		methodColor = param.MethodColor()
		resetColor = param.ResetColor()
	}

	if param.Latency > time.Minute {
		param.Latency = param.Latency - param.Latency%time.Second
	}
	path := tokenRegexp.ReplaceAllString(param.Path, "token=[masked]")
	return fmt.Sprintf("%v |%s %3d %s| %13v | %15s |%s %-7s %s %#v\n%s",
		param.TimeStamp.Format(time.RFC3339),
		statusColor, param.StatusCode, resetColor,
		param.Latency,
		param.ClientIP,
		methodColor, param.Method, resetColor,
		path,
		param.ErrorMessage,
	)
}

type onlyImageFS struct {
	inner http.FileSystem
}

func (fs *onlyImageFS) Open(name string) (http.File, error) {
	ext := filepath.Ext(name)
	if !api.ValidApplicationImageExt(ext) {
		return nil, fmt.Errorf("invalid file")
	}
	return fs.inner.Open(name)
}
