// Package netrc manages entries in a netrc file used by Nix (via curl) for
// Basic Auth hosts such as private binary caches.
package netrc

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	filePermissions = 0o600
	dirPermissions  = 0o755
	// minEntryFields is "machine <host>".
	minEntryFields = 2
)

// ErrMultiLineEntry is returned when the target machine is defined across
// several lines, which this package does not rewrite.
var ErrMultiLineEntry = errors.New("netrc entry spans multiple lines; edit it manually")

// Entry is a single netrc machine entry.
type Entry struct {
	Machine  string
	Login    string
	Password string
}

// File is a netrc file on disk.
type File struct {
	path string
}

// New returns a File for the given path. A leading "~/" is expanded.
func New(path string) *File {
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			path = filepath.Join(home, path[2:])
		}
	}

	return &File{path: path}
}

// Path returns the file path.
func (f *File) Path() string { return f.path }

// Set adds or replaces the entry for e.Machine, preserving all other lines.
func (f *File) Set(e Entry) error {
	if e.Machine == "" || e.Login == "" || e.Password == "" {
		return errors.New("machine, login and password are required")
	}

	if strings.ContainsAny(e.Machine+e.Login+e.Password, " \t\r\n") {
		return errors.New("machine, login and password must not contain whitespace")
	}

	lines, err := f.readLines()
	if err != nil {
		return err
	}

	newLine := fmt.Sprintf("machine %s login %s password %s", e.Machine, e.Login, e.Password)
	out := make([]string, 0, len(lines)+1)
	replaced := false

	for _, l := range lines {
		m, _, kind := classify(l)
		if m != e.Machine {
			out = append(out, l)
			continue
		}

		if kind == kindPartial {
			return fmt.Errorf("%s: %w", e.Machine, ErrMultiLineEntry)
		}

		if !replaced {
			out = append(out, newLine)
			replaced = true
		}
	}

	if !replaced {
		out = append(out, newLine)
	}

	return f.write(out)
}

// Remove deletes the entry for machine. It reports whether one was removed.
func (f *File) Remove(machine string) (bool, error) {
	lines, err := f.readLines()
	if err != nil {
		return false, err
	}

	out := make([]string, 0, len(lines))
	removed := false

	for _, l := range lines {
		m, _, kind := classify(l)
		if m != machine {
			out = append(out, l)
			continue
		}

		if kind == kindPartial {
			return false, fmt.Errorf("%s: %w", machine, ErrMultiLineEntry)
		}

		removed = true
	}

	if !removed {
		return false, nil
	}

	if len(out) == 0 {
		if err := os.Remove(f.path); err != nil && !os.IsNotExist(err) {
			return false, err
		}

		return true, nil
	}

	return true, f.write(out)
}

// Get returns the entry for machine, if present on a single line.
func (f *File) Get(machine string) (Entry, bool, error) {
	entries, err := f.List()
	if err != nil {
		return Entry{}, false, err
	}

	for _, e := range entries {
		if e.Machine == machine {
			return e, true, nil
		}
	}

	return Entry{}, false, nil
}

// List returns all single-line entries sorted by machine.
func (f *File) List() ([]Entry, error) {
	lines, err := f.readLines()
	if err != nil {
		return nil, err
	}

	var entries []Entry

	for _, l := range lines {
		if m, e, kind := classify(l); m != "" && kind == kindFull {
			entries = append(entries, e)
		}
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].Machine < entries[j].Machine })

	return entries, nil
}

type lineKind int

const (
	kindOther   lineKind = iota
	kindFull             // machine + login + password on one line
	kindPartial          // machine without credentials (multi-line entry)
)

// classify extracts the machine from a line. Comments and unrelated lines
// return an empty machine.
func classify(line string) (string, Entry, lineKind) {
	if i := strings.IndexByte(line, '#'); i == 0 || (i > 0 && strings.TrimSpace(line[:i]) == "") {
		return "", Entry{}, kindOther
	}

	fields := strings.Fields(line)
	if len(fields) < minEntryFields || fields[0] != "machine" {
		return "", Entry{}, kindOther
	}

	e := Entry{Machine: fields[1]}

	for i := 2; i+1 < len(fields); i += 2 {
		switch fields[i] {
		case "login":
			e.Login = fields[i+1]
		case "password":
			e.Password = fields[i+1]
		}
	}

	if e.Login == "" || e.Password == "" {
		return e.Machine, e, kindPartial
	}

	return e.Machine, e, kindFull
}

func (f *File) readLines() ([]string, error) {
	data, err := os.ReadFile(f.path) //nolint:gosec // user-configured netrc path
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, err
	}

	text := strings.TrimRight(string(data), "\n")
	if text == "" {
		return nil, nil
	}

	return strings.Split(text, "\n"), nil
}

// write atomically replaces the file (temp file + rename) with mode 0600.
func (f *File) write(lines []string) error {
	dir := filepath.Dir(f.path)
	if err := os.MkdirAll(dir, dirPermissions); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".netrc-*")
	if err != nil {
		return err
	}

	defer func() { _ = os.Remove(tmp.Name()) }() // no-op after successful rename

	if err := tmp.Chmod(filePermissions); err != nil {
		_ = tmp.Close()
		return err
	}

	if _, err := tmp.WriteString(strings.Join(lines, "\n") + "\n"); err != nil {
		_ = tmp.Close()
		return err
	}

	if err := tmp.Close(); err != nil {
		return err
	}

	return os.Rename(tmp.Name(), f.path)
}
