export function normalizeFrame(frame) {
  const id = frame?.id || frame?.ID;
  if (!id) return null;
  return {
    ...frame,
    id,
    ID: frame.ID || id,
  };
}

export function normalizeFrames(nextFrames) {
  return (nextFrames || []).map(normalizeFrame).filter(Boolean);
}

export function frameIDs(items) {
  return (items || []).map((frame) => frame.id || frame.ID).filter(Boolean);
}

