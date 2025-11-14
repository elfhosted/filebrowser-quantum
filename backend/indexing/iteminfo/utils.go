package iteminfo

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/gtsteffaniak/filebrowser/backend/ffmpeg"
	"github.com/gtsteffaniak/go-logger/logger"
)

type SubtitleTrack struct {
	Name     string `json:"name"`               // filename for external, or descriptive name for embedded
	Language string `json:"language,omitempty"` // language code
	Title    string `json:"title,omitempty"`    // title/description
	Index    *int   `json:"index,omitempty"`    // stream index for embedded subtitles (nil for external)
	Codec    string `json:"codec,omitempty"`    // codec name for embedded subtitles
	IsFile   bool   `json:"isFile"`             // true for external files, false for embedded
}

type FFProbeOutput struct {
	Streams []struct {
		Index       int               `json:"index"`
		CodecType   string            `json:"codec_type"`
		CodecName   string            `json:"codec_name"`
		Tags        map[string]string `json:"tags,omitempty"`
		Disposition map[string]int    `json:"disposition,omitempty"`
	} `json:"streams"`
}

// detects subtitles for video files.
func (i *ExtendedFileInfo) DetectSubtitles(parentInfo *FileInfo) {
	if !strings.HasPrefix(i.Type, "video") {
		logger.Debug("subtitles are not supported for this file : " + i.Name)
		return
	}
	// Use unified subtitle detection that finds both embedded and external
	parentDir := filepath.Dir(i.RealPath)
	i.Subtitles = ffmpeg.DetectAllSubtitles(i.RealPath, parentDir, i.ModTime)
}

// LoadSubtitleContent loads the actual content for all detected subtitle tracks
func (i *ExtendedFileInfo) LoadSubtitleContent() error {
	return ffmpeg.LoadAllSubtitleContent(i.RealPath, i.Subtitles, i.ModTime)
}

func (info *FileInfo) SortItems() {
	sort.Slice(info.Folders, func(i, j int) bool {
		nameWithoutExt := strings.Split(info.Folders[i].Name, ".")[0]
		nameWithoutExt2 := strings.Split(info.Folders[j].Name, ".")[0]
		// Convert strings to integers for numeric sorting if both are numeric
		numI, errI := strconv.Atoi(nameWithoutExt)
		numJ, errJ := strconv.Atoi(nameWithoutExt2)
		if errI == nil && errJ == nil {
			return numI < numJ
		}
		// Fallback to case-insensitive lexicographical sorting
		return strings.ToLower(info.Folders[i].Name) < strings.ToLower(info.Folders[j].Name)
	})
	sort.Slice(info.Files, func(i, j int) bool {
		nameWithoutExt := strings.Split(info.Files[i].Name, ".")[0]
		nameWithoutExt2 := strings.Split(info.Files[j].Name, ".")[0]
		// Convert strings to integers for numeric sorting if both are numeric
		numI, errI := strconv.Atoi(nameWithoutExt)
		numJ, errJ := strconv.Atoi(nameWithoutExt2)
		if errI == nil && errJ == nil {
			return numI < numJ
		}
		// Fallback to case-insensitive lexicographical sorting
		return strings.ToLower(info.Files[i].Name) < strings.ToLower(info.Files[j].Name)
	})
}

// ResolveSymlinks resolves symlinks in the given path and returns
// the final resolved path, whether it's a directory (considering bundle logic), and any error.
func ResolveSymlinks(path string) (string, bool, error) {
	// Prefer using EvalSymlinks which handles cycles and relative targets robustly
	resolved, err := filepath.EvalSymlinks(path)
	if err == nil {
		info, statErr := os.Lstat(resolved)
		if statErr != nil {
			return resolved, false, fmt.Errorf("could not stat resolved path: %s, %v", resolved, statErr)
		}
		return resolved, IsDirectory(info), nil
	}

	// Fallback: manual resolution with cycle guard
	seen := make(map[string]struct{})
	current := path
	for i := 0; i < 64; i++ { // hard cap to prevent infinite loops
		if _, ok := seen[current]; ok {
			return current, false, fmt.Errorf("symlink cycle detected at: %s", current)
		}
		seen[current] = struct{}{}

		info, lerr := os.Lstat(current)
		if lerr != nil {
			return current, false, fmt.Errorf("could not stat path: %s, %v", current, lerr)
		}
		if info.Mode()&os.ModeSymlink == 0 {
			return current, IsDirectory(info), nil
		}
		target, rerr := os.Readlink(current)
		if rerr != nil {
			return current, false, fmt.Errorf("could not read symlink: %s, %v", current, rerr)
		}
		// Resolve the symlink's target relative to its directory
		current = filepath.Join(filepath.Dir(current), target)
	}
	return current, false, fmt.Errorf("too many symlink hops resolving: %s", path)
}
