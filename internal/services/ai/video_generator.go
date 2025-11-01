package ai

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"

    "github.com/google/uuid"
    "github.com/rs/zerolog/log"
    "github.com/D43M0N18/qilin_core/internal/models"
    "github.com/D43M0N18/qilin_core/internal/services/storage"
)

// VideoGenerator handles video generation using external APIs
// ...existing code...
type VideoGenerator struct {
    apiKey         string
    apiURL         string
    httpClient     *http.Client
    storage        storage.StorageService
    characterSelector *CharacterSelector
}

type VideoGenerationRequest struct {
    ProductName     string
    ProductDesc     string
    ProductImageURL string
    CharacterType   string
    CharacterName   string
    Script          string
    Duration        int
    AspectRatio     string
    Resolution      string
    VoiceType       string
}

type VideoGenerationJob struct {
    JobID       string
    Status      string
    Progress    int
    VideoURL    string
    ThumbnailURL string
    ErrorMessage string
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

func NewVideoGenerator(apiKey, apiURL string, storage storage.StorageService, characterSelector *CharacterSelector) *VideoGenerator {
    return &VideoGenerator{
        apiKey:         apiKey,
        apiURL:         apiURL,
        storage:        storage,
        characterSelector: characterSelector,
        httpClient: &http.Client{
            Timeout: 30 * time.Second,
        },
    }
}

func (vg *VideoGenerator) GenerateVideo(ctx context.Context, video *models.Video, req *VideoGenerationRequest) error {
    log.Info().Str("video_id", video.ID.String()).Str("product_name", req.ProductName).Msg("Starting video generation")
    video.Status = models.VideoStatusAnalyzing
    video.Progress = 5
    var selection *CharacterSelection
    var err error
    if req.CharacterType == "" {
        log.Info().Msg("No character specified, selecting automatically")
        selection, err = vg.characterSelector.SelectCharacter(ctx, req.ProductName, req.ProductDesc, req.ProductImageURL)
        if err != nil {
            return fmt.Errorf("failed to select character: %w", err)
        }
        req.CharacterType = selection.CharacterType
        req.CharacterName = selection.CharacterName
        video.CharacterType = selection.CharacterType
        video.CharacterName = selection.CharacterName
        video.ProductInfo = models.JSONB(map[string]interface{}{
            "analysis": selection.ProductAnalysis,
        })
    }
    video.Status = models.VideoStatusGenerating
    video.Progress = 15
    if req.Script == "" {
        log.Info().Msg("No script provided, generating automatically")
        if selection == nil {
            selection, err = vg.characterSelector.SelectCharacter(ctx, req.ProductName, req.ProductDesc, req.ProductImageURL)
            if err != nil {
                return fmt.Errorf("failed to select character for script: %w", err)
            }
        }
        req.Script, err = vg.characterSelector.GenerateScript(ctx, selection, req.ProductName, req.ProductDesc, req.Duration)
        if err != nil {
            return fmt.Errorf("failed to generate script: %w", err)
        }
        video.Script = req.Script
    }
    video.Progress = 30
    log.Info().Msg("Calling video generation API")
    jobID, err := vg.submitVideoGeneration(ctx, req)
    if err != nil {
        return fmt.Errorf("failed to submit video generation: %w", err)
    }
    video.ExternalJobID = jobID
    video.Status = models.VideoStatusProcessing
    video.Progress = 40
    log.Info().Str("video_id", video.ID.String()).Str("job_id", jobID).Msg("Video generation job submitted")
    return nil
}

func (vg *VideoGenerator) submitVideoGeneration(ctx context.Context, req *VideoGenerationRequest) (string, error) {
    payload := map[string]interface{}{
        "product_name":      req.ProductName,
        "product_description": req.ProductDesc,
        "product_image_url": req.ProductImageURL,
        "character_type":    req.CharacterType,
        "character_name":    req.CharacterName,
        "script":            req.Script,
        "duration":          req.Duration,
        "aspect_ratio":      req.AspectRatio,
        "resolution":        req.Resolution,
        "voice_type":        req.VoiceType,
        "style":             "ugc",
        "quality":           "high",
    }
    jsonData, err := json.Marshal(payload)
    if err != nil {
        return "", fmt.Errorf("failed to marshal payload: %w", err)
    }
    httpReq, err := http.NewRequestWithContext(ctx, "POST", vg.apiURL+"/generate", bytes.NewBuffer(jsonData))
    if err != nil {
        return "", fmt.Errorf("failed to create request: %w", err)
    }
    httpReq.Header.Set("Content-Type", "application/json")
    httpReq.Header.Set("Authorization", "Bearer "+vg.apiKey)
    resp, err := vg.httpClient.Do(httpReq)
    if err != nil {
        return "", fmt.Errorf("failed to send request: %w", err)
    }
    defer resp.Body.Close()
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return "", fmt.Errorf("failed to read response: %w", err)
    }
    if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
        return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
    }
    var result struct {
        JobID   string `json:"job_id"`
        Status  string `json:"status"`
        Message string `json:"message"`
    }
    if err := json.Unmarshal(body, &result); err != nil {
        return "", fmt.Errorf("failed to parse response: %w", err)
    }
    if result.JobID == "" {
        return "", fmt.Errorf("no job ID in response")
    }
    log.Info().Str("job_id", result.JobID).Str("status", result.Status).Msg("Video generation job created")
    return result.JobID, nil
}

