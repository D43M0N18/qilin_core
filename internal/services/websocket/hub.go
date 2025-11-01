package websocket

import (
    "context"
    "encoding/json"
    "fmt"
    "sync"
    "time"

    "github.com/google/uuid"
    "github.com/rs/zerolog/log"
    
    "github.com/D43M0N18/qilin_core/internal/models"
)

// Hub maintains the set of active clients and broadcasts messages
type Hub struct {
    clients map[uuid.UUID]map[*Client]bool
    conversations map[uuid.UUID]map[*Client]bool
    broadcast chan *BroadcastMessage
    register chan *Client
    unregister chan *Client
    mu sync.RWMutex
    ctx context.Context
    cancel context.CancelFunc
}

// BroadcastMessage represents a message to be broadcast
type BroadcastMessage struct {
    ConversationID uuid.UUID
    UserID         uuid.UUID
    Message        interface{}
    ExcludeClient  *Client // Don't send to this client (sender)
}

// NewHub creates a new Hub instance
func NewHub() *Hub {
    ctx, cancel := context.WithCancel(context.Background())
    return &Hub{
        broadcast:     make(chan *BroadcastMessage, 256),
        register:      make(chan *Client, 64),
        unregister:    make(chan *Client, 64),
        clients:       make(map[uuid.UUID]map[*Client]bool),
        conversations: make(map[uuid.UUID]map[*Client]bool),
        ctx:           ctx,
        cancel:        cancel,
    }
}

func (h *Hub) Run() {
    defer func() {
        log.Info().Msg("Hub shutting down")
        h.cleanup()
    }()
    cleanupTicker := time.NewTicker(30 * time.Second)
    defer cleanupTicker.Stop()
    statsTicker := time.NewTicker(5 * time.Minute)
    defer statsTicker.Stop()
    for {
        select {
        case <-h.ctx.Done():
            log.Info().Msg("Hub context cancelled, stopping")
            return
        case client := <-h.register:
            h.registerClient(client)
        case client := <-h.unregister:
            h.unregisterClient(client)
        case message := <-h.broadcast:
            h.broadcastMessage(message)
        case <-cleanupTicker.C:
            h.cleanupStaleConnections()
        case <-statsTicker.C:
            h.logStatistics()
        }
    }
}

func (h *Hub) registerClient(client *Client) {
    h.mu.Lock()
    defer h.mu.Unlock()
    if h.clients[client.userID] == nil {
        h.clients[client.userID] = make(map[*Client]bool)
    }
    h.clients[client.userID][client] = true
    if client.conversationID != uuid.Nil {
        if h.conversations[client.conversationID] == nil {
            h.conversations[client.conversationID] = make(map[*Client]bool)
        }
        h.conversations[client.conversationID][client] = true
    }
    log.Info().Str("user_id", client.userID.String()).Str("conversation_id", client.conversationID.String()).Str("client_id", client.id).Msg("Client registered")
}

func (h *Hub) unregisterClient(client *Client) {
    h.mu.Lock()
    defer h.mu.Unlock()
    if clients, ok := h.clients[client.userID]; ok {
        if _, exists := clients[client]; exists {
            delete(clients, client)
            close(client.send)
            if len(clients) == 0 {
                delete(h.clients, client.userID)
            }
        }
    }
    if client.conversationID != uuid.Nil {
        if clients, ok := h.conversations[client.conversationID]; ok {
            delete(clients, client)
            if len(clients) == 0 {
                delete(h.conversations, client.conversationID)
            }
        }
    }
    log.Info().Str("user_id", client.userID.String()).Str("conversation_id", client.conversationID.String()).Str("client_id", client.id).Msg("Client unregistered")
}

func (h *Hub) broadcastMessage(broadcast *BroadcastMessage) {
    h.mu.RLock()
    defer h.mu.RUnlock()
    data, err := json.Marshal(broadcast.Message)
    if err != nil {
        log.Error().Err(err).Msg("Failed to marshal broadcast message")
        return
    }
    if clients, ok := h.conversations[broadcast.ConversationID]; ok {
        sentCount := 0
        for client := range clients {
            if broadcast.ExcludeClient != nil && client == broadcast.ExcludeClient {
                continue
            }
            select {
            case client.send <- data:
                sentCount++
            default:
                log.Warn().Str("client_id", client.id).Msg("Client send channel full, unregistering")
                go h.unregisterClient(client)
            }
        }
        log.Debug().Str("conversation_id", broadcast.ConversationID.String()).Int("recipients", sentCount).Msg("Message broadcast")
    }
}

