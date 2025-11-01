package handlers

import (
    "context"
    "fmt"
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    "github.com/rs/zerolog/log"
    
    "ugc-platform/internal/database/repository"
    "ugc-platform/internal/models"
    "ugc-platform/internal/services/ai"
    "ugc-platform/internal/services/websocket"
)

// VideoHandler handles video generation requests
type VideoHandler struct {
    videoRepo      *repository.VideoRepository
    conversationRepo *repository.ConversationRepository
    videoGenerator *ai.VideoGenerator
    hub            *websocket.Hub
}

// NewVideoHandler creates a new video handler
func NewVideoHandler(
    videoRepo *repository.VideoRepository,
    conversationRepo *repository.ConversationRepository,
    videoGenerator *ai.VideoGenerator,
    hub *websocket.Hub,
) *VideoHandler {
    return &VideoHandler{
        videoRepo:        videoRepo,
        conversationRepo: conversationRepo,
        videoGenerator:   videoGenerator,
        hub:              hub,
    }
}

// GenerateVideo initiates video generation
// POST /api/v1/videos/generate
func (h *VideoHandler) GenerateVideo(c *gin.Context) {
    userID := c.MustGet("user_id").(uuid.UUID)

    var input models.GenerateVideoInput
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": err.Error(),
        })
        return
    }

    log.Info().
        Str("user_id", userID.String()).
        Str("conversation_id", input.ConversationID.String()).
        Str("product_name", input.ProductName).
        Msg("Video generation request received")

    // Verify conversation exists and belongs to user
    conversation, err := h.conversationRepo.FindByID(c.Request.Context(), input.ConversationID)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{
            "error": "Conversation not found",
        })
        return
    }

    if conversation.UserID != userID {
        c.JSON(http.StatusForbidden, gin.H{
            "error": "Access denied",
        })
        return
    }

    // Create video record
    video := &models.Video{
        ConversationID: input.ConversationID,
        UserID:         userID,
        Status:         models.VideoStatusQueued,
        Progress:       0,
        ProductName:    input.ProductName,
        ProductDesc:    input.ProductDesc,
        CharacterType:  input.CharacterType,
    }

    if input.Duration == 0 {
        input.Duration = 30 // Default 30 seconds
    }

    // Save video record
    if err := h.videoRepo.Create(c.Request.Context(), video); err != nil {
        log.Error().Err(err).Msg("Failed to create video record")
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": "Failed to create video record",
        })
        return
    }

    // Start video generation in background
    go h.processVideoGeneration(context.Background(), video, &input)

    log.Info().
        Str("video_id", video.ID.String()).
        Str("user_id", userID.String()).
        Msg("Video generation initiated")

    c.JSON(http.StatusAccepted, gin.H{
        "success": true,
        "data":    video.ToResponse(false),
        "message": "Video generation started",
    })
}

// processVideoGeneration handles the video generation process
func (h *VideoHandler) processVideoGeneration(ctx context.Context, video *models.Video, input *models.GenerateVideoInput) {
    // Mark as started
    video.MarkStarted()
    h.videoRepo.Update(ctx, video)

    // Send initial progress update via WebSocket
    h.sendProgressUpdate(video)

    // Build generation request
    req := &ai.VideoGenerationRequest{
        ProductName:     input.ProductName,
        ProductDesc:     input.ProductDesc,
        ProductImageURL: input.ProductImageURL,
        CharacterType:   input.CharacterType,
        Duration:        input.Duration,
        AspectRatio:     "16:9",
        Resolution:      "1080p",
        VoiceType:       "neutral",
    }

    // Generate video
    if err := h.videoGenerator.GenerateVideo(ctx, video, req); err != nil {
        log.Error().Err(err).Str("video_id", video.ID.String()).Msg("Failed to generate video")
        video.MarkFailed(fmt.Sprintf("Failed to start generation: %v", err))
        h.videoRepo.Update(ctx, video)
        h.sendProgressUpdate(video)
        return
    }

    // Save updated video
    h.videoRepo.Update(ctx, video)
    h.sendProgressUpdate(video)

    // Monitor video generation progress
    updateCallback := func(v *models.Video) error {
        if err := h.videoRepo.Update(ctx, v); err != nil {
            return err
        }
        h.sendProgressUpdate(v)
        return nil
    }

    if err := h.videoGenerator.MonitorVideoGeneration(ctx, video, updateCallback); err != nil {
        log.Error().Err(err).Str("video_id", video.ID.String()).Msg("Error monitoring video generation")
    }

    log.Info().
        Str("video_id", video.ID.String()).
        Str("status", video.Status).
        Msg("Video generation completed")
}

// sendProgressUpdate sends video progress update via WebSocket
func (h *VideoHandler) sendProgressUpdate(video *models.Video) {
    message := models.NewWebSocketMessage("video_progress", video.ConversationID, uuid.Nil)
    message.Metadata = map[string]interface{}{
        "video_id":  video.ID.String(),
        "status":    video.Status,
        "progress":  video.Progress,
        "video":     video.ToResponse(false),
    }

    h.hub.BroadcastToConversation(video.ConversationID, message, nil)
}

