# Qilin Core - AI UGC Video Platform

This is the core backend service for Qilin's UGC ad creation platform. The service handles:

1. AI-powered conversation and analysis
2. Product description and image analysis
3. Character selection and script generation
4. Video ad generation and processing
5. Real-time WebSocket communication
6. File upload and storage management

## Code Generation Guidelines

When working on this project:

1. Use idiomatic Go code following standard Go project layout
2. Implement proper error handling with custom error types
3. Use GORM for database operations with proper model relationships
4. Follow REST API best practices for endpoints
5. Use WebSocket for real-time features
6. Implement proper validation for all inputs
7. Include comprehensive logging with zerolog
8. Add proper documentation and comments

## API Structure

Structure API handlers following this pattern:

```go
func (h *Handler) HandleEndpoint() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 1. Input validation
        // 2. Authorization check
        // 3. Business logic
        // 4. Response formatting
    }
}
```

## WebSocket Guidelines

For WebSocket handlers:

1. Use gorilla/websocket for connections
2. Implement proper connection pooling
3. Handle disconnections gracefully
4. Use proper message types for different events
5. Implement reconnection logic

## Model Guidelines

For database models:

1. Use UUIDs for primary keys
2. Implement proper GORM tags
3. Include proper indexes
4. Use soft deletes where appropriate
5. Implement proper validation
6. Add helper methods for common operations

## Error Handling

Use proper error handling with custom error types:

```go
type AppError struct {
    Code    int
    Message string
    Err     error
}
```

## Security Guidelines

1. Use proper JWT authentication
2. Implement rate limiting
3. Validate all file uploads
4. Sanitize user inputs
5. Use proper CORS settings
6. Implement proper logging
7. Use environment variables for sensitive data
