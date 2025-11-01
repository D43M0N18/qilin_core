import { createSlice, createAsyncThunk, PayloadAction } from '@reduxjs/toolkit';
import { api } from '../../services/api';

interface Conversation {
  id: string;
  title: string;
  preview: string;
  status: string;
  message_count: number;
  last_message_at?: string;
  created_at: string;
  updated_at: string;
}

interface ConversationState {
  list: Conversation[];
  isLoading: boolean;
  error: string | null;
}

const initialState: ConversationState = {
  list: [],
  isLoading: false,
  error: null,
};

// Async thunks
export const fetchConversations = createAsyncThunk(
  'conversations/fetchAll',
  async (_, { rejectWithValue }) => {
    try {
      const response = await api.get('/conversations');
      return response.data.data;
    } catch (error: any) {
      return rejectWithValue(
        error.response?.data?.error || 'Failed to fetch conversations'
      );
    }
  }
);

export const createConversation = createAsyncThunk(
  'conversations/create',
  async (
    data: { title?: string; initial_message?: string },
    { rejectWithValue }
  ) => {
    try {
      const response = await api.post('/conversations', data);
      return response.data.data;
    } catch (error: any) {
      return rejectWithValue(
        error.response?.data?.error || 'Failed to create conversation'
      );
    }
  }
);

export const updateConversation = createAsyncThunk(
  'conversations/update',
  async (
    { id, data }: { id: string; data: { title?: string; status?: string } },
    { rejectWithValue }
  ) => {
    try {
      const response = await api.put(`/conversations/${id}`, data);
      return response.data.data;
    } catch (error: any) {
      return rejectWithValue(
        error.response?.data?.error || 'Failed to update conversation'
      );
    }
  }
);

export const deleteConversation = createAsyncThunk(
  'conversations/delete',
  async (id: string, { rejectWithValue }) => {
    try {
      await api.delete(`/conversations/${id}`);
      return id;
    } catch (error: any) {
      return rejectWithValue(
        error.response?.data?.error || 'Failed to delete conversation'
      );
    }
  }
);

// Slice
const conversationSlice = createSlice({
  name: 'conversations',
  initialState,
  reducers: {
    addConversation: (state, action: PayloadAction<Conversation>) => {
      state.list.unshift(action.payload);
    },
    updateConversationLocal: (
      state,
      action: PayloadAction<{ id: string; data: Partial<Conversation> }>
    ) => {
      const index = state.list.findIndex((c) => c.id === action.payload.id);
      if (index !== -1) {
        state.list[index] = { ...state.list[index], ...action.payload.data };
      }
    },
    removeConversation: (state, action: PayloadAction<string>) => {
      state.list = state.list.filter((c) => c.id !== action.payload);
    },
    clearError: (state) => {
      state.error = null;
    },
  },
  extraReducers: (builder) => {
    // Fetch conversations
    builder
      .addCase(fetchConversations.pending, (state) => {
        state.isLoading = true;
        state.error = null;
      })
      .addCase(fetchConversations.fulfilled, (state, action) => {
        state.isLoading = false;
        state.list = action.payload;
      })
      .addCase(fetchConversations.rejected, (state, action) => {
        state.isLoading = false;
        state.error = action.payload as string;
      });

    // Create conversation
    builder
      .addCase(createConversation.pending, (state) => {
        state.isLoading = true;
        state.error = null;
      })
      .addCase(createConversation.fulfilled, (state, action) => {
        state.isLoading = false;
        state.list.unshift(action.payload);
      })
      .addCase(createConversation.rejected, (state, action) => {
        state.isLoading = false;
        state.error = action.payload as string;
      });

    // Update conversation
    builder
      .addCase(updateConversation.fulfilled, (state, action) => {
        const index = state.list.findIndex((c) => c.id === action.payload.id);
        if (index !== -1) {
          state.list[index] = action.payload;
        }
      })
      .addCase(updateConversation.rejected, (state, action) => {
        state.error = action.payload as string;
      });

    // Delete conversation
    builder
      .addCase(deleteConversation.fulfilled, (state, action) => {
        state.list = state.list.filter((c) => c.id !== action.payload);
      })
      .addCase(deleteConversation.rejected, (state, action) => {
        state.error = action.payload as string;
      });
  },
});

export const {
  addConversation,
  updateConversationLocal,
  removeConversation,
  clearError,
} = conversationSlice.actions;

export default conversationSlice.reducer;
