package storage

import (
    "context"
    "io"
    "mime/multipart"
    "time"

    "github.com/google/uuid"
)

// StorageService defines the interface for file storage operations
// ...existing code...
type StorageService interface {
    Upload(ctx context.Context, file multipart.File, header *multipart.FileHeader, opts *UploadOptions) (*UploadResult, error)
    UploadFromReader(ctx context.Context, reader io.Reader, filename string, contentType string, size int64, opts *UploadOptions) (*UploadResult, error)
    Download(ctx context.Context, storageKey string) ([]byte, error)
    DownloadToWriter(ctx context.Context, storageKey string, writer io.Writer) error
    Delete(ctx context.Context, storageKey string) error
    DeleteMultiple(ctx context.Context, storageKeys []string) error
    GeneratePresignedURL(ctx context.Context, storageKey string, expiry time.Duration) (string, error)
    GenerateThumbnail(ctx context.Context, storageKey string, width, height int) (*UploadResult, error)
    GetMetadata(ctx context.Context, storageKey string) (*FileMetadata, error)
    Exists(ctx context.Context, storageKey string) (bool, error)
    Copy(ctx context.Context, sourceKey, destKey string) error
    Move(ctx context.Context, sourceKey, destKey string) error
    ListFiles(ctx context.Context, prefix string, limit int) ([]*FileInfo, error)
    GetStorageURL(storageKey string) string
}

// UploadOptions contains options for file upload
// ...existing code...
type UploadOptions struct {
    Folder string
    GenerateThumbnail bool
    ThumbnailWidth  int
    ThumbnailHeight int
    CustomFilename string
    ContentType string
    UserID uuid.UUID
    Metadata map[string]string
    ACL string
    CacheControl string
}

// UploadResult contains the result of a file upload
// ...existing code...
type UploadResult struct {
    StorageKey   string
    StoragePath  string
    URL          string
    ThumbnailURL string
    FileName     string
    FileSize     int64
    ContentType  string
    Width        int
    Height       int
    Metadata     map[string]string
}

// FileMetadata contains metadata about a stored file
// ...existing code...
type FileMetadata struct {
    StorageKey   string
    FileName     string
    FileSize     int64
    ContentType  string
    LastModified time.Time
    ETag         string
    Metadata     map[string]string
}

// FileInfo contains basic information about a file
// ...existing code...
type FileInfo struct {
    StorageKey   string
    FileName     string
    FileSize     int64
    LastModified time.Time
    IsDirectory  bool
}

const (
    DefaultThumbnailWidth  = 300
    DefaultThumbnailHeight = 300
    DefaultACL             = "private"
    DefaultCacheControl    = "max-age=31536000"
)

// NewUploadOptions creates default upload options
func NewUploadOptions() *UploadOptions {
    return &UploadOptions{
        GenerateThumbnail: false,
        ThumbnailWidth:    DefaultThumbnailWidth,
        ThumbnailHeight:   DefaultThumbnailHeight,
        ACL:               DefaultACL,
        CacheControl:      DefaultCacheControl,
        Metadata:          make(map[string]string),
    }
}