func (vg *VideoGenerator) PollVideoStatus(ctx context.Context, jobID string) (*VideoGenerationJob, error) {
    httpReq, err := http.NewRequestWithContext(ctx, "GET", vg.apiURL+"/status/"+jobID, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %w", err)
    }
    httpReq.Header.Set("Authorization", "Bearer "+vg.apiKey)
    resp, err := vg.httpClient.Do(httpReq)
    if err != nil {
        return nil, fmt.Errorf("failed to send request: %w", err)
    }
    defer resp.Body.Close()
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("failed to read response: %w", err)
    }
    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
    }
    var result struct {
        JobID        string    `json:"job_id"`
        Status       string    `json:"status"`
        Progress     int       `json:"progress"`
        VideoURL     string    `json:"video_url"`
        ThumbnailURL string    `json:"thumbnail_url"`
        ErrorMessage string    `json:"error_message"`
        CreatedAt    time.Time `json:"created_at"`
        UpdatedAt    time.Time `json:"updated_at"`
    }
    if err := json.Unmarshal(body, &result); err != nil {
        return nil, fmt.Errorf("failed to parse response: %w", err)
    }
    job := &VideoGenerationJob{
        JobID:        result.JobID,
        Status:       result.Status,
        Progress:     result.Progress,
        VideoURL:     result.VideoURL,
        ThumbnailURL: result.ThumbnailURL,
        ErrorMessage: result.ErrorMessage,
        CreatedAt:    result.CreatedAt,
        UpdatedAt:    result.UpdatedAt,
    }
    return job, nil
}

func (vg *VideoGenerator) MonitorVideoGeneration(ctx context.Context, video *models.Video, updateCallback func(*models.Video) error) error {
    ticker := time.NewTicker(5 * time.Second)
    defer ticker.Stop()
    timeout := time.After(30 * time.Minute)
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-timeout:
            video.MarkFailed("Video generation timed out")
            return updateCallback(video)
        case <-ticker.C:
            job, err := vg.PollVideoStatus(ctx, video.ExternalJobID)
            if err != nil {
                log.Error().Err(err).Str("job_id", video.ExternalJobID).Msg("Failed to poll video status")
                continue
            }
            video.UpdateProgress(job.Status, job.Progress)
            if job.Status == "completed" && job.VideoURL != "" {
                log.Info().Str("video_id", video.ID.String()).Str("job_id", job.JobID).Msg("Video generation completed")
                if err := vg.downloadAndStoreVideo(ctx, video, job.VideoURL, job.ThumbnailURL); err != nil {
                    log.Error().Err(err).Msg("Failed to download video")
                    video.MarkFailed(fmt.Sprintf("Failed to download video: %v", err))
                } else {
                    video.MarkCompleted()
                }
                return updateCallback(video)
            }
            if job.Status == "failed" {
                log.Error().Str("video_id", video.ID.String()).Str("error", job.ErrorMessage).Msg("Video generation failed")
                video.MarkFailed(job.ErrorMessage)
                return updateCallback(video)
            }
            if err := updateCallback(video); err != nil {
                log.Error().Err(err).Msg("Failed to update video")
            }
        }
    }
}

func (vg *VideoGenerator) downloadAndStoreVideo(ctx context.Context, video *models.Video, videoURL, thumbnailURL string) error {
    log.Info().Str("video_url", videoURL).Msg("Downloading generated video")
    videoResp, err := http.Get(videoURL)
    if err != nil {
        return fmt.Errorf("failed to download video: %w", err)
    }
    defer videoResp.Body.Close()
    filename := fmt.Sprintf("%s_%s.mp4", video.ProductName, video.ID.String())
    opts := storage.NewUploadOptions()
    opts.Folder = "videos"
    opts.UserID = video.UserID
    opts.ContentType = "video/mp4"
    opts.ACL = "public-read"
    result, err := vg.storage.UploadFromReader(ctx, videoResp.Body, filename, "video/mp4", videoResp.ContentLength, opts)
    if err != nil {
        return fmt.Errorf("failed to upload video: %w", err)
    }
    video.StorageKey = result.StorageKey
    video.URL = result.URL
    video.FileSize = result.FileSize
    video.Format = "mp4"
    if thumbnailURL != "" {
        if err := vg.downloadAndStoreThumbnail(ctx, video, thumbnailURL); err != nil {
            log.Warn().Err(err).Msg("Failed to download thumbnail")
        }
    }
    log.Info().Str("video_id", video.ID.String()).Str("storage_key", video.StorageKey).Msg("Video stored successfully")
    return nil
}

func (vg *VideoGenerator) downloadAndStoreThumbnail(ctx context.Context, video *models.Video, thumbnailURL string) error {
    thumbResp, err := http.Get(thumbnailURL)
    if err != nil {
        return err
    }
    defer thumbResp.Body.Close()
    filename := fmt.Sprintf("%s_%s_thumb.jpg", video.ProductName, video.ID.String())
    opts := storage.NewUploadOptions()
    opts.Folder = "thumbnails"
    opts.UserID = video.UserID
    opts.ContentType = "image/jpeg"
    opts.ACL = "public-read"
    result, err := vg.storage.UploadFromReader(ctx, thumbResp.Body, filename, "image/jpeg", thumbResp.ContentLength, opts)
    if err != nil {
        return err
    }
    video.ThumbnailURL = result.URL
    return nil
}

func (vg *VideoGenerator) CancelVideoGeneration(ctx context.Context, jobID string) error {
    httpReq, err := http.NewRequestWithContext(ctx, "DELETE", vg.apiURL+"/cancel/"+jobID, nil)
    if err != nil {
        return fmt.Errorf("failed to create request: %w", err)
    }
    httpReq.Header.Set("Authorization", "Bearer "+vg.apiKey)
    resp, err := vg.httpClient.Do(httpReq)
    if err != nil {
        return fmt.Errorf("failed to send request: %w", err)
    }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
    }
    log.Info().Str("job_id", jobID).Msg("Video generation cancelled")
    return nil
}
