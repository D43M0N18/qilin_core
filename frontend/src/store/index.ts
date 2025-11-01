import { configureStore } from '@reduxjs/toolkit';
import authReducer from './slices/authSlice';
import chatReducer from './slices/chatSlice';
import conversationReducer from './slices/conversationSlice';
import videoReducer from './slices/videoSlice';

export const store = configureStore({
  reducer: {
    auth: authReducer,
    chat: chatReducer,
    conversations: conversationReducer,
    videos: videoReducer,
  },
  middleware: (getDefaultMiddleware) =>
    getDefaultMiddleware({
      serializableCheck: {
        // Ignore these action types
        ignoredActions: ['chat/addMessage', 'chat/updateMessage'],
        // Ignore these field paths in all actions
        ignoredActionPaths: ['payload.timestamp', 'payload.created_at'],
        // Ignore these paths in the state
        ignoredPaths: ['chat.messages', 'videos.list'],
      },
    }),
});

export type RootState = ReturnType<typeof store.getState>;
export type AppDispatch = typeof store.dispatch;
