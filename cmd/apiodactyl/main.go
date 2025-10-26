package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/thebearodactyl/apiodactyl/internal/config"
	"github.com/thebearodactyl/apiodactyl/internal/database"
	"github.com/thebearodactyl/apiodactyl/internal/handlers"
	"github.com/thebearodactyl/apiodactyl/internal/middleware"
	"github.com/thebearodactyl/apiodactyl/internal/utils"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	db, err := database.InitDB(cfg.Database.Path)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	router := setupRouter(db, cfg)
	router.MaxMultipartMemory = 16 << 20

	server := &http.Server{
		Addr:         ":" + cfg.App.Port, // e.g., 18081
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Printf("Starting HTTP server on port %s (behind nginx TLS)", cfg.App.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

func setupRouter(db *database.DB, cfg *config.Config) *gin.Engine {
	router := gin.Default()

	router.Use(middleware.RequestLogger())
	router.Use(gin.Recovery())

	router.Use(func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		clientHeader := c.GetHeader("X-Bearodactyl-Client")

		allow := false

		if clientHeader != "" {
			lowerClient := strings.ToLower(clientHeader)
			if strings.Contains(lowerClient, "bearodactyl") {
				allow = true
			} else {
				wellKnownClients := []string{"mozilla", "chrome", "safari", "edge", "postman"}
				for _, client := range wellKnownClients {
					if strings.Contains(lowerClient, client) {
						allow = true
						break
					}
				}
			}
		}

		if allow {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization, X-Bearodactyl-Client")
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	})

	router.NoRoute(handlers.NotFound)

	h := handlers.NewHandler(db)
	authHandler := handlers.NewAuthHandler(db, cfg.JWT.Secret, cfg.JWT.ExpirationHours)
	gamesHandler := handlers.NewGameHandler(db)
	booksHandler := handlers.NewBookHandler(db)
	commentsHandler := handlers.NewCommentHandler(db)
	routeHandler := handlers.NewRouteHandler()

	router.Static("/files", "./files")

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":      "ok",
			"environment": cfg.App.Environment,
		})
	})

	router.GET("/api/v1/routes", routeHandler.GetAllRoutes)

	public := router.Group("/api/v1")
	{
		public.POST("/auth/register", authHandler.Register)
		public.POST("/auth/login", authHandler.Login)
	}

	protected := router.Group("/api/v1")
	protected.Use(middleware.JWTAuth(cfg.JWT.Secret))
	{
		protected.GET("/me", authHandler.GetProfile)
		protected.POST("/upload", middleware.RequireAdmin(), utils.UploadFile)

		resources := protected.Group("/resources")
		{
			resources.GET("/routes", routeHandler.GetResourcesRoutes)
			resources.GET("", h.GetResources)
			resources.GET("/:id", h.GetResource)
			resources.POST("", middleware.RequireAdmin(), h.CreateResource)
			resources.PUT("/:id", middleware.RequireAdmin(), h.UpdateResource)
			resources.DELETE("/:id", middleware.RequireAdmin(), h.DeleteResource)
		}

		games := protected.Group("/games")
		{
			games.GET("/routes", routeHandler.GetGamesRoutes)
			games.GET("", gamesHandler.GetGames)
			games.GET("/search", gamesHandler.SearchGames)
			games.GET("/:id", gamesHandler.GetGame)
			games.POST("", middleware.RequireAdmin(), gamesHandler.CreateGame)
			games.PUT("/:id", middleware.RequireAdmin(), gamesHandler.UpdateGame)
			games.DELETE("/:id", middleware.RequireAdmin(), gamesHandler.DeleteGame)
		}

		books := protected.Group("/books")
		{
			books.GET("/routes", routeHandler.GetBooksRoutes)
			books.GET("", booksHandler.GetBooks)
			books.GET("/search", booksHandler.SearchBooks)
			books.GET("/:id", booksHandler.GetBook)
			books.POST("", middleware.RequireAdmin(), booksHandler.CreateBook)
			books.PUT("/:id", middleware.RequireAdmin(), booksHandler.UpdateBook)
			books.DELETE("/:id", middleware.RequireAdmin(), booksHandler.DeleteBook)
		}

		comments := protected.Group("/comments")
		{
			comments.GET("/routes", routeHandler.GetCommentsRoutes)
			comments.POST("", commentsHandler.CreateComment)
			comments.GET("", commentsHandler.GetComments)
			comments.PUT("/:id", commentsHandler.UpdateComment)
			comments.DELETE("/:id", commentsHandler.DeleteComment)
		}
	}

	return router
}
