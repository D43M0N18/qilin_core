import React from 'react';
import { useSelector } from 'react-redux';
import { RootState } from '../../store';
import { useChat } from '../../hooks/useChat';
import Message from './Message';
import ChatInput from './ChatInput';
import ImagePreview from './ImagePreview';

interface ChatContainerProps {
  conversationId: string;
}

const ChatContainer: React.FC<ChatContainerProps> = ({ conversationId }) => {
  const { messages, isTyping, sendMessage, startTyping, retryMessage, regenerateResponse } = useChat({ conversationId });
  const { user } = useSelector((state: RootState) => state.auth);

  return (
    <div className="chat-container" style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      <div className="messages-list" style={{ flex: 1, overflowY: 'auto', padding: 16 }}>
        {messages.map((msg) => (
          <Message key={msg.id} message={msg} user={user} onRetry={retryMessage} />
        ))}
        {isTyping && (
          <Message
            message={{
              id: 'typing',
              conversation_id: conversationId,
              role: 'assistant',
              content: 'Typing...',
              is_streaming: true,
              created_at: new Date().toISOString(),
              updated_at: new Date().toISOString(),
            }}
            user={user}
            isTyping
          />
        )}
      </div>
      <ImagePreview />
      <ChatInput
        onSend={sendMessage}
        onTyping={startTyping}
        disabled={false}
        conversationId={conversationId}
      />
    </div>
  );
};

export default ChatContainer;
