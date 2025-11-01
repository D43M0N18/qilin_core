import { useEffect, useRef, useState, useCallback } from 'react';
import { useDispatch } from 'react-redux';
import { addMessage, updateMessage, setTyping } from '../store/slices/chatSlice';
import { updateVideoProgress } from '../store/slices/videoSlice';

interface WebSocketMessage {
  type: string;
  message_id?: string;
  conversation_id?: string;
  role?: string;
  content?: string;
  delta?: string;
  attachments?: any[];
  metadata?: any;
  error?: string;
  timestamp: string;
}

interface UseWebSocketOptions {
  conversationId: string;
  token: string;
  onConnect?: () => void;
  onDisconnect?: () => void;
  onError?: (error: Event) => void;
}

export const useWebSocket = ({
  conversationId,
  token,
  onConnect,
  onDisconnect,
  onError,
}: UseWebSocketOptions) => {
  const [isConnected, setIsConnected] = useState(false);
  const [isTyping, setIsTypingState] = useState(false);
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<NodeJS.Timeout>();
  const reconnectAttemptsRef = useRef(0);
  const messageQueueRef = useRef<string[]>([]);
  const dispatch = useDispatch();

  const MAX_RECONNECT_ATTEMPTS = 5;
  const RECONNECT_INTERVAL = 3000;

  // Connect to WebSocket
  const connect = useCallback(() => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      return;
    }

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/api/v1/ws/conversations/${conversationId}?token=${token}`;

    try {
      const ws = new WebSocket(wsUrl);

      ws.onopen = () => {
        console.log('WebSocket connected');
        setIsConnected(true);
        reconnectAttemptsRef.current = 0;
        onConnect?.();

        // Send queued messages
        while (messageQueueRef.current.length > 0) {
          const message = messageQueueRef.current.shift();
          if (message) {
            ws.send(message);
          }
        }
      };

      ws.onclose = (event) => {
        console.log('WebSocket disconnected', event);
        setIsConnected(false);
        wsRef.current = null;
        onDisconnect?.();

        // Attempt to reconnect
        if (reconnectAttemptsRef.current < MAX_RECONNECT_ATTEMPTS) {
          reconnectAttemptsRef.current++;
          console.log(
            `Reconnecting... Attempt ${reconnectAttemptsRef.current}/${MAX_RECONNECT_ATTEMPTS}`
          );
          reconnectTimeoutRef.current = setTimeout(() => {
            connect();
          }, RECONNECT_INTERVAL * reconnectAttemptsRef.current);
        }
      };

      ws.onerror = (error) => {
        console.error('WebSocket error', error);
        onError?.(error);
      };

      ws.onmessage = (event) => {
        try {
          const data: WebSocketMessage = JSON.parse(event.data);
          handleMessage(data);
        } catch (error) {
          console.error('Failed to parse WebSocket message', error);
        }
      };

      wsRef.current = ws;
    } catch (error) {
      console.error('Failed to create WebSocket connection', error);
    }
  }, [conversationId, token, onConnect, onDisconnect, onError]);

  // Handle incoming messages
  const handleMessage = useCallback(
    (data: WebSocketMessage) => {
      switch (data.type) {
        case 'message_start':
          // New message starting
          dispatch(
            addMessage({
              id: data.message_id!,
              conversation_id: data.conversation_id!,
              role: data.role!,
              content: '',
              is_streaming: true,
              created_at: data.timestamp,
              updated_at: data.timestamp,
            })
          );
          break;

        case 'content_delta':
          // Streaming content update
          dispatch(
            updateMessage({
              id: data.message_id!,
              delta: data.delta!,
            })
          );
          break;

        case 'message_complete':
          // Message finished
          dispatch(
            updateMessage({
              id: data.message_id!,
              content: data.content!,
              is_streaming: false,
              stream_complete: true,
            })
          );
          break;

        case 'typing':
          // Typing indicator
          const isUserTyping = data.metadata?.user_id !== token;
          setIsTypingState(isUserTyping && data.metadata?.is_typing);
          dispatch(setTyping(isUserTyping && data.metadata?.is_typing));
          break;

        case 'video_progress':
          // Video generation progress
          dispatch(
            updateVideoProgress({
              videoId: data.metadata?.video_id,
              status: data.metadata?.status,
              progress: data.metadata?.progress,
              video: data.metadata?.video,
            })
          );
          break;

        case 'error':
          // Error message
          console.error('WebSocket error message:', data.error);
          break;

        case 'pong':
          // Pong response
          break;

        default:
          console.warn('Unknown message type:', data.type);
      }
    },
    [dispatch, token]
  );

  // Send message
  const sendMessage = useCallback(
    (content: string, attachmentIds: string[] = []) => {
      const message = JSON.stringify({
        type: 'message',
        content,
        attachment_ids: attachmentIds,
        conversation_id: conversationId,
      });

      if (wsRef.current?.readyState === WebSocket.OPEN) {
        wsRef.current.send(message);
      } else {
        // Queue message if not connected
        messageQueueRef.current.push(message);
        connect();
      }
    },
    [conversationId, connect]
  );

  // Send typing indicator
  const sendTyping = useCallback(() => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(
        JSON.stringify({
          type: 'typing',
          conversation_id: conversationId,
        })
      );
    }
  }, [conversationId]);

  // Disconnect
  const disconnect = useCallback(() => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
    }
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }
    setIsConnected(false);
  }, []);

  // Connect on mount, disconnect on unmount
  useEffect(() => {
    connect();

    return () => {
      disconnect();
    };
  }, [connect, disconnect]);

  // Ping interval to keep connection alive
  useEffect(() => {
    if (!isConnected) return;

    const pingInterval = setInterval(() => {
      if (wsRef.current?.readyState === WebSocket.OPEN) {
        wsRef.current.send(JSON.stringify({ type: 'ping' }));
      }
    }, 30000); // Ping every 30 seconds

    return () => clearInterval(pingInterval);
  }, [isConnected]);

  return {
    isConnected,
    isTyping,
    sendMessage,
    sendTyping,
    disconnect,
    reconnect: connect,
  };
};
