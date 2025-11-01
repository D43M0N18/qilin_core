package main

import (
    "context"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/joho/godotenv"
    "github.com/rs/zerolog"
    
    "github.com/D43M0N18/qilin_core/internal/api/routes"
    "github.com/D43M0N18/qilin_core/internal/config"
    "github.com/D43M0N18/qilin_core/internal/database"
    "github.com/D43M0N18/qilin_core/internal/services/websocket"
    "github.com/D43M0N18/qilin_core/internal/services/ai"
    "github.com/D43M0N18/qilin_core/internal/services/storage"
)

func main() {
    // 1. Initialize logger with structured logging
    logger := zerolog.New(os.Stdout).With().
        Timestamp().
        Caller().
        Logger()

    // 2. Load environment variables
    if err := godotenv.Load(); err != nil {
        logger.Warn().Msg("No .env file found, using system environment variables")
    }

    // 3. Load configuration
    cfg, err := config.Load()
    if err != nil {
        logger.Fatal().Err(err).Msg("Failed to load configuration")
    }

    // 4. Initialize database connection with retry logic
    db, err := database.NewPostgresConnection(cfg.Database)
    if err != nil {
        logger.Fatal().Err(err).Msg("Failed to connect to database")
    }
    defer db.Close()

    // 5. Run database migrations
    if err := database.RunMigrations(db); err != nil {
        logger.Fatal().Err(err).Msg("Failed to run migrations")
    }

    // 6. Initialize Redis connection
    redisClient := database.NewRedisClient(cfg.Redis)
    defer redisClient.Close()

    // 7. Initialize services
    storageService := storage.NewS3Service(cfg.Storage)
    aiService := ai.NewClaudeClient(cfg.AI.AnthropicAPIKey)
    
    // 8. Initialize WebSocket hub
    wsHub := websocket.NewHub()
    go wsHub.Run()

    // 9. Set Gin mode based on environment
    if cfg.Server.Environment == "production" {
        gin.SetMode(gin.ReleaseMode)
    }

    // 10. Initialize Gin router
    router := gin.New()
    router.Use(gin.Recovery())
    
    // 11. Setup routes
    routes.SetupRoutes(router, cfg, db, redisClient, storageService, aiService, wsHub)

    // 12. Create HTTP server with timeouts
    srv := &http.Server{
        Addr:           ":" + cfg.Server.Port,
        Handler:        router,
        ReadTimeout:    15 * time.Second,
        WriteTimeout:   15 * time.Second,
        IdleTimeout:    60 * time.Second,
        MaxHeaderBytes: 1 << 20, // 1 MB
    }

    // 13. Start server in goroutine
    go func() {
        logger.Info().
            Str("port", cfg.Server.Port).
            Str("environment", cfg.Server.Environment).
            Msg("Server starting")
        
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            logger.Fatal().Err(err).Msg("Failed to start server")
        }
    }()

    // 14. Graceful shutdown
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    logger.Info().Msg("Server shutting down...")

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    if err := srv.Shutdown(ctx); err != nil {
        logger.Fatal().Err(err).Msg("Server forced to shutdown")
    }

    logger.Info().Msg("Server exited")
}
