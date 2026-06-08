import { describe, expect, it } from 'vitest';
import { frameIDs, normalizeFrames } from './frameModel';

describe('frameModel', () => {
  it('normalizes backend FrameItem ID into dnd id', () => {
    const frames = normalizeFrames([
      { ID: 'a', Name: 'a.bmp' },
      { ID: 'b', Name: 'b.bmp' },
    ]);

    expect(frames).toEqual([
      { ID: 'a', id: 'a', Name: 'a.bmp' },
      { ID: 'b', id: 'b', Name: 'b.bmp' },
    ]);
  });

  it('preserves dnd id during drag events', () => {
    const frames = normalizeFrames([
      { id: 'b', ID: 'b', Name: 'b.bmp' },
      { id: 'a', ID: 'a', Name: 'a.bmp' },
    ]);

    expect(frameIDs(frames)).toEqual(['b', 'a']);
  });

  it('drops malformed drag placeholders without ids', () => {
    const frames = normalizeFrames([{ Name: 'missing-id.bmp' }, { id: 'a', Name: 'a.bmp' }]);

    expect(frames).toHaveLength(1);
    expect(frames[0].id).toBe('a');
  });
});

