package api

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"clortho/internal/api/handlers"
	"clortho/internal/api/middleware"
	"clortho/internal/config"
	"clortho/internal/store"
)

type Server struct {
	Router            *gin.Engine
	DB                *pgxpool.Pool
	Config            config.Config

	LicenseStore      store.LicenseStore
	ProductStore      store.ProductStore
	ProductGroupStore store.ProductGroupStore
	ReleaseStore      store.ReleaseStore
	FeatureStore      store.FeatureStore
	LogStore          store.LogStore
	StatsStore        store.StatsStore
}

func NewServer(cfg config.Config, db *pgxpool.Pool, ls store.LicenseStore, ps store.ProductStore, pgs store.ProductGroupStore, rs store.ReleaseStore, fs store.FeatureStore, logs store.LogStore, ss store.StatsStore) *Server {
	r := gin.Default()

	r.Use(middleware.ResponseSigningMiddleware(cfg.ResponseSigningPrivateKey))
	if len(cfg.TrustedProxies) > 0 {
		r.SetTrustedProxies(cfg.TrustedProxies)
	}

	server := &Server{
		Router:            r,
		DB:                db,
		Config:            cfg,
		LicenseStore:      ls,
		ProductStore:      ps,
		ProductGroupStore: pgs,
		ReleaseStore:      rs,
		FeatureStore:      fs,
		LogStore:          logs,
		StatsStore:        ss,
	}

	server.setupRoutes()
	return server
}

func (s *Server) setupRoutes() {
	// Initialize Rate Limiters
	adminRateLimiter := middleware.RateLimitMiddleware(s.Config.RateLimitAdmin)
	checkRateLimiter := middleware.RateLimitMiddleware(s.Config.RateLimitCheck)

	// Public routes
	s.Router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// License Key Public Endpoints
	s.Router.GET("/check", checkRateLimiter, handlers.CheckLicenseHandler(s.LicenseStore, s.ProductStore, s.Config.ResponseSigningPrivateKey, s.LogStore))

	// Protected routes
	authorized := s.Router.Group("/")
	authorized.Use(adminRateLimiter)
	authorized.Use(middleware.JWTAuth(s.Config))
	{
		// Dashboard Stats
		authorized.GET("/admin/stats", handlers.GetDashboardStatsHandler(s.StatsStore))

		// License Management
		authorized.GET("/admin/keys", handlers.GetLicenseHandler(s.LicenseStore))
		authorized.POST("/admin/keys", handlers.GenerateLicenseHandler(s.LicenseStore, s.ProductStore, s.ProductGroupStore, s.LogStore))
		authorized.PUT("/admin/keys", handlers.UpdateLicenseHandler(s.LicenseStore, s.LogStore))
		authorized.DELETE("/admin/keys", handlers.RevokeLicenseHandler(s.LicenseStore, s.LogStore))
		authorized.DELETE("/admin/keys/purge", handlers.DeleteLicenseHandler(s.LicenseStore, s.LogStore))

		// Product Management
		authorized.GET("/admin/products", handlers.ListProductsHandler(s.ProductStore))
		authorized.POST("/admin/products", handlers.CreateProductHandler(s.ProductStore, s.LogStore))
		authorized.GET("/admin/products/:id", handlers.GetProductHandler(s.ProductStore, s.ProductGroupStore))
		authorized.PUT("/admin/products/:id", handlers.UpdateProductHandler(s.ProductStore, s.LogStore))
		authorized.DELETE("/admin/products/:id", handlers.DeleteProductHandler(s.ProductStore, s.LogStore))

		// Product Group Management
		authorized.GET("/admin/product-groups", handlers.ListProductGroupsHandler(s.ProductGroupStore))
		authorized.POST("/admin/product-groups", handlers.CreateProductGroupHandler(s.ProductGroupStore, s.LogStore))
		authorized.GET("/admin/product-groups/:id", handlers.GetProductGroupHandler(s.ProductGroupStore))
		authorized.PUT("/admin/product-groups/:id", handlers.UpdateProductGroupHandler(s.ProductGroupStore, s.LogStore))
		authorized.DELETE("/admin/product-groups/:id", handlers.DeleteProductGroupHandler(s.ProductGroupStore, s.LogStore))


		// Feature Management
		authorized.POST("/admin/features", handlers.CreateFeatureHandler(s.FeatureStore, s.LogStore))
		authorized.GET("/admin/features", handlers.ListAllFeaturesHandler(s.FeatureStore))
		authorized.GET("/admin/features/global", handlers.ListGlobalFeaturesHandler(s.FeatureStore))
		authorized.GET("/admin/features/:id", handlers.GetFeatureHandler(s.FeatureStore))
		authorized.PUT("/admin/features/:featureId", handlers.UpdateFeatureHandler(s.FeatureStore, s.LogStore))
		authorized.DELETE("/admin/features/:featureId", handlers.DeleteFeatureHandler(s.FeatureStore, s.LogStore))

		// Release Management
		authorized.POST("/admin/releases", handlers.CreateReleaseHandler(s.ReleaseStore, s.LogStore))
		authorized.GET("/admin/releases", handlers.ListAllReleasesHandler(s.ReleaseStore))
		authorized.GET("/admin/releases/global", handlers.ListGlobalReleasesHandler(s.ReleaseStore))
		authorized.GET("/admin/releases/:id", handlers.GetReleaseHandler(s.ReleaseStore))
		authorized.PUT("/admin/releases/:releaseId", handlers.UpdateReleaseHandler(s.ReleaseStore, s.LogStore))
		authorized.DELETE("/admin/releases/:releaseId", handlers.DeleteReleaseHandler(s.ReleaseStore, s.LogStore))

		// Log Management
		authorized.GET("/admin/logs/license-checks", handlers.GetLicenseCheckLogsHandler(s.LogStore))
		authorized.GET("/admin/logs/admin-actions", handlers.GetAdminLogsHandler(s.LogStore))

	}
}
