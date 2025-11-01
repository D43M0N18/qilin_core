package config

import (
    "fmt"
    "os"
    "strconv"
    "time"
)

type Config struct {
    Server   ServerConfig
    Database DatabaseConfig
    Redis    RedisConfig
    Storage  StorageConfig
    AI       AIConfig
    JWT      JWTConfig
    Upload   UploadConfig
}

type ServerConfig struct {
    Port        string
    Environment string // development, staging, production
    BaseURL     string
}

type DatabaseConfig struct {
    Host            string
    Port            string
    User            string
    Password        string
    DBName          string
    SSLMode         string
    MaxOpenConns    int
    MaxIdleConns    int
    ConnMaxLifetime time.Duration
}

type RedisConfig struct {
    Host     string
    Port     string
    Password string
    DB       int
}

type StorageConfig struct {
    Provider      string // s3, minio, local
    Bucket        string
    Region        string
    Endpoint      string // For MinIO
    AccessKey     string
    SecretKey     string
    MaxUploadSize int64 // in bytes
}

type AIConfig struct {
    AnthropicAPIKey string
    VideoGenAPIKey  string
    VideoGenAPIURL  string
    MaxTokens       int
    Temperature     float64
}

type JWTConfig struct {
    Secret               string
    AccessTokenDuration  time.Duration
    RefreshTokenDuration time.Duration
}

type UploadConfig struct {
    MaxFileSize      int64 // in bytes
    AllowedImageExts []string
    AllowedVideoExts []string
    TempDir          string
}

func Load() (*Config, error) {
    cfg := &Config{
        Server: ServerConfig{
            Port:        getEnv("SERVER_PORT", "8080"),
            Environment: getEnv("ENVIRONMENT", "development"),
            BaseURL:     getEnv("BASE_URL", "http://localhost:8080"),
        },
        Database: DatabaseConfig{
            Host:            getEnv("DB_HOST", "localhost"),
            Port:            getEnv("DB_PORT", "5432"),
            User:            getEnv("DB_USER", "postgres"),
            Password:        getEnv("DB_PASSWORD", ""),
            DBName:          getEnv("DB_NAME", "qilin_ugc"),
            SSLMode:         getEnv("DB_SSLMODE", "disable"),
            MaxOpenConns:    getEnvInt("DB_MAX_OPEN_CONNS", 25),
            MaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNS", 5),
            ConnMaxLifetime: time.Duration(getEnvInt("DB_CONN_MAX_LIFETIME", 300)) * time.Second,
        },
        Redis: RedisConfig{
            Host:     getEnv("REDIS_HOST", "localhost"),
            Port:     getEnv("REDIS_PORT", "6379"),
            Password: getEnv("REDIS_PASSWORD", ""),
            DB:       getEnvInt("REDIS_DB", 0),
        },
        Storage: StorageConfig{
            Provider:      getEnv("STORAGE_PROVIDER", "local"),
            Bucket:        getEnv("STORAGE_BUCKET", "qilin-uploads"),
            Region:        getEnv("STORAGE_REGION", "us-east-1"),
            Endpoint:      getEnv("STORAGE_ENDPOINT", ""),
            AccessKey:     getEnv("STORAGE_ACCESS_KEY", ""),
            SecretKey:     getEnv("STORAGE_SECRET_KEY", ""),
            MaxUploadSize: int64(getEnvInt("MAX_UPLOAD_SIZE", 100*1024*1024)), // 100MB default
        },
        AI: AIConfig{
            AnthropicAPIKey: getEnv("ANTHROPIC_API_KEY", ""),
            VideoGenAPIKey:  getEnv("VIDEOGEN_API_KEY", ""),
            VideoGenAPIURL:  getEnv("VIDEOGEN_API_URL", ""),
            MaxTokens:       getEnvInt("AI_MAX_TOKENS", 4096),
            Temperature:     getEnvFloat("AI_TEMPERATURE", 0.7),
        },
        JWT: JWTConfig{
            Secret:               getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
            AccessTokenDuration:  time.Duration(getEnvInt("JWT_ACCESS_DURATION", 15)) * time.Minute,
            RefreshTokenDuration: time.Duration(getEnvInt("JWT_REFRESH_DURATION", 7*24)) * time.Hour,
        },
        Upload: UploadConfig{
            MaxFileSize:      int64(getEnvInt("MAX_FILE_SIZE", 50*1024*1024)), // 50MB
            AllowedImageExts: []string{".jpg", ".jpeg", ".png", ".gif", ".webp"},
            AllowedVideoExts: []string{".mp4", ".mov", ".avi", ".webm"},
            TempDir:          getEnv("TEMP_DIR", "/tmp/qilin-uploads"),
        },
    }

    // Validate critical configuration
    if err := cfg.Validate(); err != nil {
        return nil, err
    }

    return cfg, nil
}

func (c *Config) Validate() error {
    if c.JWT.Secret == "your-secret-key-change-in-production" && c.Server.Environment == "production" {
        return fmt.Errorf("JWT secret must be changed in production")
    }

    if c.AI.AnthropicAPIKey == "" {
        return fmt.Errorf("Anthropic API key is required")
    }

    if c.Database.Password == "" && c.Server.Environment == "production" {
        return fmt.Errorf("database password is required in production")
    }

    return nil
}

func getEnv(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
    if value := os.Getenv(key); value != "" {
        if intValue, err := strconv.Atoi(value); err == nil {
            return intValue
        }
    }
    return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
    if value := os.Getenv(key); value != "" {
        if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
            return floatValue
        }
    }
    return defaultValue
}
