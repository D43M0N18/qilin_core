package handlers

import (
    "fmt"
    "image"
    _ "image/gif"
    _ "image/jpeg"
    _ "image/png"
    "net/http"
    "path/filepath"
    "strings"
    "time"
    "mime/multipart"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    "github.com/rs/zerolog/log"
    
    "ugc-platform/internal/config"
    "ugc-platform/internal/database/repository"
    "ugc-platform/internal/models"
    "ugc-platform/internal/services/storage"
)

// UploadHandler handles file upload operations
// ...existing code...
type UploadHandler struct {
    attachmentRepo *repository.AttachmentRepository
    storage        storage.StorageService
    config         *config.Config
}

func NewUploadHandler(attachmentRepo *repository.AttachmentRepository, storage storage.StorageService, cfg *config.Config) *UploadHandler {
    return &UploadHandler{
        attachmentRepo: attachmentRepo,
        storage:        storage,
        config:         cfg,
    }
}

func (h *UploadHandler) UploadFile(c *gin.Context) {
    userID := c.MustGet("user_id").(uuid.UUID)
    if err := c.Request.ParseMultipartForm(h.config.Upload.MaxFileSize); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "File too large or invalid form data"})
        return
    }
    file, header, err := c.Request.FormFile("file")
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
        return
    }
    defer file.Close()
    var conversationID *uuid.UUID
    if convID := c.PostForm("conversation_id"); convID != "" {
        if id, err := uuid.Parse(convID); err == nil {
            conversationID = &id
        }
    }
    var messageID *uuid.UUID
    if msgID := c.PostForm("message_id"); msgID != "" {
        if id, err := uuid.Parse(msgID); err == nil {
            messageID = &id
        }
    }
    log.Info().Str("user_id", userID.String()).Str("filename", header.Filename).Int64("size", header.Size).Msg("File upload started")
    if err := h.validateFile(header); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    fileType := h.determineFileType(header.Filename)
    opts := storage.NewUploadOptions()
    opts.Folder = "uploads"
    opts.UserID = userID
    opts.Metadata = map[string]string{
        "user_id":      userID.String(),
        "original_name": header.Filename,
    }
    if conversationID != nil {
        opts.Metadata["conversation_id"] = conversationID.String()
    }
    if strings.HasPrefix(fileType, "image/") {
        opts.GenerateThumbnail = true
        opts.ThumbnailWidth = 300
        opts.ThumbnailHeight = 300
    }
    result, err := h.storage.Upload(c.Request.Context(), file, header, opts)
    if err != nil {
        log.Error().Err(err).Msg("Failed to upload file")
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload file"})
        return
    }
    var width, height int
    if strings.HasPrefix(fileType, "image/") {
        if _, err := file.Seek(0, 0); err == nil {
            if img, _, err := image.DecodeConfig(file); err == nil {
                width = img.Width
                height = img.Height
            }
        }
    }
    attachment := &models.Attachment{
        UserID:       userID,
        FileName:     result.FileName,
        OriginalName: header.Filename,
        FileType:     fileType,
        FileSize:     header.Size,
        Width:        width,
        Height:       height,
        StorageKey:   result.StorageKey,
        StoragePath:  result.StoragePath,
        URL:          result.URL,
        ThumbnailURL: result.ThumbnailURL,
        Status:       "uploaded",
    }
    if messageID != nil {
        attachment.MessageID = *messageID
    } else {
        attachment.MessageID = uuid.New()
    }
    if err := h.attachmentRepo.Create(c.Request.Context(), attachment); err != nil {
        log.Error().Err(err).Msg("Failed to save attachment")
    }
    log.Info().Str("attachment_id", attachment.ID.String()).Str("storage_key", result.StorageKey).Int64("size", header.Size).Msg("File uploaded successfully")
    c.JSON(http.StatusOK, gin.H{"success": true, "data": attachment.ToResponse()})
}

func (h *UploadHandler) UploadMultiple(c *gin.Context) {
    userID := c.MustGet("user_id").(uuid.UUID)
    if err := c.Request.ParseMultipartForm(h.config.Upload.MaxFileSize * 5); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Files too large or invalid form data"})
        return
    }
    form := c.Request.MultipartForm
    files := form.File["files"]
    if len(files) == 0 {
        c.JSON(http.StatusBadRequest, gin.H{"error": "No files uploaded"})
        return
    }
    if len(files) > 10 {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Maximum 10 files allowed per request"})
        return
    }
    var conversationID *uuid.UUID
    if convID := c.PostForm("conversation_id"); convID != "" {
        if id, err := uuid.Parse(convID); err == nil {
            conversationID = &id
        }
    }
    var uploadedFiles []models.AttachmentResponse
    var errors []string
    for _, fileHeader := range files {
        file, err := fileHeader.Open()
        if err != nil {
            errors = append(errors, fmt.Sprintf("%s: failed to open", fileHeader.Filename))
            continue
        }
        if err := h.validateFile(fileHeader); err != nil {
            file.Close()
            errors = append(errors, fmt.Sprintf("%s: %v", fileHeader.Filename, err))
            continue
        }
        opts := storage.NewUploadOptions()
        opts.Folder = "uploads"
        opts.UserID = userID
        opts.GenerateThumbnail = true
        result, err := h.storage.Upload(c.Request.Context(), file, fileHeader, opts)
        file.Close()
        if err != nil {
            errors = append(errors, fmt.Sprintf("%s: upload failed", fileHeader.Filename))
            continue
        }
        attachment := &models.Attachment{
            MessageID:    uuid.New(),
            UserID:       userID,
            FileName:     result.FileName,
            OriginalName: fileHeader.Filename,
            FileType:     h.determineFileType(fileHeader.Filename),
            FileSize:     fileHeader.Size,
            StorageKey:   result.StorageKey,
            StoragePath:  result.StoragePath,
            URL:          result.URL,
            ThumbnailURL: result.ThumbnailURL,
            Status:       "uploaded",
        }
        if err := h.attachmentRepo.Create(c.Request.Context(), attachment); err == nil {
            uploadedFiles = append(uploadedFiles, *attachment.ToResponse())
        }
    }
    response := gin.H{"success": len(uploadedFiles) > 0, "data": uploadedFiles, "count": len(uploadedFiles)}
    if len(errors) > 0 {
        response["errors"] = errors
    }
    c.JSON(http.StatusOK, response)
}

