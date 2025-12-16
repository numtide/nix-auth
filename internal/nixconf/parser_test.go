package nixconf

import (
	"os"
	"path/filepath"
	"testing"
)

//nolint:gocognit // Comprehensive test suite requires multiple test cases
func TestParser(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "minimal-parser-test-*")
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = os.RemoveAll(tmpDir) }()

	t.Run("preserves exact formatting", func(t *testing.T) {
		content := `# Comment
foo = bar  # inline comment
  # indented comment
  baz = qux  
!include optional.conf
`
		path := filepath.Join(tmpDir, "format.conf")

		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}

		parser := NewParser()

		config, err := parser.ParseFile(path)
		if err != nil {
			t.Fatal(err)
		}

		// Check settings parsed correctly
		if config.Settings["foo"] != "bar" {
			t.Errorf("foo setting wrong: %q", config.Settings["foo"])
		}

		if config.Settings["baz"] != "qux" {
			t.Errorf("baz setting wrong: %q", config.Settings["baz"])
		}

		// Check includes tracked
		if !config.HasInclude("optional.conf") {
			t.Error("!include not tracked")
		}

		// Verify raw lines preserved exactly
		expectedLines := []string{
			"# Comment",
			"foo = bar  # inline comment",
			"  # indented comment",
			"  baz = qux  ",
			"!include optional.conf",
		}

		if len(config.Lines) != len(expectedLines) {
			t.Fatalf("expected %d lines, got %d", len(expectedLines), len(config.Lines))
		}

		for i, expected := range expectedLines {
			if config.Lines[i].Raw != expected {
				t.Errorf("line %d: expected %q, got %q", i, expected, config.Lines[i].Raw)
			}
		}
	})

	t.Run("parses included files", func(t *testing.T) {
		// Create main file
		mainContent := `foo = main
include sub.conf`
		mainPath := filepath.Join(tmpDir, "main.conf")

		if err := os.WriteFile(mainPath, []byte(mainContent), 0o600); err != nil {
			t.Fatal(err)
		}

		// Create included file
		subContent := `bar = included`
		subPath := filepath.Join(tmpDir, "sub.conf")

		if err := os.WriteFile(subPath, []byte(subContent), 0o600); err != nil {
			t.Fatal(err)
		}

		parser := NewParser()

		config, err := parser.ParseFile(mainPath)
		if err != nil {
			t.Fatal(err)
		}

		// Check both settings are available
		if config.Settings["foo"] != "main" {
			t.Error("main file setting not found")
		}

		if config.Settings["bar"] != "included" {
			t.Error("included file setting not found")
		}

		// Check line sources
		var mainLines, subLines int

		for _, line := range config.Lines {
			switch line.SourceFile {
			case mainPath:
				mainLines++
			case subPath:
				subLines++
			}
		}

		if mainLines != 2 {
			t.Errorf("expected 2 lines from main file, got %d", mainLines)
		}

		if subLines != 1 {
			t.Errorf("expected 1 line from sub file, got %d", subLines)
		}
	})

	t.Run("finds setting lines", func(t *testing.T) {
		content := `foo = first
bar = value
foo = second  # This should win`

		path := filepath.Join(tmpDir, "find.conf")
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}

		parser := NewParser()

		config, err := parser.ParseFile(path)
		if err != nil {
			t.Fatal(err)
		}

		// Should find the last occurrence
		line := config.FindSettingLine("foo")
		if line == nil {
			t.Fatal("setting line not found")
		}

		if line.Value != "second" {
			t.Errorf("wrong line found: %q", line.Value)
		}

		if line.LineNum != 3 {
			t.Errorf("wrong line number: %d", line.LineNum)
		}
	})
}