func (h *Hub) BroadcastToConversation(conversationID uuid.UUID, message interface{}, excludeClient *Client) {
    h.broadcast <- &BroadcastMessage{
        ConversationID: conversationID,
        Message:        message,
        ExcludeClient:  excludeClient,
    }
}

func (h *Hub) BroadcastToUser(userID uuid.UUID, message interface{}) {
    h.mu.RLock()
    defer h.mu.RUnlock()
    data, err := json.Marshal(message)
    if err != nil {
        log.Error().Err(err).Msg("Failed to marshal user broadcast message")
        return
    }
    if clients, ok := h.clients[userID]; ok {
        for client := range clients {
            select {
            case client.send <- data:
            default:
                log.Warn().Str("client_id", client.id).Msg("Client send channel full")
            }
        }
    }
}

func (h *Hub) SendToClient(client *Client, message interface{}) error {
    data, err := json.Marshal(message)
    if err != nil {
        return err
    }
    select {
    case client.send <- data:
        return nil
    case <-time.After(5 * time.Second):
        return fmt.Errorf("timeout sending to client")
    }
}

func (h *Hub) GetConversationClients(conversationID uuid.UUID) []*Client {
    h.mu.RLock()
    defer h.mu.RUnlock()
    var clients []*Client
    if conversationClients, ok := h.conversations[conversationID]; ok {
        for client := range conversationClients {
            clients = append(clients, client)
        }
    }
    return clients
}

func (h *Hub) GetUserClients(userID uuid.UUID) []*Client {
    h.mu.RLock()
    defer h.mu.RUnlock()
    var clients []*Client
    if userClients, ok := h.clients[userID]; ok {
        for client := range userClients {
            clients = append(clients, client)
        }
    }
    return clients
}

func (h *Hub) IsUserConnected(userID uuid.UUID) bool {
    h.mu.RLock()
    defer h.mu.RUnlock()
    clients, ok := h.clients[userID]
    return ok && len(clients) > 0
}

func (h *Hub) GetConversationClientCount(conversationID uuid.UUID) int {
    h.mu.RLock()
    defer h.mu.RUnlock()
    if clients, ok := h.conversations[conversationID]; ok {
        return len(clients)
    }
    return 0
}

func (h *Hub) cleanupStaleConnections() {
    h.mu.Lock()
    defer h.mu.Unlock()
    now := time.Now()
    staleThreshold := 5 * time.Minute
    var staleClients []*Client
    for _, clients := range h.clients {
        for client := range clients {
            if now.Sub(client.lastActivity) > staleThreshold {
                staleClients = append(staleClients, client)
            }
        }
    }
    if len(staleClients) > 0 {
        log.Info().Int("count", len(staleClients)).Msg("Cleaning up stale connections")
        for _, client := range staleClients {
            go h.unregisterClient(client)
        }
    }
}

func (h *Hub) logStatistics() {
    h.mu.RLock()
    defer h.mu.RUnlock()
    totalClients := 0
    for _, clients := range h.clients {
        totalClients += len(clients)
    }
    log.Info().Int("total_clients", totalClients).Int("unique_users", len(h.clients)).Int("active_conversations", len(h.conversations)).Msg("Hub statistics")
}

func (h *Hub) cleanup() {
    h.mu.Lock()
    defer h.mu.Unlock()
    for _, clients := range h.clients {
        for client := range clients {
            close(client.send)
        }
    }
    h.clients = make(map[uuid.UUID]map[*Client]bool)
    h.conversations = make(map[uuid.UUID]map[*Client]bool)
}

func (h *Hub) Shutdown() {
    log.Info().Msg("Initiating hub shutdown")
    h.cancel()
}

func (h *Hub) GetStats() map[string]interface{} {
    h.mu.RLock()
    defer h.mu.RUnlock()
    totalClients := 0
    for _, clients := range h.clients {
        totalClients += len(clients)
    }
    return map[string]interface{}{
        "total_clients":        totalClients,
        "unique_users":         len(h.clients),
        "active_conversations": len(h.conversations),
        "timestamp":            time.Now(),
    }
}
