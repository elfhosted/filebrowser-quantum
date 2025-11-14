package iteminfo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveSymlinks_File(t *testing.T) {
	tmp := t.TempDir()
	file := filepath.Join(tmp, "file.txt")
	if err := os.WriteFile(file, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	link := filepath.Join(tmp, "file.link")
	if err := os.Symlink("file.txt", link); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	resolved, isDir, err := ResolveSymlinks(link)
	if err != nil {
		t.Fatalf("ResolveSymlinks error: %v", err)
	}
	// Normalize expected path for platform-specific differences (e.g., macOS /private prefix)
	expected, _ := filepath.EvalSymlinks(file)
	if resolved != expected {
		t.Fatalf("expected resolved=%s got %s", expected, resolved)
	}
	if isDir {
		t.Fatalf("expected isDir=false for file symlink")
	}
}

func TestResolveSymlinks_Directory(t *testing.T) {
	tmp := t.TempDir()
	dir := filepath.Join(tmp, "sub")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	link := filepath.Join(tmp, "dir.link")
	if err := os.Symlink("sub", link); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	resolved, isDir, err := ResolveSymlinks(link)
	if err != nil {
		t.Fatalf("ResolveSymlinks error: %v", err)
	}
	expected, _ := filepath.EvalSymlinks(dir)
	if resolved != expected {
		t.Fatalf("expected resolved=%s got %s", expected, resolved)
	}
	if !isDir {
		t.Fatalf("expected isDir=true for directory symlink")
	}
}

func TestResolveSymlinks_Cycle(t *testing.T) {
	tmp := t.TempDir()
	linkA := filepath.Join(tmp, "a")
	linkB := filepath.Join(tmp, "b")
	// Create a cycle: a -> b, b -> a
	if err := os.Symlink("b", linkA); err != nil {
		t.Fatalf("symlink a->b: %v", err)
	}
	if err := os.Symlink("a", linkB); err != nil {
		t.Fatalf("symlink b->a: %v", err)
	}

	_, _, err := ResolveSymlinks(linkA)
	if err == nil {
		t.Fatalf("expected error for symlink cycle, got nil")
	}
}
