package pathindex

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func touch(t *testing.T, root, rel string) {
	t.Helper()
	p := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, nil, 0o644); err != nil {
		t.Fatal(err)
	}
}

func paths(es []Entry) []string {
	out := make([]string, len(es))
	for i, e := range es {
		out[i] = e.Path
	}
	return out
}

func buildFixture(t *testing.T) *Snapshot {
	t.Helper()
	root := t.TempDir()
	for _, rel := range []string{
		"paper/siggraph2027/inventory/park2025magnet.pdf",
		"paper/siggraph2027/inventory/smith2024.pdf",
		"notes/parking/readme.md",
		"spark.md",
		".git/park-hidden",
		"studies/.cache/park.bin",
	} {
		touch(t, root, rel)
	}
	if err := os.Symlink("paper/siggraph2027", filepath.Join(root, "sg")); err != nil {
		t.Fatal(err)
	}
	return Build([]Root{{Mount: "/ws", Dir: root}})
}

func TestSearchFindsAnyPathFragment(t *testing.T) {
	s := buildFixture(t)
	got := paths(s.Search("PARK", 10, nil))
	want := []string{"/ws/spark.md", "/ws/notes/parking", "/ws/paper/siggraph2027/inventory/park2025magnet.pdf", "/ws/notes/parking/readme.md"}
	// name hits first; among them a name starting with the token wins, then shorter paths
	if !slices.Contains(got, "/ws/paper/siggraph2027/inventory/park2025magnet.pdf") {
		t.Fatalf("park2025magnet.pdf not found: %v", got)
	}
	if got[0] != "/ws/notes/parking" || got[1] != "/ws/paper/siggraph2027/inventory/park2025magnet.pdf" {
		t.Fatalf("order = %v, want names starting with park first", got)
	}
	if len(got) != len(want) {
		t.Fatalf("got %v, want the %d entries of %v", got, len(want), want)
	}
	for _, p := range got {
		if strings.Contains(p, "/.") {
			t.Fatalf("hidden entry returned: %s", p)
		}
	}
}

func TestSearchNeedsEveryToken(t *testing.T) {
	s := buildFixture(t)
	got := paths(s.Search("inventory pdf smith", 10, nil))
	if !slices.Equal(got, []string{"/ws/paper/siggraph2027/inventory/smith2024.pdf"}) {
		t.Fatalf("got %v", got)
	}
}

func TestSymlinkedDirIsListedNotFollowed(t *testing.T) {
	s := buildFixture(t)
	got := s.Search("sg", 10, nil)
	var link *Entry
	for i := range got {
		if got[i].Path == "/ws/sg" {
			link = &got[i]
		}
		if strings.HasPrefix(got[i].Path, "/ws/sg/") {
			t.Fatalf("followed the symlink: %s", got[i].Path)
		}
	}
	if link == nil || !link.IsDir {
		t.Fatalf("symlink to a dir should be listed as a dir: %+v", got)
	}
}

func TestSearchAppliesAcceptAndLimit(t *testing.T) {
	s := buildFixture(t)
	got := s.Search("pdf", 1, func(e Entry) bool { return !strings.Contains(e.Path, "park") })
	if !slices.Equal(paths(got), []string{"/ws/paper/siggraph2027/inventory/smith2024.pdf"}) {
		t.Fatalf("got %v", paths(got))
	}
}
