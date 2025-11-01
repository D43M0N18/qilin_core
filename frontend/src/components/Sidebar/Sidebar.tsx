import React, { useState } from 'react';
import {
  Plus,
  MessageSquare,
  Trash2,
  Edit2,
  Check,
  X,
  Search,
  User,
  Settings,
  LogOut,
  ChevronDown,
  Menu,
} from 'lucide-react';
import { format, isToday, isYesterday, isThisWeek, isThisMonth } from 'date-fns';
import './Sidebar.module.css';

interface Conversation {
  id: string;
  title: string;
  preview: string;
  created_at: string;
  updated_at: string;
  message_count: number;
}

interface User {
  id: string;
  email: string;
  full_name: string;
  avatar_url?: string;
  plan: string;
}

interface SidebarProps {
  conversations: Conversation[];
  currentConversationId: string | null;
  user: User;
  isOpen: boolean;
  onNewChat: () => void;
  onSelectConversation: (id: string) => void;
  onDeleteConversation: (id: string) => void;
  onUpdateConversation: (id: string, title: string) => void;
  onLogout: () => void;
  onToggle: () => void;
}

const Sidebar: React.FC<SidebarProps> = ({
  conversations,
  currentConversationId,
  user,
  isOpen,
  onNewChat,
  onSelectConversation,
  onDeleteConversation,
  onUpdateConversation,
  onLogout,
  onToggle,
}) => {
  const [searchQuery, setSearchQuery] = useState('');
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editingTitle, setEditingTitle] = useState('');
  const [showProfileMenu, setShowProfileMenu] = useState(false);

  // Filter conversations
  const filteredConversations = conversations.filter((conv) =>
    conv.title.toLowerCase().includes(searchQuery.toLowerCase()) ||
    conv.preview.toLowerCase().includes(searchQuery.toLowerCase())
  );

  // Group conversations by date
  const groupedConversations = {
    today: filteredConversations.filter((c) => isToday(new Date(c.updated_at))),
    yesterday: filteredConversations.filter((c) => isYesterday(new Date(c.updated_at))),
    lastWeek: filteredConversations.filter(
      (c) =>
        isThisWeek(new Date(c.updated_at)) &&
        !isToday(new Date(c.updated_at)) &&
        !isYesterday(new Date(c.updated_at))
    ),
    lastMonth: filteredConversations.filter(
      (c) =>
        isThisMonth(new Date(c.updated_at)) &&
        !isThisWeek(new Date(c.updated_at))
    ),
    older: filteredConversations.filter((c) => !isThisMonth(new Date(c.updated_at))),
  };

  // Start editing
  const handleStartEdit = (id: string, currentTitle: string) => {
    setEditingId(id);
    setEditingTitle(currentTitle);
  };

  // Save edit
  const handleSaveEdit = () => {
    if (editingId && editingTitle.trim()) {
      onUpdateConversation(editingId, editingTitle.trim());
      setEditingId(null);
      setEditingTitle('');
    }
  };

  // Cancel edit
  const handleCancelEdit = () => {
    setEditingId(null);
    setEditingTitle('');
  };

  // Render conversation item
  const renderConversation = (conv: Conversation) => {
    const isActive = conv.id === currentConversationId;
    const isEditing = editingId === conv.id;

    return (
      <div
        key={conv.id}
        className={`conversation-item ${isActive ? 'active' : ''}`}
        onClick={() => !isEditing && onSelectConversation(conv.id)}
      >
        <div className="conversation-icon">
          <MessageSquare size={18} />
        </div>

        <div className="conversation-content">
          {isEditing ? (
            <input
              type="text"
              className="edit-input"
              value={editingTitle}
              onChange={(e) => setEditingTitle(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') handleSaveEdit();
                if (e.key === 'Escape') handleCancelEdit();
              }}
              onClick={(e) => e.stopPropagation()}
              autoFocus
            />
          ) : (
            <>
              <div className="conversation-title">{conv.title}</div>
              <div className="conversation-preview">{conv.preview}</div>
            </>
          )}
        </div>

        <div className="conversation-actions">
          {isEditing ? (
            <>
              <button
                className="action-btn save"
                onClick={(e) => {
                  e.stopPropagation();
                  handleSaveEdit();
                }}
              >
                <Check size={14} />
              </button>
              <button
                className="action-btn cancel"
                onClick={(e) => {
                  e.stopPropagation();
                  handleCancelEdit();
                }}
              >
                <X size={14} />
              </button>
            </>
          ) : (
            <>
              <button
                className="action-btn edit"
                onClick={(e) => {
                  e.stopPropagation();
                  handleStartEdit(conv.id, conv.title);
                }}
                title="Rename"
              >
                <Edit2 size={14} />
              </button>
              <button
                className="action-btn delete"
                onClick={(e) => {
                  e.stopPropagation();
                  if (confirm('Delete this conversation?')) {
                    onDeleteConversation(conv.id);
                  }
                }}
                title="Delete"
              >
                <Trash2 size={14} />
              </button>
            </>
          )}
        </div>
      </div>
    );
  };

  // Render conversation group
  const renderGroup = (title: string, conversations: Conversation[]) => {
    if (conversations.length === 0) return null;

    return (
      <div className="conversation-group">
        <div className="group-title">{title}</div>
        {conversations.map(renderConversation)}
      </div>
    );
  };

  return (
    <>
      {/* Mobile Toggle Button */}
      <button className="mobile-toggle" onClick={onToggle}>
        <Menu size={24} />
      </button>

      {/* Sidebar */}
      <div className={`sidebar ${isOpen ? 'open' : ''}`}>
        {/* Header */}
        <div className="sidebar-header">
          <button className="new-chat-button" onClick={onNewChat}>
            <Plus size={20} />
            <span>New Chat</span>
          </button>
        </div>

        {/* Search */}
        <div className="search-container">
          <Search size={18} className="search-icon" />
          <input
            type="text"
            className="search-input"
            placeholder="Search conversations..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
          />
        </div>

        {/* Conversations List */}
        <div className="conversations-list">
          {filteredConversations.length === 0 ? (
            <div className="empty-state">
              <MessageSquare size={48} />
              <p>No conversations yet</p>
              <button className="start-button" onClick={onNewChat}>
                Start a new chat
              </button>
            </div>
          ) : (
            <>
              {renderGroup('Today', groupedConversations.today)}
              {renderGroup('Yesterday', groupedConversations.yesterday)}
              {renderGroup('Previous 7 Days', groupedConversations.lastWeek)}
              {renderGroup('Previous 30 Days', groupedConversations.lastMonth)}
              {renderGroup('Older', groupedConversations.older)}
            </>
          )}
        </div>

        {/* Profile Section */}
        <div className="profile-section">
          <div
            className="profile-trigger"
            onClick={() => setShowProfileMenu(!showProfileMenu)}
          >
            <div className="profile-avatar">
              {user.avatar_url ? (
                <img src={user.avatar_url} alt={user.full_name} />
              ) : (
                <User size={20} />
              )}
            </div>
            <div className="profile-info">
              <div className="profile-name">{user.full_name}</div>
              <div className="profile-plan">{user.plan} plan</div>
            </div>
            <ChevronDown
              size={20}
              className={`profile-chevron ${showProfileMenu ? 'open' : ''}`}
            />
          </div>

          {/* Profile Menu */}
          {showProfileMenu && (
            <div className="profile-menu">
              <button className="menu-item">
                <User size={18} />
                <span>Profile</span>
              </button>
              <button className="menu-item">
                <Settings size={18} />
                <span>Settings</span>
              </button>
              <div className="menu-divider" />
              <button className="menu-item logout" onClick={onLogout}>
                <LogOut size={18} />
                <span>Log out</span>
              </button>
            </div>
          )}
        </div>
      </div>

      {/* Mobile Overlay */}
      {isOpen && <div className="sidebar-overlay" onClick={onToggle} />}
    </>
  );
};

export default Sidebar;