// GetVideo retrieves video details
// GET /api/v1/videos/:id
func (h *VideoHandler) GetVideo(c *gin.Context) {
    userID := c.MustGet("user_id").(uuid.UUID)
    videoID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "Invalid video ID",
        })
        return
    }

    video, err := h.videoRepo.FindByID(c.Request.Context(), videoID)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{
            "error": "Video not found",
        })
        return
    }

    // Check ownership
    if video.UserID != userID {
        c.JSON(http.StatusForbidden, gin.H{
            "error": "Access denied",
        })
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "data":    video.ToResponse(true),
    })
}

// GetVideoStatus retrieves video generation status
// GET /api/v1/videos/:id/status
func (h *VideoHandler) GetVideoStatus(c *gin.Context) {
    userID := c.MustGet("user_id").(uuid.UUID)
    videoID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "Invalid video ID",
        })
        return
    }

    video, err := h.videoRepo.FindByID(c.Request.Context(), videoID)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{
            "error": "Video not found",
        })
        return
    }

    // Check ownership
    if video.UserID != userID {
        c.JSON(http.StatusForbidden, gin.H{
            "error": "Access denied",
        })
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "data": gin.H{
            "video_id": video.ID,
            "status":   video.Status,
            "progress": video.Progress,
            "url":      video.URL,
            "error":    video.ErrorMessage,
        },
    })
}

// ListUserVideos lists all videos for a user
// GET /api/v1/videos
func (h *VideoHandler) ListUserVideos(c *gin.Context) {
    userID := c.MustGet("user_id").(uuid.UUID)

    // Parse query parameters
    conversationID := c.Query("conversation_id")
    status := c.Query("status")

    var videos []*models.Video
    var err error

    if conversationID != "" {
        convID, err := uuid.Parse(conversationID)
        if err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid conversation ID"})
            return
        }
        videos, err = h.videoRepo.FindByConversationID(c.Request.Context(), convID)
    } else if status != "" {
        videos, err = h.videoRepo.FindByUserIDAndStatus(c.Request.Context(), userID, status)
    } else {
        videos, err = h.videoRepo.FindByUserID(c.Request.Context(), userID)
    }

    if err != nil {
        log.Error().Err(err).Msg("Failed to list videos")
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": "Failed to retrieve videos",
        })
        return
    }

    response := make([]models.VideoResponse, len(videos))
    for i, video := range videos {
        response[i] = *video.ToResponse(false)
    }

    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "data":    response,
        "count":   len(response),
    })
}

// DeleteVideo deletes a video
// DELETE /api/v1/videos/:id
func (h *VideoHandler) DeleteVideo(c *gin.Context) {
    userID := c.MustGet("user_id").(uuid.UUID)
    videoID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "Invalid video ID",
        })
        return
    }

    video, err := h.videoRepo.FindByID(c.Request.Context(), videoID)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{
            "error": "Video not found",
        })
        return
    }

    // Check ownership
    if video.UserID != userID {
        c.JSON(http.StatusForbidden, gin.H{
            "error": "Access denied",
        })
        return
    }

    // Cancel generation if in progress
    if video.IsProcessing() && video.ExternalJobID != "" {
        if err := h.videoGenerator.CancelVideoGeneration(c.Request.Context(), video.ExternalJobID); err != nil {
            log.Warn().Err(err).Msg("Failed to cancel video generation")
        }
    }

    // Delete from database
    if err := h.videoRepo.Delete(c.Request.Context(), videoID); err != nil {
        log.Error().Err(err).Msg("Failed to delete video")
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": "Failed to delete video",
        })
        return
    }

    log.Info().
        Str("video_id", videoID.String()).
        Str("user_id", userID.String()).
        Msg("Video deleted")

    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "message": "Video deleted successfully",
    })
}

// RetryVideoGeneration retries failed video generation
// POST /api/v1/videos/:id/retry
func (h *VideoHandler) RetryVideoGeneration(c *gin.Context) {
    userID := c.MustGet("user_id").(uuid.UUID)
    videoID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "Invalid video ID",
        })
        return
    }

    video, err := h.videoRepo.FindByID(c.Request.Context(), videoID)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{
            "error": "Video not found",
        })
        return
    }

    // Check ownership
    if video.UserID != userID {
        c.JSON(http.StatusForbidden, gin.H{
            "error": "Access denied",
        })
        return
    }

    // Only allow retry for failed videos
    if !video.IsFailed() {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "Video is not in failed state",
        })
        return
    }

    // Reset video status
    video.Status = models.VideoStatusQueued
    video.Progress = 0
    video.ErrorMessage = ""
    video.ExternalJobID = ""

    if err := h.videoRepo.Update(c.Request.Context(), video); err != nil {
        log.Error().Err(err).Msg("Failed to update video")
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": "Failed to retry video generation",
        })
        return
    }

    // Restart generation
    input := &models.GenerateVideoInput{
        ConversationID:  video.ConversationID,
        ProductName:     video.ProductName,
        ProductDesc:     video.ProductDesc,
        CharacterType:   video.CharacterType,
        Duration:        int(video.Duration),
    }

    go h.processVideoGeneration(context.Background(), video, input)

    log.Info().
        Str("video_id", videoID.String()).
        Msg("Video generation retry initiated")

    c.JSON(http.StatusAccepted, gin.H{
        "success": true,
        "data":    video.ToResponse(false),
        "message": "Video generation retry started",
    })
}
