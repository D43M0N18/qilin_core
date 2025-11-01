import React, { useState } from 'react';
import { X, ZoomIn } from 'lucide-react';
import './ImagePreview.module.css';

interface ImagePreviewProps {
  url: string;
  alt: string;
  fullUrl?: string;
  onRemove?: () => void;
}

const ImagePreview: React.FC<ImagePreviewProps> = ({
  url,
  alt,
  fullUrl,
  onRemove,
}) => {
  const [isLightboxOpen, setIsLightboxOpen] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const [hasError, setHasError] = useState(false);

  const handleImageClick = () => {
    if (fullUrl) {
      setIsLightboxOpen(true);
    }
  };

  const handleCloseLightbox = () => {
    setIsLightboxOpen(false);
  };

  const handleImageLoad = () => {
    setIsLoading(false);
  };

  const handleImageError = () => {
    setIsLoading(false);
    setHasError(true);
  };

  return (
    <>
      <div className="image-preview-container">
        {isLoading && (
          <div className="image-loading">
            <div className="spinner" />
          </div>
        )}

        {hasError ? (
          <div className="image-error">
            <span>Failed to load image</span>
          </div>
        ) : (
          <>
            <img
              src={url}
              alt={alt}
              className={`preview-image ${isLoading ? 'loading' : ''}`}
              onClick={handleImageClick}
              onLoad={handleImageLoad}
              onError={handleImageError}
            />

            {!isLoading && fullUrl && (
              <div className="image-overlay">
                <button className="zoom-button" onClick={handleImageClick}>
                  <ZoomIn size={20} />
                </button>
              </div>
            )}
          </>
        )}

        {onRemove && (
          <button className="remove-button" onClick={onRemove}>
            <X size={16} />
          </button>
        )}
      </div>

      {/* Lightbox */}
      {isLightboxOpen && fullUrl && (
        <div className="lightbox-overlay" onClick={handleCloseLightbox}>
          <button className="lightbox-close" onClick={handleCloseLightbox}>
            <X size={24} />
          </button>
          <img
            src={fullUrl}
            alt={alt}
            className="lightbox-image"
            onClick={(e) => e.stopPropagation()}
          />
        </div>
      )}
    </>
  );
};

export default ImagePreview;
