package storage

import (
    "bytes"
    "context"
    "fmt"
    "image"
    _ "image/gif"
    _ "image/jpeg"
    _ "image/png"
    "io"
    "mime/multipart"
    "path"
    "path/filepath"
    "strings"
    "time"

    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/credentials"
    "github.com/aws/aws-sdk-go-v2/service/s3"
    "github.com/aws/aws-sdk-go-v2/service/s3/types"
    "github.com/disintegration/imaging"
    "github.com/google/uuid"
    "github.com/rs/zerolog/log"
    appconfig "github.com/D43M0N18/qilin_core/internal/config"
)

// S3Service implements StorageService using AWS S3 or MinIO
// ...existing code...
type S3Service struct {
    client     *s3.Client
    bucket     string
    region     string
    baseURL    string
    publicURL  string
    endpoint   string // For MinIO
}

// NewS3Service creates a new S3 storage service
func NewS3Service(cfg appconfig.StorageConfig) (*S3Service, error) {
    ctx := context.Background()
    var awsCfg aws.Config
    var err error
    if cfg.Endpoint != "" {
        customResolver := aws.EndpointResolverWithOptionsFunc(
            func(service, region string, options ...interface{}) (aws.Endpoint, error) {
                return aws.Endpoint{
                    URL:               cfg.Endpoint,
                    SigningRegion:     cfg.Region,
                    HostnameImmutable: true,
                }, nil
            })
        awsCfg, err = config.LoadDefaultConfig(ctx,
            config.WithRegion(cfg.Region),
            config.WithCredentialsProvider(
                credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
            ),
            config.WithEndpointResolverWithOptions(customResolver),
        )
    } else {
        awsCfg, err = config.LoadDefaultConfig(ctx,
            config.WithRegion(cfg.Region),
            config.WithCredentialsProvider(
                credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
            ),
        )
    }
    if err != nil {
        return nil, fmt.Errorf("failed to load AWS config: %w", err)
    }
    client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
        if cfg.Endpoint != "" {
            o.BaseEndpoint = aws.String(cfg.Endpoint)
            o.UsePathStyle = true
        }
    })
    publicURL := cfg.Endpoint
    if publicURL == "" {
        publicURL = fmt.Sprintf("https://%s.s3.%s.amazonaws.com", cfg.Bucket, cfg.Region)
    }
    service := &S3Service{
        client:    client,
        bucket:    cfg.Bucket,
        region:    cfg.Region,
        baseURL:   publicURL,
        publicURL: publicURL,
        endpoint:  cfg.Endpoint,
    }
    if err := service.verifyBucket(ctx); err != nil {
        return nil, fmt.Errorf("failed to verify bucket: %w", err)
    }
    log.Info().Str("bucket", cfg.Bucket).Str("region", cfg.Region).Str("endpoint", cfg.Endpoint).Msg("S3 storage service initialized")
    return service, nil
}

func (s *S3Service) verifyBucket(ctx context.Context) error {
    _, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{
        Bucket: aws.String(s.bucket),
    })
    if err != nil {
        return fmt.Errorf("bucket %s not accessible: %w", s.bucket, err)
    }
    return nil
}

// ...existing code...
// (The rest of the S3Service methods as specified in your instructions)
