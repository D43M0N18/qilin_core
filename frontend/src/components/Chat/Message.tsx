import React, { useState, useEffect, useRef } from 'react';
import ReactMarkdown from 'react-markdown';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { oneDark, oneLight } from 'react-syntax-highlighter/dist/esm/styles/prism';
import { Copy, Check, RotateCw, User, Bot, Image as ImageIcon } from 'lucide-react';
import { format } from 'date-fns';
import ImagePreview from './ImagePreview';
import './Message.module.css';

interface Attachment {
  id: string;
  file_name: string;
  url: string;
  thumbnail_url?: string;
  file_type: string;
  is_image: boolean;
}

interface MessageProps {
  id: string;
  role: 'user' | 'assistant' | 'system';
  content: string;
  attachments?: Attachment[];
  isStreaming?: boolean;
  timestamp: string;
  onRetry?: () => void;
  onCopy?: (content: string) => void;
  error?: boolean;
}

const Message: React.FC<MessageProps> = ({
  id,
  role,
  content,
  attachments = [],
  isStreaming = false,
  timestamp,
  onRetry,
  onCopy,
  error = false,
}) => {
  const [copied, setCopied] = useState(false);
  const [showActions, setShowActions] = useState(false);
  const [displayedContent, setDisplayedContent] = useState('');
  const messageRef = useRef<HTMLDivElement>(null);
  const isDarkMode = document.documentElement.classList.contains('dark');

  // Streaming effect
  useEffect(() => {
    if (isStreaming && content) {
      setDisplayedContent(content);
    } else if (!isStreaming) {
      setDisplayedContent(content);
    }
  }, [content, isStreaming]);

  // Auto-scroll during streaming
  useEffect(() => {
    if (isStreaming && messageRef.current) {
      messageRef.current.scrollIntoView({ behavior: 'smooth', block: 'end' });
    }
  }, [displayedContent, isStreaming]);

  // Handle copy
  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(content);
      setCopied(true);
      onCopy?.(content);
      setTimeout(() => setCopied(false), 2000);
    } catch (error) {
      console.error('Failed to copy:', error);
    }
  };

  // Format timestamp
  const formattedTime = format(new Date(timestamp), 'HH:mm');

  const isUser = role === 'user';
  const isAssistant = role === 'assistant';

  return (
    <div
      ref={messageRef}
      className={`message ${role} ${error ? 'error' : ''}`}
      onMouseEnter={() => setShowActions(true)}
      onMouseLeave={() => setShowActions(false)}
    >
      <div className="message-container">
        {/* Avatar */}
        <div className="message-avatar">
          {isUser ? (
            <div className="avatar user-avatar">
              <User size={20} />
            </div>
          ) : (
            <div className="avatar assistant-avatar">
              <Bot size={20} />
            </div>
          )}
        </div>

        {/* Content */}
        <div className="message-content">
          {/* Role Label */}
          <div className="message-header">
            <span className="role-label">
              {isUser ? 'You' : 'Assistant'}
            </span>
            <span className="timestamp">{formattedTime}</span>
          </div>

          {/* Attachments */}
          {attachments.length > 0 && (
            <div className="message-attachments">
              {attachments.map((attachment) => (
                <div key={attachment.id} className="attachment">
                  {attachment.is_image ? (
                    <ImagePreview
                      url={attachment.thumbnail_url || attachment.url}
                      alt={attachment.file_name}
                      fullUrl={attachment.url}
                    />
                  ) : (
                    <div className="file-attachment">
                      <ImageIcon size={20} />
                      <span>{attachment.file_name}</span>
                    </div>
                  )}
                </div>
              ))}
            </div>
          )}

          {/* Text Content */}
          {displayedContent && (
            <div className="message-text">
              {isUser ? (
                <p>{displayedContent}</p>
              ) : (
                <ReactMarkdown
                  components={{
                    code({ node, inline, className, children, ...props }) {
                      const match = /language-(\w+)/.exec(className || '');
                      const codeString = String(children).replace(/\n$/, '');
                      
                      return !inline && match ? (
                        <div className="code-block">
                          <div className="code-header">
                            <span className="code-language">{match[1]}</span>
                            <button
                              className="code-copy-button"
                              onClick={() => {
                                navigator.clipboard.writeText(codeString);
                                setCopied(true);
                                setTimeout(() => setCopied(false), 2000);
                              }}
                            >
                              {copied ? <Check size={14} /> : <Copy size={14} />}
                              {copied ? 'Copied!' : 'Copy'}
                            </button>
                          </div>
                          <SyntaxHighlighter
                            style={isDarkMode ? oneDark : oneLight}
                            language={match[1]}
                            PreTag="div"
                            {...props}
                          >
                            {codeString}
                          </SyntaxHighlighter>
                        </div>
                      ) : (
                        <code className={className} {...props}>
                          {children}
                        </code>
                      );
                    },
                    a({ node, href, children, ...props }) {
                      return (
                        <a href={href} target="_blank" rel="noopener noreferrer" {...props}>
                          {children}
                        </a>
                      );
                    },
                  }}
                >
                  {displayedContent}
                </ReactMarkdown>
              )}

              {/* Streaming cursor */}
              {isStreaming && <span className="streaming-cursor">▋</span>}
            </div>
          )}

          {/* Error Message */}
          {error && (
            <div className="error-message">
              <span>Failed to send message</span>
              {onRetry && (
                <button className="retry-button" onClick={onRetry}>
                  <RotateCw size={14} />
                  Retry
                </button>
              )}
            </div>
          )}

          {/* Actions */}
          {showActions && !isStreaming && isAssistant && (
            <div className="message-actions">
              <button
                className="action-button"
                onClick={handleCopy}
                title="Copy message"
              >
                {copied ? <Check size={16} /> : <Copy size={16} />}
                {copied ? 'Copied' : 'Copy'}
              </button>
              {onRetry && (
                <button
                  className="action-button"
                  onClick={onRetry}
                  title="Regenerate response"
                >
                  <RotateCw size={16} />
                  Regenerate
                </button>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default Message;
