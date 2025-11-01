import { useState, useCallback } from 'react';
import axios, { AxiosProgressEvent } from 'axios';
import { api } from '../services/api';

interface UploadResult {
  id: string;
  file_name: string;
  url: string;
  thumbnail_url?: string;
  file_type: string;
  file_size: number;
  width?: number;
  height?: number;
}

export const useFileUpload = () => {
  const [isUploading, setIsUploading] = useState(false);
  const [progress, setProgress] = useState(0);
  const [error, setError] = useState<string | null>(null);

  const uploadFile = useCallback(
    async (file: File, conversationId?: string): Promise<UploadResult> => {
      setIsUploading(true);
      setProgress(0);
      setError(null);

      const formData = new FormData();
      formData.append('file', file);
      if (conversationId) {
        formData.append('conversation_id', conversationId);
      }

      try {
        const response = await api.post<{ success: boolean; data: UploadResult }>(
          '/upload',
          formData,
          {
            headers: {
              'Content-Type': 'multipart/form-data',
            },
            onUploadProgress: (progressEvent: AxiosProgressEvent) => {
              const percentCompleted = progressEvent.total
                ? Math.round((progressEvent.loaded * 100) / progressEvent.total)
                : 0;
              setProgress(percentCompleted);
            },
          }
        );

        setIsUploading(false);
        setProgress(100);
        return response.data.data;
      } catch (err: any) {
        setIsUploading(false);
        setProgress(0);
        const errorMessage = err.response?.data?.error || 'Failed to upload file';
        setError(errorMessage);
        throw new Error(errorMessage);
      }
    },
    []
  );

  const uploadMultiple = useCallback(
    async (files: File[], conversationId?: string): Promise<UploadResult[]> => {
      setIsUploading(true);
      setProgress(0);
      setError(null);

      const formData = new FormData();
      files.forEach((file) => {
        formData.append('files', file);
      });
      if (conversationId) {
        formData.append('conversation_id', conversationId);
      }

      try {
        const response = await api.post<{
          success: boolean;
          data: UploadResult[];
          errors?: string[];
        }>('/upload/multiple', formData, {
          headers: {
            'Content-Type': 'multipart/form-data',
          },
          onUploadProgress: (progressEvent: AxiosProgressEvent) => {
            const percentCompleted = progressEvent.total
              ? Math.round((progressEvent.loaded * 100) / progressEvent.total)
              : 0;
            setProgress(percentCompleted);
          },
        });

        setIsUploading(false);
        setProgress(100);

        if (response.data.errors && response.data.errors.length > 0) {
          console.warn('Some files failed to upload:', response.data.errors);
        }

        return response.data.data;
      } catch (err: any) {
        setIsUploading(false);
        setProgress(0);
        const errorMessage = err.response?.data?.error || 'Failed to upload files';
        setError(errorMessage);
        throw new Error(errorMessage);
      }
    },
    []
  );

  const reset = useCallback(() => {
    setIsUploading(false);
    setProgress(0);
    setError(null);
  }, []);

  return {
    uploadFile,
    uploadMultiple,
    isUploading,
    progress,
    error,
    reset,
  };
};
