import { useCallback, useEffect } from 'react';
import { useDispatch, useSelector } from 'react-redux';
import { RootState } from '../store';
import { useWebSocket } from './useWebSocket';
import {
  setCurrentConversation,
  addMessage,
  clearMessages,
} from '@/store/slices/chatSlice';
import { api } from '../services/api';

interface UseChatOptions {
  conversationId: string;
  autoConnect?: boolean;
}

export const useChat = ({ conversationId, autoConnect = true }: UseChatOptions) => {
  const dispatch = useDispatch();
  const { token } = useSelector((state: RootState) => state.auth);
  const { messages, isTyping, currentConversation } = useSelector(
    (state: RootState) => state.chat
  );

  // WebSocket connection
  const {
    isConnected,
    sendMessage: wsSendMessage,
    sendTyping,
    disconnect,
    reconnect,
  } = useWebSocket({
    conversationId,
    token: token || '',
    onConnect: () => {
      console.log('Connected to chat');
    },
    onDisconnect: () => {
      console.log('Disconnected from chat');
    },
    onError: (error) => {
      console.error('WebSocket error:', error);
    },
  });

  // Load conversation on mount
  useEffect(() => {
    if (conversationId && autoConnect) {
      loadConversation();
    }

    return () => {
      dispatch(clearMessages());
    };
  }, [conversationId, autoConnect]);

  // Load conversation details and messages
  const loadConversation = useCallback(async () => {
    try {
      const response = await api.get(`/conversations/${conversationId}`);
      dispatch(setCurrentConversation(response.data.data));
    } catch (error: any) {
      console.error('Failed to load conversation:', error);
      throw error;
    }
  }, [conversationId, dispatch]);

  // Send message
  const sendMessage = useCallback(
    async (content: string, attachmentIds: string[] = []) => {
      if (!content.trim() && attachmentIds.length === 0) {
        return;
      }

      // Optimistically add user message
      const tempId = `temp-${Date.now()}`;
      dispatch(
        addMessage({
          id: tempId,
          conversation_id: conversationId,
          role: 'user',
          content,
          attachments: [],
          is_streaming: false,
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
        })
      );

      try {
        // Send via WebSocket
        wsSendMessage(content, attachmentIds);
      } catch (error: any) {
        console.error('Failed to send message:', error);
        // TODO: Handle error - maybe show retry option
        throw error;
      }
    },
    [conversationId, dispatch, wsSendMessage]
  );

  // Send typing indicator
  const startTyping = useCallback(() => {
    sendTyping();
  }, [sendTyping]);

  // Retry failed message
  const retryMessage = useCallback(
    async (messageId: string) => {
      const message = messages.find((m) => m.id === messageId);
      if (message) {
        await sendMessage(message.content, []);
      }
    },
    [messages, sendMessage]
  );

  // Regenerate assistant response
  const regenerateResponse = useCallback(async () => {
    // Find the last user message
    const lastUserMessage = [...messages]
      .reverse()
      .find((m) => m.role === 'user');

    if (lastUserMessage) {
      // Remove last assistant message if exists
      const lastAssistantIndex = messages.findIndex(
        (m, i) => i > messages.indexOf(lastUserMessage) && m.role === 'assistant'
      );

      if (lastAssistantIndex !== -1) {
        // TODO: Implement message deletion from store
      }

      // Resend the last user message
      await sendMessage(lastUserMessage.content, []);
    }
  }, [messages, sendMessage]);

  return {
    // State
    messages,
    isTyping,
    isConnected,
    currentConversation,

    // Actions
    sendMessage,
    startTyping,
    retryMessage,
    regenerateResponse,
    loadConversation,
    disconnect,
    reconnect,
  };
};
