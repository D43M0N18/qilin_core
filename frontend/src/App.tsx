import React, { useEffect, useState } from 'react';
import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import { useDispatch, useSelector } from 'react-redux';
import { RootState } from './store';
import { fetchProfile } from './store/slices/authSlice';
import { fetchConversations, createConversation, deleteConversation, updateConversation } from './store/slices/conversationSlice';
import { logout } from './store/slices/authSlice';
import { retryVideoGeneration } from './store/slices/videoSlice';

// Components
import Sidebar from './components/Sidebar/Sidebar';
import ChatContainer from './components/Chat/ChatContainer';
import VideoCanvas from './components/Canvas/VideoCanvas';
import Login from './components/Auth/Login';
import Register from './components/Auth/Register';
import Loading from './components/Common/Loading';

// Styles
import './styles/global.css';

const App: React.FC = () => {
  const dispatch = useDispatch();
  const { isAuthenticated, isLoading: authLoading, user } = useSelector(
    (state: RootState) => state.auth
  );
  const { list: conversations } = useSelector((state: RootState) => state.conversations);
  const { currentVideo, isGenerating } = useSelector((state: RootState) => state.videos);

  const [isSidebarOpen, setIsSidebarOpen] = useState(true);
  const [currentConversationId, setCurrentConversationId] = useState<string | null>(null);
  const [isInitialized, setIsInitialized] = useState(false);

  // Initialize app
  useEffect(() => {
    const initializeApp = async () => {
      if (isAuthenticated) {
        try {
          await dispatch(fetchProfile()).unwrap();
          await dispatch(fetchConversations()).unwrap();
        } catch (error) {
          console.error('Failed to initialize app:', error);
        }
      }
      setIsInitialized(true);
    };

    initializeApp();
  }, [isAuthenticated, dispatch]);

  // Set default conversation
  useEffect(() => {
    if (conversations.length > 0 && !currentConversationId) {
      setCurrentConversationId(conversations[0].id);
    }
  }, [conversations, currentConversationId]);

  // Handle new chat
  const handleNewChat = async () => {
    try {
      const result = await dispatch(
        createConversation({ title: 'New Conversation' })
      ).unwrap();
      setCurrentConversationId(result.id);
    } catch (error) {
      console.error('Failed to create conversation:', error);
    }
  };

  // Handle select conversation
  const handleSelectConversation = (id: string) => {
    setCurrentConversationId(id);
    if (window.innerWidth < 768) {
      setIsSidebarOpen(false);
    }
  };

  // Handle delete conversation
  const handleDeleteConversation = async (id: string) => {
    try {
      await dispatch(deleteConversation(id)).unwrap();
      if (currentConversationId === id) {
        setCurrentConversationId(conversations[0]?.id || null);
      }
    } catch (error) {
      console.error('Failed to delete conversation:', error);
    }
  };

  // Handle update conversation
  const handleUpdateConversation = async (id: string, title: string) => {
    try {
      await dispatch(updateConversation({ id, data: { title } })).unwrap();
    } catch (error) {
      console.error('Failed to update conversation:', error);
    }
  };

  // Handle logout
  const handleLogout = async () => {
    try {
      await dispatch(logout()).unwrap();
      setCurrentConversationId(null);
    } catch (error) {
      console.error('Failed to logout:', error);
    }
  };

  // Show loading while initializing
  if (authLoading || !isInitialized) {
    return <Loading />;
  }

  // Protected route wrapper
  const ProtectedRoute: React.FC<{ children: React.ReactNode }> = ({ children }) => {
    if (!isAuthenticated) {
      return <Navigate to="/login" replace />;
    }
    return <>{children}</>;
  };

  return (
    <Router>
      <Routes>
        {/* Auth Routes */}
        <Route
          path="/login"
          element={isAuthenticated ? <Navigate to="/" replace /> : <Login />}
        />
        <Route
          path="/register"
          element={isAuthenticated ? <Navigate to="/" replace /> : <Register />}
        />

        {/* Main App */}
        <Route
          path="/"
          element={
            <ProtectedRoute>
              <div className="app-container">
                {/* Sidebar */}
                <Sidebar
                  conversations={conversations}
                  currentConversationId={currentConversationId}
                  user={user!}
                  isOpen={isSidebarOpen}
                  onNewChat={handleNewChat}
                  onSelectConversation={handleSelectConversation}
                  onDeleteConversation={handleDeleteConversation}
                  onUpdateConversation={handleUpdateConversation}
                  onLogout={handleLogout}
                  onToggle={() => setIsSidebarOpen(!isSidebarOpen)}
                />

                {/* Main Content */}
                <div className="main-content">
                  {currentConversationId ? (
                    <ChatContainer conversationId={currentConversationId} />
                  ) : (
                    <div className="empty-chat-state">
                      <h2>Welcome to UGC Ad Platform</h2>
                      <p>Start a new conversation to create your first ad video</p>
                      <button className="start-button" onClick={handleNewChat}>
                        Start New Chat
                      </button>
                    </div>
                  )}
                </div>

                {/* Video Canvas */}
                <div className="video-canvas-sidebar">
                  <VideoCanvas
                    video={currentVideo}
                    isGenerating={isGenerating}
                    onRegenerate={() => {
                      if (currentVideo) {
                        dispatch(retryVideoGeneration(currentVideo.id));
                      }
                    }}
                    onDownload={() => {
                      console.log('Download video');
                    }}
                  />
                </div>
              </div>
            </ProtectedRoute>
          }
        />

        {/* Catch all */}
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </Router>
  );
};

export default App;
