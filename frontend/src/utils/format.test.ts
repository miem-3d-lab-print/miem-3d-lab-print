import { describe, expect, it } from 'vitest';

import { formatBytes, formatDate, hoursToHuman } from './format';

describe('format utilities', () => {
  it('formats byte values', () => {
    expect(formatBytes(512)).toBe('512 Б');
    expect(formatBytes(1536)).toBe('1.5 КБ');
    expect(formatBytes(2 * 1024 * 1024)).toBe('2.0 МБ');
  });

  it('returns a fallback for an invalid date', () => {
    expect(formatDate('not-a-date')).toBe('—');
  });

  it('formats durations', () => {
    expect(hoursToHuman(null)).toBe('—');
    expect(hoursToHuman(1.5)).toBe('2 ч');
  });
});
