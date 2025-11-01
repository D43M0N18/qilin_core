package handlers

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "strings"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    "github.com/gorilla/websocket"
    "github.com/rs/zerolog/log"
    "github.com/D43M0N18/qilin_core/internal/database/repository"
    "github.com/D43M0N18/qilin_core/internal/models"
    "github.com/D43M0N18/qilin_core/internal/services/ai"
    wsservice "github.com/D43M0N18/qilin_core/internal/services/websocket"
)

var upgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
    CheckOrigin: func(r *http.Request) bool {
        // TODO: Implement proper origin checking in production
        return true
    },
}

// ChatHandler handles chat-related HTTP and WebSocket requests
// ...existing code...
type ChatHandler struct {
    conversationRepo *repository.ConversationRepository
    messageRepo      *repository.MessageRepository
    hub              *wsservice.Hub
    aiService        *ai.CharacterSelector
}

func NewChatHandler(conversationRepo *repository.ConversationRepository, messageRepo *repository.MessageRepository, hub *wsservice.Hub, aiService *ai.CharacterSelector) *ChatHandler {
    return &ChatHandler{
        conversationRepo: conversationRepo,
        messageRepo:      messageRepo,
        hub:              hub,
        aiService:        aiService,
    }
}

func (h *ChatHandler) CreateConversation(c *gin.Context) {
    userID := c.MustGet("user_id").(uuid.UUID)
    var input models.CreateConversationInput
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    conversation := &models.Conversation{
        UserID: userID,
        Title:  input.Title,
        Status: "active",
    }
    if input.Title == "" {
        conversation.Title = "New Conversation"
    }
    if err := h.conversationRepo.Create(c.Request.Context(), conversation); err != nil {
        log.Error().Err(err).Msg("Failed to create conversation")
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create conversation"})
        return
    }
    if input.InitialMessage != "" {
        message := &models.Message{
            ConversationID: conversation.ID,
            Role:           "user",
            Content:        input.InitialMessage,
        }
        if err := h.messageRepo.Create(c.Request.Context(), message); err != nil {
            log.Error().Err(err).Msg("Failed to create initial message")
        } else {
            conversation.UpdatePreview(input.InitialMessage)
            conversation.UpdateTitle(input.InitialMessage)
            h.conversationRepo.Update(c.Request.Context(), conversation)
        }
    }
    log.Info().Str("conversation_id", conversation.ID.String()).Str("user_id", userID.String()).Msg("Conversation created")
    c.JSON(http.StatusCreated, gin.H{
        "success": true,
        "data":    conversation.ToResponse(false),
    })
}

func (h *ChatHandler) ListConversations(c *gin.Context) {
    userID := c.MustGet("user_id").(uuid.UUID)
    conversations, err := h.conversationRepo.FindByUserID(c.Request.Context(), userID)
    if err != nil {
        log.Error().Err(err).Msg("Failed to list conversations")
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list conversations"})
        return
    }
    response := make([]*models.ConversationResponse, len(conversations))
    for i, conv := range conversations {
        response[i] = conv.ToResponse(false)
    }
    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "data":    response,
    })
}

func (h *ChatHandler) GetConversation(c *gin.Context) {
    userID := c.MustGet("user_id").(uuid.UUID)
    conversationID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid conversation ID"})
        return
    }
    conversation, err := h.conversationRepo.FindByID(c.Request.Context(), conversationID)
    if err != nil {
        log.Error().Err(err).Msg("Failed to get conversation")
        c.JSON(http.StatusNotFound, gin.H{"error": "Conversation not found"})
        return
    }
    if conversation.UserID != userID {
        c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
        return
    }
    messages, err := h.messageRepo.FindByConversationID(c.Request.Context(), conversationID)
    if err != nil {
        log.Error().Err(err).Msg("Failed to load messages")
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load messages"})
        return
    }
    conversation.Messages = messages
    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "data":    conversation.ToResponse(true),
    })
}

func (h *ChatHandler) DeleteConversation(c *gin.Context) {
    userID := c.MustGet("user_id").(uuid.UUID)
    conversationID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid conversation ID"})
        return
    }
    conversation, err := h.conversationRepo.FindByID(c.Request.Context(), conversationID)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Conversation not found"})
        return
    }
    if conversation.UserID != userID {
        c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
        return
    }
    if err := h.conversationRepo.Delete(c.Request.Context(), conversationID); err != nil {
        log.Error().Err(err).Msg("Failed to delete conversation")
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete conversation"})
        return
    }
    log.Info().Str("conversation_id", conversationID.String()).Str("user_id", userID.String()).Msg("Conversation deleted")
    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "message": "Conversation deleted successfully",
    })
}

