import { createSlice, createAsyncThunk, PayloadAction } from '@reduxjs/toolkit';
import { api } from '../../services/api';

interface Video {
  id: string;
  conversation_id: string;
  status: string;
  progress: number;
  url?: string;
  thumbnail_url?: string;
  duration?: number;
  product_name: string;
  character_type: string;
  character_name?: string;
  script?: string;
  error_message?: string;
  created_at: string;
  updated_at: string;
}

interface VideoState {
  list: Video[];
  currentVideo: Video | null;
  isGenerating: boolean;
  isLoading: boolean;
  error: string | null;
}

const initialState: VideoState = {
  list: [],
  currentVideo: null,
  isGenerating: false,
  isLoading: false,
  error: null,
};

// Async thunks
export const generateVideo = createAsyncThunk(
  'videos/generate',
  async (
    data: {
      conversation_id: string;
      product_name: string;
      product_desc: string;
      product_image_url: string;
      character_type?: string;
      duration?: number;
    },
    { rejectWithValue }
  ) => {
    try {
      const response = await api.post('/videos/generate', data);
      return response.data.data;
    } catch (error: any) {
      return rejectWithValue(
        error.response?.data?.error || 'Failed to generate video'
      );
    }
  }
);

export const fetchVideo = createAsyncThunk(
  'videos/fetch',
  async (id: string, { rejectWithValue }) => {
    try {
      const response = await api.get(`/videos/${id}`);
      return response.data.data;
    } catch (error: any) {
      return rejectWithValue(error.response?.data?.error || 'Failed to fetch video');
    }
  }
);

export const fetchUserVideos = createAsyncThunk(
  'videos/fetchAll',
  async (filters?: { conversation_id?: string; status?: string }, { rejectWithValue }) => {
    try {
      const params = new URLSearchParams();
      if (filters?.conversation_id) {
        params.append('conversation_id', filters.conversation_id);
      }
      if (filters?.status) {
        params.append('status', filters.status);
      }

      const response = await api.get(`/videos?${params.toString()}`);
      return response.data.data;
    } catch (error: any) {
      return rejectWithValue(error.response?.data?.error || 'Failed to fetch videos');
    }
  }
);

export const deleteVideo = createAsyncThunk(
  'videos/delete',
  async (id: string, { rejectWithValue }) => {
    try {
      await api.delete(`/videos/${id}`);
      return id;
    } catch (error: any) {
      return rejectWithValue(error.response?.data?.error || 'Failed to delete video');
    }
  }
);

export const retryVideoGeneration = createAsyncThunk(
  'videos/retry',
  async (id: string, { rejectWithValue }) => {
    try {
      const response = await api.post(`/videos/${id}/retry`);
      return response.data.data;
    } catch (error: any) {
      return rejectWithValue(
        error.response?.data?.error || 'Failed to retry video generation'
      );
    }
  }
);

// Slice
const videoSlice = createSlice({
  name: 'videos',
  initialState,
  reducers: {
    setCurrentVideo: (state, action: PayloadAction<Video | null>) => {
      state.currentVideo = action.payload;
    },

    updateVideoProgress: (
      state,
      action: PayloadAction<{
        videoId: string;
        status: string;
        progress: number;
        video?: Video;
      }>
    ) => {
      // Update in list
      const index = state.list.findIndex((v) => v.id === action.payload.videoId);
      if (index !== -1) {
        state.list[index].status = action.payload.status;
        state.list[index].progress = action.payload.progress;
        if (action.payload.video) {
          state.list[index] = action.payload.video;
        }
      }

      // Update current video
      if (state.currentVideo?.id === action.payload.videoId) {
        state.currentVideo.status = action.payload.status;
        state.currentVideo.progress = action.payload.progress;
        if (action.payload.video) {
          state.currentVideo = action.payload.video;
        }
      }
    },

    addVideo: (state, action: PayloadAction<Video>) => {
      state.list.unshift(action.payload);
      if (!state.currentVideo) {
        state.currentVideo = action.payload;
      }
    },

    removeVideo: (state, action: PayloadAction<string>) => {
      state.list = state.list.filter((v) => v.id !== action.payload);
      if (state.currentVideo?.id === action.payload) {
        state.currentVideo = null;
      }
    },

    clearError: (state) => {
      state.error = null;
    },
  },
  extraReducers: (builder) => {
    // Generate video
    builder
      .addCase(generateVideo.pending, (state) => {
        state.isGenerating = true;
        state.error = null;
      })
      .addCase(generateVideo.fulfilled, (state, action) => {
        state.isGenerating = false;
        state.list.unshift(action.payload);
        state.currentVideo = action.payload;
      })
      .addCase(generateVideo.rejected, (state, action) => {
        state.isGenerating = false;
        state.error = action.payload as string;
      });

    // Fetch video
    builder
      .addCase(fetchVideo.pending, (state) => {
        state.isLoading = true;
      })
      .addCase(fetchVideo.fulfilled, (state, action) => {
        state.isLoading = false;
        state.currentVideo = action.payload;

        // Update in list if exists
        const index = state.list.findIndex((v) => v.id === action.payload.id);
        if (index !== -1) {
          state.list[index] = action.payload;
        }
      })
      .addCase(fetchVideo.rejected, (state, action) => {
        state.isLoading = false;
        state.error = action.payload as string;
      });

    // Fetch user videos
    builder
      .addCase(fetchUserVideos.pending, (state) => {
        state.isLoading = true;
      })
      .addCase(fetchUserVideos.fulfilled, (state, action) => {
        state.isLoading = false;
        state.list = action.payload;
      })
      .addCase(fetchUserVideos.rejected, (state, action) => {
        state.isLoading = false;
        state.error = action.payload as string;
      });

    // Delete video
    builder
      .addCase(deleteVideo.fulfilled, (state, action) => {
        state.list = state.list.filter((v) => v.id !== action.payload);
        if (state.currentVideo?.id === action.payload) {
          state.currentVideo = null;
        }
      })
      .addCase(deleteVideo.rejected, (state, action) => {
        state.error = action.payload as string;
      });

    // Retry video generation
    builder
      .addCase(retryVideoGeneration.pending, (state) => {
        state.isGenerating = true;
      })
      .addCase(retryVideoGeneration.fulfilled, (state, action) => {
        state.isGenerating = false;
        const index = state.list.findIndex((v) => v.id === action.payload.id);
        if (index !== -1) {
          state.list[index] = action.payload;
        }
        if (state.currentVideo?.id === action.payload.id) {
          state.currentVideo = action.payload;
        }
      })
      .addCase(retryVideoGeneration.rejected, (state, action) => {
        state.isGenerating = false;
        state.error = action.payload as string;
      });
  },
});

export const {
  setCurrentVideo,
  updateVideoProgress,
  addVideo,
  removeVideo,
  clearError,
} = videoSlice.actions;

export default videoSlice.reducer;
