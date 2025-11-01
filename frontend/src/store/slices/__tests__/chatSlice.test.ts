import { describe, it, expect } from 'vitest';
import chatReducer, {
  addMessage,
  clearMessages,
  setTyping,
} from '../chatSlice';

describe('chatSlice', () => {
  const initialState = {
    messages: [],
    currentConversation: null,
    isTyping: false,
    isLoading: false,
    error: null,
  };

  it('should return initial state', () => {
    expect(chatReducer(undefined, { type: 'unknown' })).toEqual(initialState);
  });

  it('should handle setTyping', () => {
    const state = chatReducer(initialState, setTyping(true));
    expect(state.isTyping).toBe(true);
  });

  it('should handle clearMessages', () => {
    const stateWithMessages = {
      ...initialState,
      messages: [
        {
          id: '1',
          conversation_id: 'conv-1',
          role: 'user' as const,
          content: 'Test',
          is_streaming: false,
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
        },
      ],
    };

    const state = chatReducer(stateWithMessages, clearMessages());
    expect(state.messages).toHaveLength(0);
  });

  it('should handle addMessage', () => {
    const message = {
      id: 'test-id',
      conversation_id: 'conv-id',
      role: 'user' as const,
      content: 'Test message',
      is_streaming: false,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    };

    const state = chatReducer(initialState, addMessage(message));
    expect(state.messages).toHaveLength(1);
    expect(state.messages[0]).toEqual(message);
  });
});
