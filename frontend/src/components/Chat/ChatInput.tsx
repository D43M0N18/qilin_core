import React, { useState, useRef, useEffect, useCallback } from 'react';
import { Send, Paperclip, Image as ImageIcon, X, Loader2 } from 'lucide-react';
import { useFileUpload } from '../../hooks/useFileUpload';
import ImagePreview from './ImagePreview';
import './ChatInput.module.css';

interface ChatInputProps {
  onSendMessage: (content: string, attachmentIds: string[]) => void;
  onFileUpload: (files: File[]) => void;
  disabled?: boolean;
  conversationId: string;
  placeholder?: string;
}

interface UploadedFile {
  id: string;
  name: string;
  url: string;
  thumbnailUrl?: string;
  type: string;
  size: number;
}

const ChatInput: React.FC<ChatInputProps> = ({
  onSendMessage,
  onFileUpload,
  disabled = false,
  conversationId,
  placeholder = "Message...",
}) => {
  const [content, setContent] = useState('');
  const [attachments, setAttachments] = useState<UploadedFile[]>([]);
  const [isDragging, setIsDragging] = useState(false);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const { uploadFile, isUploading, progress } = useFileUpload();

  // Auto-resize textarea
  useEffect(() => {
    const textarea = textareaRef.current;
    if (!textarea) return;

    textarea.style.height = 'auto';
    const newHeight = Math.min(textarea.scrollHeight, 200); // Max 200px
    textarea.style.height = `${newHeight}px`;
  }, [content]);

  // Focus on mount
  useEffect(() => {
    textareaRef.current?.focus();
  }, [conversationId]);

  // Handle text change
  const handleTextChange = useCallback((e: React.ChangeEvent<HTMLTextAreaElement>) => {
    setContent(e.target.value);
  }, []);

  // Handle send
  const handleSend = useCallback(() => {
    if (!content.trim() && attachments.length === 0) return;
    if (disabled || isUploading) return;

    const attachmentIds = attachments.map(a => a.id);
    onSendMessage(content.trim(), attachmentIds);
    
    setContent('');
    setAttachments([]);
    
    // Reset textarea height
    if (textareaRef.current) {
      textareaRef.current.style.height = 'auto';
    }
  }, [content, attachments, disabled, isUploading, onSendMessage]);

  // Handle keyboard shortcuts
  const handleKeyDown = useCallback((e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    // Enter to send (without Shift)
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }

    // Ctrl/Cmd + Enter to send
    if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) {
      e.preventDefault();
      handleSend();
    }

    // Escape to clear
    if (e.key === 'Escape') {
      setContent('');
      setAttachments([]);
    }
  }, [handleSend]);

  // Handle file selection
  const handleFileSelect = useCallback(async (files: FileList | null) => {
    if (!files || files.length === 0) return;

    const fileArray = Array.from(files);
    
    // Validate files
    const validFiles = fileArray.filter(file => {
      const isImage = file.type.startsWith('image/');
      const isValidSize = file.size <= 50 * 1024 * 1024; // 50MB
      return isImage && isValidSize;
    });

    if (validFiles.length === 0) {
      alert('Please select valid image files (max 50MB each)');
      return;
    }

    // Upload files
    for (const file of validFiles) {
      try {
        const result = await uploadFile(file, conversationId);
        setAttachments(prev => [...prev, {
          id: result.id,
          name: result.file_name,
          url: result.url,
          thumbnailUrl: result.thumbnail_url,
          type: result.file_type,
          size: result.file_size,
        }]);
      } catch (error) {
        console.error('Failed to upload file:', error);
        alert(`Failed to upload ${file.name}`);
      }
    }

    onFileUpload(validFiles);
  }, [uploadFile, conversationId, onFileUpload]);

  // Handle drag and drop
  const handleDragEnter = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setIsDragging(true);
  }, []);

  const handleDragLeave = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setIsDragging(false);
  }, []);

  const handleDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
  }, []);

  const handleDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setIsDragging(false);

    const files = e.dataTransfer.files;
    handleFileSelect(files);
  }, [handleFileSelect]);

  // Handle paste
  const handlePaste = useCallback((e: React.ClipboardEvent) => {
    const items = e.clipboardData.items;
    const files: File[] = [];

    for (let i = 0; i < items.length; i++) {
      if (items[i].type.startsWith('image/')) {
        const file = items[i].getAsFile();
        if (file) files.push(file);
      }
    }

    if (files.length > 0) {
      e.preventDefault();
      const fileList = new DataTransfer();
      files.forEach(file => fileList.items.add(file));
      handleFileSelect(fileList.files);
    }
  }, [handleFileSelect]);

  // Remove attachment
  const handleRemoveAttachment = useCallback((id: string) => {
    setAttachments(prev => prev.filter(a => a.id !== id));
  }, []);

  // Open file picker
  const handleAttachClick = useCallback(() => {
    fileInputRef.current?.click();
  }, []);

  const canSend = (content.trim().length > 0 || attachments.length > 0) && !disabled && !isUploading;

  return (
    <div className="chat-input-container">
      {/* Attachment Previews */}
      {attachments.length > 0 && (
        <div className="attachments-preview">
          {attachments.map(attachment => (
            <div key={attachment.id} className="attachment-item">
              <ImagePreview
                url={attachment.thumbnailUrl || attachment.url}
                alt={attachment.name}
              />
              <button
                className="remove-attachment"
                onClick={() => handleRemoveAttachment(attachment.id)}
                aria-label="Remove attachment"
              >
                <X size={14} />
              </button>
            </div>
          ))}
        </div>
      )}

      {/* Drag Overlay */}
      {isDragging && (
        <div className="drag-overlay">
          <ImageIcon size={48} />
          <p>Drop images here</p>
        </div>
      )}

      {/* Input Area */}
      <div
        className={`input-wrapper ${isDragging ? 'dragging' : ''}`}
        onDragEnter={handleDragEnter}
        onDragLeave={handleDragLeave}
        onDragOver={handleDragOver}
        onDrop={handleDrop}
      >
        <div className="input-content">
          {/* Attach Button */}
          <button
            className="attach-button"
            onClick={handleAttachClick}
            disabled={disabled || isUploading}
            aria-label="Attach file"
            title="Attach file"
          >
            {isUploading ? (
              <Loader2 size={20} className="spinning" />
            ) : (
              <Paperclip size={20} />
            )}
          </button>

          {/* Hidden File Input */}
          <input
            ref={fileInputRef}
            type="file"
            accept="image/*"
            multiple
            style={{ display: 'none' }}
            onChange={(e) => handleFileSelect(e.target.files)}
          />

          {/* Textarea */}
          <textarea
            ref={textareaRef}
            className="message-textarea"
            value={content}
            onChange={handleTextChange}
            onKeyDown={handleKeyDown}
            onPaste={handlePaste}
            placeholder={placeholder}
            disabled={disabled}
            rows={1}
            maxLength={10000}
            aria-label="Message input"
          />

          {/* Send Button */}
          <button
            className={`send-button ${canSend ? 'active' : ''}`}
            onClick={handleSend}
            disabled={!canSend}
            aria-label="Send message"
            title="Send message (Enter)"
          >
            <Send size={20} />
          </button>
        </div>

        {/* Character Count (when approaching limit) */}
        {content.length > 9000 && (
          <div className="character-count">
            {content.length} / 10,000
          </div>
        )}

        {/* Upload Progress */}
        {isUploading && (
          <div className="upload-progress">
            <div className="progress-bar" style={{ width: `${progress}%` }} />
            <span className="progress-text">Uploading... {progress}%</span>
          </div>
        )}
      </div>

      {/* Helper Text */}
      <div className="input-helper-text">
        <span>Press <kbd>Enter</kbd> to send, <kbd>Shift + Enter</kbd> for new line</span>
      </div>
    </div>
  );
};

export default ChatInput;
