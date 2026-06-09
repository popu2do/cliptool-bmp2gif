package session

import (
	"crypto/md5"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"cliptool/internal/applog"
	"cliptool/internal/core"
)

type FrameItem struct {
	ID           string
	Path         string
	Name         string
	ThumbDataURL string
	Width        int
	Height       int
	Format       string
}

type AddResult struct {
	Frames  []FrameItem
	Added   int
	Skipped int
	Message string
}

type FrameStore struct {
	mu     sync.Mutex
	frames []FrameItem
	seen   map[string]struct{}
}

func NewFrameStore() *FrameStore {
	return &FrameStore{
		seen: make(map[string]struct{}),
	}
}

func (s *FrameStore) Frames() []FrameItem {
	s.mu.Lock()
	defer s.mu.Unlock()
	return copyFrames(s.frames)
}

func (s *FrameStore) Paths() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	paths := make([]string, 0, len(s.frames))
	for _, frame := range s.frames {
		paths = append(paths, frame.Path)
	}
	return paths
}

func (s *FrameStore) AddPaths(paths []string) AddResult {
	s.mu.Lock()
	defer s.mu.Unlock()

	applog.Debugf("开始追加路径到帧列表: input=%d current=%d", len(paths), len(s.frames))
	added := 0
	skipped := 0

	for _, rawPath := range paths {
		normalizedPath, err := normalizePath(rawPath)
		if err != nil {
			applog.Warnf("跳过无法规范化的路径: rawPath=%q err=%v", rawPath, err)
			skipped++
			continue
		}
		key := seenKey(normalizedPath)
		if _, exists := s.seen[key]; exists {
			applog.Debugf("跳过重复图片: path=%q", normalizedPath)
			skipped++
			continue
		}

		img, format, err := core.LoadImage(normalizedPath)
		if err != nil {
			applog.Warnf("跳过读取失败图片: path=%q err=%v", normalizedPath, err)
			skipped++
			continue
		}

		thumbDataURL, err := core.ThumbnailDataURL(img)
		if err != nil {
			applog.Warnf("跳过缩略图生成失败图片: path=%q err=%v", normalizedPath, err)
			skipped++
			continue
		}

		bounds := img.Bounds()
		s.frames = append(s.frames, FrameItem{
			ID:           pathID(normalizedPath),
			Path:         normalizedPath,
			Name:         filepath.Base(normalizedPath),
			ThumbDataURL: thumbDataURL,
			Width:        bounds.Dx(),
			Height:       bounds.Dy(),
			Format:       format,
		})
		s.seen[key] = struct{}{}
		added++
		applog.Debugf("已追加图片帧: path=%q format=%q width=%d height=%d id=%q", normalizedPath, format, bounds.Dx(), bounds.Dy(), pathID(normalizedPath))
	}

	applog.Debugf("追加路径到帧列表完成: added=%d skipped=%d total=%d", added, skipped, len(s.frames))
	return AddResult{
		Frames:  copyFrames(s.frames),
		Added:   added,
		Skipped: skipped,
		Message: addMessage(added, skipped),
	}
}

func (s *FrameStore) Remove(id string) []FrameItem {
	s.mu.Lock()
	defer s.mu.Unlock()

	nextFrames := make([]FrameItem, 0, len(s.frames))
	for _, frame := range s.frames {
		if frame.ID == id {
			delete(s.seen, seenKey(frame.Path))
			continue
		}
		nextFrames = append(nextFrames, frame)
	}
	s.frames = nextFrames
	return copyFrames(s.frames)
}

func (s *FrameStore) Reorder(ids []string) []FrameItem {
	s.mu.Lock()
	defer s.mu.Unlock()

	byID := make(map[string]FrameItem, len(s.frames))
	for _, frame := range s.frames {
		byID[frame.ID] = frame
	}

	nextFrames := make([]FrameItem, 0, len(s.frames))
	used := make(map[string]struct{}, len(s.frames))
	for _, id := range ids {
		frame, ok := byID[id]
		if !ok {
			continue
		}
		nextFrames = append(nextFrames, frame)
		used[id] = struct{}{}
	}

	for _, frame := range s.frames {
		if _, ok := used[frame.ID]; !ok {
			nextFrames = append(nextFrames, frame)
		}
	}

	s.frames = nextFrames
	return copyFrames(s.frames)
}

func (s *FrameStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.frames = nil
	s.seen = make(map[string]struct{})
}

func normalizePath(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.Clean(absPath), nil
}

func seenKey(path string) string {
	normalized := filepath.Clean(path)
	if runtime.GOOS == "windows" {
		return strings.ToLower(normalized)
	}
	return normalized
}

func pathID(path string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(seenKey(path))))
}

func copyFrames(frames []FrameItem) []FrameItem {
	copied := make([]FrameItem, len(frames))
	copy(copied, frames)
	return copied
}

func addMessage(added, skipped int) string {
	if added == 0 && skipped == 0 {
		return "未发现可用图片"
	}
	if added == 0 {
		return fmt.Sprintf("未追加新图片，跳过 %d 项", skipped)
	}
	if skipped == 0 {
		return fmt.Sprintf("已追加 %d 张图片", added)
	}
	return fmt.Sprintf("已追加 %d 张图片，跳过 %d 项", added, skipped)
}
