package netrc

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func newTemp(t *testing.T) *File {
	t.Helper()

	return New(filepath.Join(t.TempDir(), "nix", "netrc"))
}

func TestSetCreatesFileWith0600(t *testing.T) {
	f := newTemp(t)

	if err := f.Set(Entry{Machine: "cache.example.com", Login: "alice", Password: "s3cret"}); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(f.Path())
	if err != nil {
		t.Fatal(err)
	}

	if info.Mode().Perm() != 0o600 {
		t.Errorf("mode = %o, want 600", info.Mode().Perm())
	}

	got, ok, _ := f.Get("cache.example.com")
	if !ok || got.Login != "alice" || got.Password != "s3cret" {
		t.Errorf("unexpected entry %+v ok=%v", got, ok)
	}
}

func TestSetReplacesAndPreservesOtherLines(t *testing.T) {
	f := newTemp(t)
	_ = os.MkdirAll(filepath.Dir(f.Path()), 0o755)

	orig := "# my comment\nmachine a.example login u1 password p1\nmachine b.example login u2 password p2\n"
	if err := os.WriteFile(f.Path(), []byte(orig), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := f.Set(Entry{Machine: "a.example", Login: "u9", Password: "p9"}); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(f.Path())
	want := "# my comment\nmachine a.example login u9 password p9\nmachine b.example login u2 password p2\n"

	if string(data) != want {
		t.Errorf("got:\n%s\nwant:\n%s", data, want)
	}
}

func TestSetIsIdempotent(t *testing.T) {
	f := newTemp(t)
	e := Entry{Machine: "a.example", Login: "u", Password: "p"}

	_ = f.Set(e)
	_ = f.Set(e)

	list, _ := f.List()
	if len(list) != 1 {
		t.Errorf("got %d entries, want 1", len(list))
	}
}

func TestSetValidation(t *testing.T) {
	f := newTemp(t)

	for _, e := range []Entry{
		{Machine: "", Login: "u", Password: "p"},
		{Machine: "h", Login: "", Password: "p"},
		{Machine: "h", Login: "u", Password: ""},
		{Machine: "h", Login: "u", Password: "has space"},
	} {
		if err := f.Set(e); err == nil {
			t.Errorf("expected error for %+v", e)
		}
	}
}

func TestMultiLineEntryRejected(t *testing.T) {
	f := newTemp(t)
	_ = os.MkdirAll(filepath.Dir(f.Path()), 0o755)
	_ = os.WriteFile(f.Path(), []byte("machine a.example\n  login u\n  password p\n"), 0o600)

	if err := f.Set(Entry{Machine: "a.example", Login: "x", Password: "y"}); !errors.Is(err, ErrMultiLineEntry) {
		t.Errorf("err = %v, want ErrMultiLineEntry", err)
	}

	if _, err := f.Remove("a.example"); !errors.Is(err, ErrMultiLineEntry) {
		t.Errorf("err = %v, want ErrMultiLineEntry", err)
	}
}

func TestRemove(t *testing.T) {
	f := newTemp(t)
	_ = f.Set(Entry{Machine: "a.example", Login: "u", Password: "p"})
	_ = f.Set(Entry{Machine: "b.example", Login: "u", Password: "p"})

	removed, err := f.Remove("a.example")
	if err != nil || !removed {
		t.Fatalf("removed=%v err=%v", removed, err)
	}

	if removed, _ := f.Remove("nope.example"); removed {
		t.Error("removing a missing machine should report false")
	}

	// Removing the last entry deletes the file.
	_, _ = f.Remove("b.example")

	if _, err := os.Stat(f.Path()); !os.IsNotExist(err) {
		t.Errorf("expected file to be removed, stat err = %v", err)
	}
}

func TestListMissingFile(t *testing.T) {
	list, err := newTemp(t).List()
	if err != nil || len(list) != 0 {
		t.Errorf("list=%v err=%v", list, err)
	}
}
