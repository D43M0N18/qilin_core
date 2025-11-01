import React, { useState, useRef, useEffect } from 'react';
import {
  Play,
  Pause,
  Volume2,
  VolumeX,
  Maximize,
  Download,
  RotateCw,
  Loader2,
  CheckCircle,
  AlertCircle,
} from 'lucide-react';
import './Canvas.module.css';

interface Video {
  id: string;
  status: string;
  progress: number;
  url?: string;
  thumbnail_url?: string;
  duration?: number;
  product_name: string;
  character_type: string;
  error_message?: string;
}

interface VideoCanvasProps {
  video: Video | null;
  isGenerating: boolean;
  onRegenerate?: () => void;
  onDownload?: () => void;
}

const VideoCanvas: React.FC<VideoCanvasProps> = ({
  video,
  isGenerating,
  onRegenerate,
  onDownload,
}) => {
  const [isPlaying, setIsPlaying] = useState(false);
  const [isMuted, setIsMuted] = useState(false);
  const [currentTime, setCurrentTime] = useState(0);
  const [duration, setDuration] = useState(0);
  const [volume, setVolume] = useState(1);
  const [showControls, setShowControls] = useState(true);
  const videoRef = useRef<HTMLVideoElement>(null);
  const controlsTimeoutRef = useRef<NodeJS.Timeout>();

  // Reset when video changes
  useEffect(() => {
    if (video?.url && videoRef.current) {
      videoRef.current.load();
      setIsPlaying(false);
      setCurrentTime(0);
    }
  }, [video?.url]);

  // Hide controls after inactivity
  useEffect(() => {
    if (showControls) {
      if (controlsTimeoutRef.current) {
        clearTimeout(controlsTimeoutRef.current);
      }
      controlsTimeoutRef.current = setTimeout(() => {
        if (isPlaying) {
          setShowControls(false);
        }
      }, 3000);
    }
    return () => {
      if (controlsTimeoutRef.current) {
        clearTimeout(controlsTimeoutRef.current);
      }
    };
  }, [showControls, isPlaying]);

  // Play/Pause
  const togglePlay = () => {
    if (videoRef.current) {
      if (isPlaying) {
        videoRef.current.pause();
      } else {
        videoRef.current.play();
      }
      setIsPlaying(!isPlaying);
    }
  };

  // Mute/Unmute
  const toggleMute = () => {
    if (videoRef.current) {
      videoRef.current.muted = !isMuted;
      setIsMuted(!isMuted);
    }
  };

  // Volume change
  const handleVolumeChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const newVolume = parseFloat(e.target.value);
    setVolume(newVolume);
    if (videoRef.current) {
      videoRef.current.volume = newVolume;
      setIsMuted(newVolume === 0);
    }
  };

  // Time update
  const handleTimeUpdate = () => {
    if (videoRef.current) {
      setCurrentTime(videoRef.current.currentTime);
    }
  };

  // Loaded metadata
  const handleLoadedMetadata = () => {
    if (videoRef.current) {
      setDuration(videoRef.current.duration);
    }
  };

  // Seek
  const handleSeek = (e: React.ChangeEvent<HTMLInputElement>) => {
    const newTime = parseFloat(e.target.value);
    setCurrentTime(newTime);
    if (videoRef.current) {
      videoRef.current.currentTime = newTime;
    }
  };

  // Fullscreen
  const toggleFullscreen = () => {
    if (videoRef.current) {
      if (document.fullscreenElement) {
        document.exitFullscreen();
      } else {
        videoRef.current.requestFullscreen();
      }
    }
  };

  // Download video
  const handleDownload = () => {
    if (video?.url) {
      const link = document.createElement('a');
      link.href = video.url;
      link.download = `${video.product_name}_ad.mp4`;
      link.click();
      onDownload?.();
    }
  };

  // Format time
  const formatTime = (seconds: number) => {
    const mins = Math.floor(seconds / 60);
    const secs = Math.floor(seconds % 60);
    return `${mins}:${secs.toString().padStart(2, '0')}`;
  };

  // Render loading state
  if (isGenerating || (video && video.status !== 'completed' && video.status !== 'failed')) {
    return (
      <div className="video-canvas-container">
        <div className="generation-status">
          {video?.status === 'failed' ? (
            <>
              <AlertCircle size={48} className="status-icon error" />
              <h3>Generation Failed</h3>
              <p>{video.error_message || 'An error occurred during video generation'}</p>
              {onRegenerate && (
                <button className="retry-button" onClick={onRegenerate}>
                  <RotateCw size={20} />
                  Try Again
                </button>
              )}
            </>
          ) : (
            <>
              <Loader2 size={48} className="status-icon spinning" />
              <h3>Generating Video</h3>
              <p>
                {video?.status === 'analyzing' && 'Analyzing product image...'}
                {video?.status === 'generating' && 'Creating your ad...'}
                {video?.status === 'processing' && 'Processing video...'}
                {!video?.status && 'Starting generation...'}
              </p>
              {video && (
                <div className="progress-container">
                  <div className="progress-bar">
                    <div
                      className="progress-fill"
                      style={{ width: `${video.progress}%` }}
                    />
                  </div>
                  <span className="progress-text">{video.progress}%</span>
                </div>
              )}
            </>
          )}
        </div>
      </div>
    );
  }

  // Render empty state
  if (!video || !video.url) {
    return (
      <div className="video-canvas-container">
        <div className="empty-state">
          <Play size={64} className="empty-icon" />
          <h3>No Video Yet</h3>
          <p>
            Start a conversation and request a video advertisement to see it here.
          </p>
        </div>
      </div>
    );
  }

  // Render video player
  return (
    <div
      className="video-canvas-container"
      onMouseMove={() => setShowControls(true)}
      onMouseLeave={() => isPlaying && setShowControls(false)}
    >
      <div className="video-player-wrapper">
        {/* Video Element */}
        <video
          ref={videoRef}
          className="video-player"
          poster={video.thumbnail_url}
          onTimeUpdate={handleTimeUpdate}
          onLoadedMetadata={handleLoadedMetadata}
          onPlay={() => setIsPlaying(true)}
          onPause={() => setIsPlaying(false)}
          onClick={togglePlay}
        >
          <source src={video.url} type="video/mp4" />
          Your browser does not support the video tag.
        </video>

        {/* Play Overlay */}
        {!isPlaying && (
          <div className="play-overlay" onClick={togglePlay}>
            <button className="play-button-large">
              <Play size={64} />
            </button>
          </div>
        )}

        {/* Video Controls */}
        <div className={`video-controls ${showControls ? 'visible' : ''}`}>
          {/* Progress Bar */}
          <input
            type="range"
            className="progress-slider"
            min="0"
            max={duration || 0}
            value={currentTime}
            onChange={handleSeek}
          />

          {/* Control Buttons */}
          <div className="controls-row">
            <div className="controls-left">
              {/* Play/Pause */}
              <button className="control-button" onClick={togglePlay}>
                {isPlaying ? <Pause size={20} /> : <Play size={20} />}
              </button>

              {/* Time Display */}
              <span className="time-display">
                {formatTime(currentTime)} / {formatTime(duration)}
              </span>

              {/* Volume */}
              <div className="volume-control">
                <button className="control-button" onClick={toggleMute}>
                  {isMuted || volume === 0 ? (
                    <VolumeX size={20} />
                  ) : (
                    <Volume2 size={20} />
                  )}
                </button>
                <input
                  type="range"
                  className="volume-slider"
                  min="0"
                  max="1"
                  step="0.1"
                  value={volume}
                  onChange={handleVolumeChange}
                />
              </div>
            </div>

            <div className="controls-right">
              {/* Download */}
              <button
                className="control-button"
                onClick={handleDownload}
                title="Download video"
              >
                <Download size={20} />
              </button>

              {/* Regenerate */}
              {onRegenerate && (
                <button
                  className="control-button"
                  onClick={onRegenerate}
                  title="Regenerate video"
                >
                  <RotateCw size={20} />
                </button>
              )}

              {/* Fullscreen */}
              <button
                className="control-button"
                onClick={toggleFullscreen}
                title="Fullscreen"
              >
                <Maximize size={20} />
              </button>
            </div>
          </div>
        </div>

        {/* Video Info Badge */}
        <div className="video-info-badge">
          <CheckCircle size={16} />
          <span>{video.product_name}</span>
        </div>
      </div>
    </div>
  );
};

export default VideoCanvas;
