package websocket

import (
    "context"
    "encoding/json"
    "fmt"
    "sync"
    "time"

    "github.com/google/uuid"
    "github.com/gorilla/websocket"
    "github.com/rs/zerolog/log"
    
    "github.com/D43M0N18/qilin_core/internal/models"
)

const (
    writeWait      = 10 * time.Second
    pongWait       = 60 * time.Second
    pingPeriod     = (pongWait * 9) / 10
    maxMessageSize = 10 * 1024 * 1024
    sendBufferSize = 256
)

// Client represents a WebSocket client connection
// ...existing code...
type Client struct {
    id             string
    hub            *Hub
    conn           *websocket.Conn
    send           chan []byte
    userID         uuid.UUID
    conversationID uuid.UUID
    lastActivity   time.Time
    mu             sync.RWMutex
    messageHandler MessageHandler
    ctx    context.Context
    cancel context.CancelFunc
}

// MessageHandler defines the interface for handling incoming messages
// ...existing code...
type MessageHandler interface {
    HandleMessage(ctx context.Context, client *Client, message *IncomingMessage) error
    HandleTyping(ctx context.Context, client *Client) error
    HandleDisconnect(ctx context.Context, client *Client) error
}

// IncomingMessage represents a message from the client
// ...existing code...
type IncomingMessage struct {
    Type           string                 `json:"type"`
    Content        string                 `json:"content,omitempty"`
    ConversationID uuid.UUID              `json:"conversation_id,omitempty"`
    AttachmentIDs  []uuid.UUID            `json:"attachment_ids,omitempty"`
    Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// NewClient creates a new Client instance
func NewClient(hub *Hub, conn *websocket.Conn, userID uuid.UUID, conversationID uuid.UUID, handler MessageHandler) *Client {
    ctx, cancel := context.WithCancel(context.Background())
    return &Client{
        id:             uuid.New().String(),
        hub:            hub,
        conn:           conn,
        send:           make(chan []byte, sendBufferSize),
        userID:         userID,
        conversationID: conversationID,
        lastActivity:   time.Now(),
        messageHandler: handler,
        ctx:            ctx,
        cancel:         cancel,
    }
}

func (c *Client) ReadPump() {
    defer func() {
        c.hub.unregister <- c
        c.conn.Close()
        c.cancel()
        if c.messageHandler != nil {
            if err := c.messageHandler.HandleDisconnect(context.Background(), c); err != nil {
                log.Error().Err(err).Str("client_id", c.id).Msg("Error handling disconnect")
            }
        }
    }()
    c.conn.SetReadDeadline(time.Now().Add(pongWait))
    c.conn.SetReadLimit(maxMessageSize)
    c.conn.SetPongHandler(func(string) error {
        c.conn.SetReadDeadline(time.Now().Add(pongWait))
        c.updateActivity()
        return nil
    })
    for {
        select {
        case <-c.ctx.Done():
            log.Info().Str("client_id", c.id).Msg("Read pump context cancelled")
            return
        default:
            _, messageBytes, err := c.conn.ReadMessage()
            if err != nil {
                if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
                    log.Error().Err(err).Str("client_id", c.id).Msg("WebSocket error")
                } else {
                    log.Info().Str("client_id", c.id).Msg("Client disconnected")
                }
                return
            }
            c.updateActivity()
            var incomingMsg IncomingMessage
            if err := json.Unmarshal(messageBytes, &incomingMsg); err != nil {
                log.Error().Err(err).Str("client_id", c.id).Str("raw_message", string(messageBytes)).Msg("Failed to parse incoming message")
                c.SendError("Invalid message format")
                continue
            }
            log.Debug().Str("client_id", c.id).Str("type", incomingMsg.Type).Str("conversation_id", incomingMsg.ConversationID.String()).Msg("Received message")
            if err := c.handleIncomingMessage(&incomingMsg); err != nil {
                log.Error().Err(err).Str("client_id", c.id).Str("type", incomingMsg.Type).Msg("Error handling message")
                c.SendError(fmt.Sprintf("Error processing message: %v", err))
            }
        }
    }
}

