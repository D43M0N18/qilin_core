import { describe, it, expect } from 'vitest';

export const formatBytes = (bytes: number): string => {
  if (bytes === 0) return '0 Bytes';
  const k = 1024;
  const sizes = ['Bytes', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i];
};

describe('Utility Functions', () => {
  describe('formatBytes', () => {
    it('formats 0 bytes correctly', () => {
      expect(formatBytes(0)).toBe('0 Bytes');
    });
    it('formats bytes correctly', () => {
      expect(formatBytes(500)).toBe('500 Bytes');
    });
    it('formats kilobytes correctly', () => {
      expect(formatBytes(1024)).toBe('1 KB');
    });
    it('formats megabytes correctly', () => {
      expect(formatBytes(1024 * 1024)).toBe('1 MB');
    });
    it('formats gigabytes correctly', () => {
      expect(formatBytes(1024 * 1024 * 1024)).toBe('1 GB');
    });
  });
});