func (h *ChatHandler) HandleWebSocket(c *gin.Context) {
    userID := c.MustGet("user_id").(uuid.UUID)
    conversationID, err := uuid.Parse(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid conversation ID"})
        return
    }
    conversation, err := h.conversationRepo.FindByID(c.Request.Context(), conversationID)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Conversation not found"})
        return
    }
    if conversation.UserID != userID {
        c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
        return
    }
    conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
    if err != nil {
        log.Error().Err(err).Msg("Failed to upgrade connection")
        return
    }
    client := wsservice.NewClient(h.hub, conn, userID, conversationID, h)
    h.hub.register <- client
    go client.WritePump()
    go client.ReadPump()
    log.Info().Str("user_id", userID.String()).Str("conversation_id", conversationID.String()).Str("client_id", client.GetID()).Msg("WebSocket connection established")
}

func (h *ChatHandler) HandleMessage(ctx context.Context, client *wsservice.Client, incomingMsg *wsservice.IncomingMessage) error {
    conversationID := client.GetConversationID()
    userID := client.GetUserID()
    log.Info().Str("conversation_id", conversationID.String()).Str("user_id", userID.String()).Str("content_preview", truncate(incomingMsg.Content, 50)).Msg("Handling incoming message")
    userMessage := &models.Message{
        ConversationID: conversationID,
        Role:           "user",
        Content:        incomingMsg.Content,
    }
    if err := h.messageRepo.Create(ctx, userMessage); err != nil {
        return fmt.Errorf("failed to create message: %w", err)
    }
    wsMsg := models.NewWebSocketMessage(models.MessageTypeComplete, conversationID, userMessage.ID)
    wsMsg.Role = "user"
    wsMsg.Content = userMessage.Content
    client.BroadcastToConversationExceptSelf(wsMsg)
    conversation, _ := h.conversationRepo.FindByID(ctx, conversationID)
    if conversation != nil {
        if conversation.Title == "New Conversation" {
            conversation.UpdateTitle(incomingMsg.Content)
        }
        conversation.UpdatePreview(incomingMsg.Content)
        h.conversationRepo.Update(ctx, conversation)
    }
    go h.generateAIResponse(ctx, client, conversationID, userMessage)
    return nil
}

func (h *ChatHandler) generateAIResponse(ctx context.Context, client *wsservice.Client, conversationID uuid.UUID, userMessage *models.Message) {
    assistantMessage := &models.Message{
        ConversationID: conversationID,
        Role:           "assistant",
        Content:        "",
        IsStreaming:    true,
    }
    if err := h.messageRepo.Create(ctx, assistantMessage); err != nil {
        log.Error().Err(err).Msg("Failed to create assistant message")
        client.SendError("Failed to create response")
        return
    }
    startMsg := models.NewWebSocketMessage(models.MessageTypeStart, conversationID, assistantMessage.ID)
    startMsg.Role = "assistant"
    client.BroadcastToConversation(startMsg)
    responseText := h.generateMockResponse(userMessage.Content)
    words := splitIntoWords(responseText)
    for _, word := range words {
        deltaMsg := models.NewWebSocketMessage(models.MessageTypeDelta, conversationID, assistantMessage.ID)
        deltaMsg.Delta = word + " "
        client.BroadcastToConversation(deltaMsg)
        assistantMessage.AppendContent(word + " ")
        time.Sleep(50 * time.Millisecond)
    }
    assistantMessage.CompleteStream()
    if err := h.messageRepo.Update(ctx, assistantMessage); err != nil {
        log.Error().Err(err).Msg("Failed to update message")
    }
    completeMsg := models.NewWebSocketMessage(models.MessageTypeComplete, conversationID, assistantMessage.ID)
    completeMsg.Content = assistantMessage.Content
    completeMsg.Role = "assistant"
    client.BroadcastToConversation(completeMsg)
    log.Info().Str("message_id", assistantMessage.ID.String()).Int("content_length", len(assistantMessage.Content)).Msg("AI response completed")
}

func (h *ChatHandler) HandleTyping(ctx context.Context, client *wsservice.Client) error {
    client.SendTypingIndicator(true)
    return nil
}

func (h *ChatHandler) HandleDisconnect(ctx context.Context, client *wsservice.Client) error {
    log.Info().Str("user_id", client.GetUserID().String()).Str("conversation_id", client.GetConversationID().String()).Str("client_id", client.GetID()).Msg("Client disconnected")
    return nil
}

func truncate(s string, maxLen int) string {
    if len(s) <= maxLen {
        return s
    }
    return s[:maxLen] + "..."
}

func splitIntoWords(s string) []string {
    return strings.Fields(s)
}

func (h *ChatHandler) generateMockResponse(input string) string {
    return fmt.Sprintf("I received your message: '%s'. I'm an AI assistant helping you create UGC ad content. How can I assist you with your product advertisement needs?", truncate(input, 50))
}