func (h *UploadHandler) GetAttachment(c *gin.Context) {
    userID := c.MustGet("user_id").(uuid.UUID)
    attachmentID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid attachment ID"})
        return
    }
    attachment, err := h.attachmentRepo.FindByID(c.Request.Context(), attachmentID)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Attachment not found"})
        return
    }
    if attachment.UserID != userID {
        c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
        return
    }
    c.JSON(http.StatusOK, gin.H{"success": true, "data": attachment.ToResponse()})
}

func (h *UploadHandler) DeleteAttachment(c *gin.Context) {
    userID := c.MustGet("user_id").(uuid.UUID)
    attachmentID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid attachment ID"})
        return
    }
    attachment, err := h.attachmentRepo.FindByID(c.Request.Context(), attachmentID)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Attachment not found"})
        return
    }
    if attachment.UserID != userID {
        c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
        return
    }
    if err := h.storage.Delete(c.Request.Context(), attachment.StorageKey); err != nil {
        log.Warn().Err(err).Msg("Failed to delete from storage")
    }
    if attachment.ThumbnailURL != "" {
        thumbnailKey := strings.Replace(attachment.StorageKey, filepath.Ext(attachment.StorageKey), "_thumb"+filepath.Ext(attachment.StorageKey), 1)
        h.storage.Delete(c.Request.Context(), thumbnailKey)
    }
    if err := h.attachmentRepo.Delete(c.Request.Context(), attachmentID); err != nil {
        log.Error().Err(err).Msg("Failed to delete attachment")
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete attachment"})
        return
    }
    log.Info().Str("attachment_id", attachmentID.String()).Str("user_id", userID.String()).Msg("Attachment deleted")
    c.JSON(http.StatusOK, gin.H{"success": true, "message": "Attachment deleted successfully"})
}

func (h *UploadHandler) GeneratePresignedURL(c *gin.Context) {
    userID := c.MustGet("user_id").(uuid.UUID)
    attachmentID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid attachment ID"})
        return
    }
    attachment, err := h.attachmentRepo.FindByID(c.Request.Context(), attachmentID)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Attachment not found"})
        return
    }
    if attachment.UserID != userID {
        c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
        return
    }
    url, err := h.storage.GeneratePresignedURL(c.Request.Context(), attachment.StorageKey, 1*time.Hour)
    if err != nil {
        log.Error().Err(err).Msg("Failed to generate presigned URL")
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate download URL"})
        return
    }
    c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"url": url, "expires_in": 3600}})
}

func (h *UploadHandler) validateFile(header *multipart.FileHeader) error {
    if header.Size > h.config.Upload.MaxFileSize {
        return fmt.Errorf("file size exceeds maximum allowed size of %d bytes", h.config.Upload.MaxFileSize)
    }
    if header.Size == 0 {
        return fmt.Errorf("file is empty")
    }
    ext := strings.ToLower(filepath.Ext(header.Filename))
    allowedExts := append(h.config.Upload.AllowedImageExts, h.config.Upload.AllowedVideoExts...)
    isAllowed := false
    for _, allowedExt := range allowedExts {
        if ext == allowedExt {
            isAllowed = true
            break
        }
    }
    if !isAllowed {
        return fmt.Errorf("file type %s is not allowed", ext)
    }
    return nil
}

func (h *UploadHandler) determineFileType(filename string) string {
    ext := strings.ToLower(filepath.Ext(filename))
    mimeTypes := map[string]string{
        ".jpg":  "image/jpeg",
        ".jpeg": "image/jpeg",
        ".png":  "image/png",
        ".gif":  "image/gif",
        ".webp": "image/webp",
        ".mp4":  "video/mp4",
        ".mov":  "video/quicktime",
        ".avi":  "video/x-msvideo",
        ".webm": "video/webm",
        ".pdf":  "application/pdf",
    }
    if mimeType, ok := mimeTypes[ext]; ok {
        return mimeType
    }
    return "application/octet-stream"
}