func (c *Client) WritePump() {
    ticker := time.NewTicker(pingPeriod)
    defer func() {
        ticker.Stop()
        c.conn.Close()
        c.cancel()
    }()
    for {
        select {
        case <-c.ctx.Done():
            log.Info().Str("client_id", c.id).Msg("Write pump context cancelled")
            return
        case message, ok := <-c.send:
            c.conn.SetWriteDeadline(time.Now().Add(writeWait))
            if !ok {
                c.conn.WriteMessage(websocket.CloseMessage, []byte{})
                return
            }
            w, err := c.conn.NextWriter(websocket.TextMessage)
            if err != nil {
                log.Error().Err(err).Str("client_id", c.id).Msg("Error getting writer")
                return
            }
            w.Write(message)
            n := len(c.send)
            for i := 0; i < n; i++ {
                w.Write([]byte{'\n'})
                w.Write(<-c.send)
            }
            if err := w.Close(); err != nil {
                log.Error().Err(err).Str("client_id", c.id).Msg("Error closing writer")
                return
            }
        case <-ticker.C:
            c.conn.SetWriteDeadline(time.Now().Add(writeWait))
            if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
                log.Error().Err(err).Str("client_id", c.id).Msg("Error sending ping")
                return
            }
        }
    }
}

func (c *Client) handleIncomingMessage(msg *IncomingMessage) error {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    switch msg.Type {
    case "message":
        if c.messageHandler != nil {
            return c.messageHandler.HandleMessage(ctx, c, msg)
        }
    case "typing":
        if c.messageHandler != nil {
            return c.messageHandler.HandleTyping(ctx, c)
        }
    case "ping":
        return c.SendMessage(models.NewWebSocketMessage("pong", c.conversationID, uuid.Nil))
    default:
        return fmt.Errorf("unknown message type: %s", msg.Type)
    }
    return nil
}

func (c *Client) SendMessage(message *models.WebSocketMessage) error {
    data, err := json.Marshal(message)
    if err != nil {
        return fmt.Errorf("failed to marshal message: %w", err)
    }
    select {
    case c.send <- data:
        return nil
    case <-time.After(5 * time.Second):
        return fmt.Errorf("timeout sending message to client")
    case <-c.ctx.Done():
        return fmt.Errorf("client context cancelled")
    }
}

func (c *Client) SendError(errorMsg string) {
    msg := models.NewWebSocketMessage(models.MessageTypeError, c.conversationID, uuid.Nil)
    msg.Error = errorMsg
    if err := c.SendMessage(msg); err != nil {
        log.Error().Err(err).Str("client_id", c.id).Str("error_message", errorMsg).Msg("Failed to send error to client")
    }
}

func (c *Client) SendTypingIndicator(isTyping bool) {
    msg := models.NewWebSocketMessage(models.MessageTypeTyping, c.conversationID, uuid.Nil)
    msg.Metadata = map[string]interface{}{
        "user_id":   c.userID.String(),
        "is_typing": isTyping,
    }
    c.hub.BroadcastToConversation(c.conversationID, msg, c)
}

func (c *Client) BroadcastToConversation(message *models.WebSocketMessage) {
    c.hub.BroadcastToConversation(c.conversationID, message, nil)
}

func (c *Client) BroadcastToConversationExceptSelf(message *models.WebSocketMessage) {
    c.hub.BroadcastToConversation(c.conversationID, message, c)
}

func (c *Client) updateActivity() {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.lastActivity = time.Now()
}

func (c *Client) GetLastActivity() time.Time {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return c.lastActivity
}

func (c *Client) GetUserID() uuid.UUID {
    return c.userID
}

func (c *Client) GetConversationID() uuid.UUID {
    return c.conversationID
}

func (c *Client) GetID() string {
    return c.id
}

func (c *Client) Close() error {
    c.cancel()
    return c.conn.Close()
}
