package discover

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIgnoreStack_Disabled(t *testing.T) {
	stack := NewIgnoreStack(false, []string{".gitignore"})

	// Create temp directory structure
	tmpDir := t.TempDir()
	testDir := filepath.Join(tmpDir, "test")
	if err := os.Mkdir(testDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create .gitignore that would normally ignore "ignored"
	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	if err := os.WriteFile(gitignorePath, []byte("ignored\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// When disabled, Enter should not load ignore files
	if err := stack.Enter(tmpDir, 0); err != nil {
		t.Fatal(err)
	}

	// Should not ignore anything when disabled
	ignoredPath := filepath.Join(tmpDir, "ignored")
	if stack.ShouldIgnore(ignoredPath, true) {
		t.Error("Expected path not to be ignored when stack is disabled")
	}
}

func TestIgnoreStack_BasicIgnore(t *testing.T) {
	stack := NewIgnoreStack(true, []string{".gitignore"})

	// Create temp directory structure
	tmpDir := t.TempDir()
	ignoredDir := filepath.Join(tmpDir, "ignored")
	normalDir := filepath.Join(tmpDir, "normal")

	for _, dir := range []string{ignoredDir, normalDir} {
		if err := os.Mkdir(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}

	// Create .gitignore
	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	if err := os.WriteFile(gitignorePath, []byte("ignored\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Enter directory - should load .gitignore
	if err := stack.Enter(tmpDir, 0); err != nil {
		t.Fatal(err)
	}

	// Check that "ignored" is ignored
	if !stack.ShouldIgnore(ignoredDir, true) {
		t.Error("Expected 'ignored' directory to be ignored")
	}

	// Check that "normal" is not ignored
	if stack.ShouldIgnore(normalDir, true) {
		t.Error("Expected 'normal' directory not to be ignored")
	}
}

func TestIgnoreStack_DirectoryWithSlash(t *testing.T) {
	stack := NewIgnoreStack(true, []string{".gitignore"})

	tmpDir := t.TempDir()
	ignoredDir := filepath.Join(tmpDir, "ignored-dir")

	if err := os.Mkdir(ignoredDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create .gitignore with trailing slash pattern
	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	if err := os.WriteFile(gitignorePath, []byte("ignored-dir/\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := stack.Enter(tmpDir, 0); err != nil {
		t.Fatal(err)
	}

	// Should match directory with trailing slash pattern
	if !stack.ShouldIgnore(ignoredDir, true) {
		t.Error("Expected directory to match pattern with trailing slash")
	}
}

func TestIgnoreStack_WildcardPattern(t *testing.T) {
	stack := NewIgnoreStack(true, []string{".gitignore"})

	tmpDir := t.TempDir()
	dirs := []string{"project1", "project2", "other"}

	for _, dir := range dirs {
		dirPath := filepath.Join(tmpDir, dir)
		if err := os.Mkdir(dirPath, 0755); err != nil {
			t.Fatal(err)
		}
	}

	// Create .gitignore with wildcard
	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	if err := os.WriteFile(gitignorePath, []byte("project*\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := stack.Enter(tmpDir, 0); err != nil {
		t.Fatal(err)
	}

	// Check wildcard matches
	if !stack.ShouldIgnore(filepath.Join(tmpDir, "project1"), true) {
		t.Error("Expected 'project1' to be ignored by wildcard")
	}
	if !stack.ShouldIgnore(filepath.Join(tmpDir, "project2"), true) {
		t.Error("Expected 'project2' to be ignored by wildcard")
	}
	if stack.ShouldIgnore(filepath.Join(tmpDir, "other"), true) {
		t.Error("Expected 'other' not to be ignored")
	}
}

func TestIgnoreStack_MultipleIgnoreFiles(t *testing.T) {
	stack := NewIgnoreStack(true, []string{".gitignore", ".ignore"})

	tmpDir := t.TempDir()
	dir1 := filepath.Join(tmpDir, "from-gitignore")
	dir2 := filepath.Join(tmpDir, "from-ignore")

	for _, dir := range []string{dir1, dir2} {
		if err := os.Mkdir(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}

	// Create both .gitignore and .ignore
	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	if err := os.WriteFile(gitignorePath, []byte("from-gitignore\n"), 0644); err != nil {
		t.Fatal(err)
	}

	ignorePath := filepath.Join(tmpDir, ".ignore")
	if err := os.WriteFile(ignorePath, []byte("from-ignore\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := stack.Enter(tmpDir, 0); err != nil {
		t.Fatal(err)
	}

	// Both should be ignored
	if !stack.ShouldIgnore(dir1, true) {
		t.Error("Expected directory from .gitignore to be ignored")
	}
	if !stack.ShouldIgnore(dir2, true) {
		t.Error("Expected directory from .ignore to be ignored")
	}
}

func TestIgnoreStack_HierarchicalIgnore(t *testing.T) {
	stack := NewIgnoreStack(true, []string{".gitignore"})

	// Create directory hierarchy
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	ignoredInRoot := filepath.Join(tmpDir, "root-ignored")
	ignoredInSub := filepath.Join(subDir, "sub-ignored")
	normalInSub := filepath.Join(subDir, "normal")

	for _, dir := range []string{ignoredInRoot, ignoredInSub, normalInSub} {
		if err := os.Mkdir(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}

	// Create root .gitignore
	rootGitignore := filepath.Join(tmpDir, ".gitignore")
	if err := os.WriteFile(rootGitignore, []byte("root-ignored\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create subdirectory .gitignore
	subGitignore := filepath.Join(subDir, ".gitignore")
	if err := os.WriteFile(subGitignore, []byte("sub-ignored\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Enter root
	if err := stack.Enter(tmpDir, 0); err != nil {
		t.Fatal(err)
	}

	// Check root-level ignore
	if !stack.ShouldIgnore(ignoredInRoot, true) {
		t.Error("Expected root-ignored to be ignored at root level")
	}

	// Enter subdirectory
	if err := stack.Enter(subDir, 1); err != nil {
		t.Fatal(err)
	}

	// Check subdirectory-level ignore
	if !stack.ShouldIgnore(ignoredInSub, true) {
		t.Error("Expected sub-ignored to be ignored in subdirectory")
	}

	// Check that normal is not ignored
	if stack.ShouldIgnore(normalInSub, true) {
		t.Error("Expected normal directory not to be ignored")
	}

	// Leave subdirectory
	stack.Leave(1)

	// After leaving, subdirectory rules should no longer apply
	// (though we can't test this directly without re-checking)
}

func TestIgnoreStack_Leave(t *testing.T) {
	stack := NewIgnoreStack(true, []string{".gitignore"})

	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create .gitignore in subdirectory
	subGitignore := filepath.Join(subDir, ".gitignore")
	if err := os.WriteFile(subGitignore, []byte("ignored\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Enter subdirectory
	if err := stack.Enter(subDir, 1); err != nil {
		t.Fatal(err)
	}

	// Verify we have entries in the stack
	if len(stack.stack) == 0 {
		t.Error("Expected stack to have entries after entering directory with .gitignore")
	}

	// Leave subdirectory (depth 1) - should pop all entries at depth >= 1
	stack.Leave(1)

	// Stack should be empty now
	if len(stack.stack) != 0 {
		t.Errorf("Expected stack to be empty after leaving, got %d entries", len(stack.stack))
	}
}

func TestIgnoreStack_MalformedIgnoreFile(t *testing.T) {
	stack := NewIgnoreStack(true, []string{".gitignore"})

	tmpDir := t.TempDir()

	// Create a malformed .gitignore (empty is fine, but let's create one that would fail parsing)
	// The go-gitignore library is quite forgiving, so this mostly tests that we don't crash
	gitignorePath := filepath.Join(tmpDir, ".gitignore")
	if err := os.WriteFile(gitignorePath, []byte("\x00\x01\x02"), 0644); err != nil {
		t.Fatal(err)
	}

	// Should not error even with malformed file
	if err := stack.Enter(tmpDir, 0); err != nil {
		t.Errorf("Expected no error with malformed ignore file, got: %v", err)
	}
}

func TestIgnoreStack_NonExistentIgnoreFile(t *testing.T) {
	stack := NewIgnoreStack(true, []string{".gitignore"})

	tmpDir := t.TempDir()

	// Enter directory without .gitignore
	if err := stack.Enter(tmpDir, 0); err != nil {
		t.Fatal(err)
	}

	// Stack should remain empty
	if len(stack.stack) != 0 {
		t.Error("Expected stack to be empty when no ignore files exist")
	}

	// Should not ignore anything
	testPath := filepath.Join(tmpDir, "anything")
	if stack.ShouldIgnore(testPath, true) {
		t.Error("Expected no paths to be ignored when no ignore files exist")
	}
}
