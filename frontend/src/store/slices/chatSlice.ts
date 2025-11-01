import { createSlice, PayloadAction } from '@reduxjs/toolkit';

interface Attachment {
  id: string;
  file_name: string;
  url: string;
  thumbnail_url?: string;
  file_type: string;
  is_image: boolean;
}

interface Message {
  id: string;
  conversation_id: string;
  role: 'user' | 'assistant' | 'system';
  content: string;
  attachments?: Attachment[];
  is_streaming: boolean;
  stream_complete?: boolean;
  created_at: string;
  updated_at: string;
}

interface Conversation {
  id: string;
  title: string;
  preview: string;
  status: string;
  message_count: number;
  created_at: string;
  updated_at: string;
}

interface ChatState {
  messages: Message[];
  currentConversation: Conversation | null;
  isTyping: boolean;
  isLoading: boolean;
  error: string | null;
}

const initialState: ChatState = {
  messages: [],
  currentConversation: null,
  isTyping: false,
  isLoading: false,
  error: null,
};

// Fix: Accept messages in payload for setCurrentConversation
interface ConversationWithMessages extends Conversation {
  messages?: Message[];
}

const chatSlice = createSlice({
  name: 'chat',
  initialState,
  reducers: {
    setCurrentConversation: (state, action: PayloadAction<ConversationWithMessages>) => {
      state.currentConversation = action.payload;
      if (action.payload.messages) {
        state.messages = action.payload.messages;
      }
    },

    addMessage: (state, action: PayloadAction<Message>) => {
      const existingIndex = state.messages.findIndex((m) => m.id === action.payload.id);
      if (existingIndex === -1) {
        state.messages.push(action.payload);
      }
    },

    updateMessage: (
      state,
      action: PayloadAction<{
        id: string;
        delta?: string;
        content?: string;
        is_streaming?: boolean;
        stream_complete?: boolean;
      }>
    ) => {
      const message = state.messages.find((m) => m.id === action.payload.id);
      if (message) {
        if (action.payload.delta) {
          message.content += action.payload.delta;
        }
        if (action.payload.content !== undefined) {
          message.content = action.payload.content;
        }
        if (action.payload.is_streaming !== undefined) {
          message.is_streaming = action.payload.is_streaming;
        }
        if (action.payload.stream_complete !== undefined) {
          message.stream_complete = action.payload.stream_complete;
        }
        message.updated_at = new Date().toISOString();
      }
    },

    deleteMessage: (state, action: PayloadAction<string>) => {
      state.messages = state.messages.filter((m) => m.id !== action.payload);
    },

    clearMessages: (state) => {
      state.messages = [];
    },

    setTyping: (state, action: PayloadAction<boolean>) => {
      state.isTyping = action.payload;
    },

    setLoading: (state, action: PayloadAction<boolean>) => {
      state.isLoading = action.payload;
    },

    setError: (state, action: PayloadAction<string | null>) => {
      state.error = action.payload;
    },
  },
});

export const {
  setCurrentConversation,
  addMessage,
  updateMessage,
  deleteMessage,
  clearMessages,
  setTyping,
  setLoading,
  setError,
} = chatSlice.actions;

export default chatSlice.reducer;
