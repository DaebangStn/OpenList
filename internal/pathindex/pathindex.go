// Package pathindex keeps every path under the Local storages' root folders in
// memory, so the web UI can find a file by any fragment of its path without a
// search index build. Hidden entries (".name") are skipped and symlinked
// directories are not followed.
package pathindex

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	log "github.com/sirupsen/logrus"
)

// MaxAge is how old a snapshot may get before a query triggers a rebuild.
const MaxAge = time.Minute

// Root maps a local directory to the OpenList mount path that shows it.
type Root struct {
	Mount string
	Dir   string
}

type Entry struct {
	Path  string // OpenList path
	IsDir bool
	lower string
	name  int // offset of the base name in Path
}

type Snapshot struct {
	Entries []Entry
	Built   time.Time
}

// Build walks every root. Errors below a root (permissions, vanished files)
// skip that entry only.
func Build(roots []Root) *Snapshot {
	start := time.Now()
	var entries []Entry
	for _, r := range roots {
		_ = filepath.WalkDir(r.Dir, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				if d != nil && d.IsDir() {
					return fs.SkipDir
				}
				return nil
			}
			if p == r.Dir {
				return nil
			}
			if strings.HasPrefix(d.Name(), ".") {
				if d.IsDir() {
					return fs.SkipDir
				}
				return nil
			}
			rel, err := filepath.Rel(r.Dir, p)
			if err != nil {
				return nil
			}
			isDir := d.IsDir()
			if d.Type()&fs.ModeSymlink != 0 {
				if st, err := os.Stat(p); err == nil {
					isDir = st.IsDir()
				}
			}
			full := path.Join(r.Mount, filepath.ToSlash(rel))
			entries = append(entries, Entry{
				Path:  full,
				IsDir: isDir,
				lower: strings.ToLower(full),
				name:  strings.LastIndexByte(full, '/') + 1,
			})
			return nil
		})
	}
	log.Infof("pathindex: %d paths under %d roots in %s", len(entries), len(roots), time.Since(start).Round(time.Millisecond))
	return &Snapshot{Entries: entries, Built: time.Now()}
}

// Search returns up to limit entries whose path holds every whitespace
// separated token of q, case-insensitively, that accept lets through.
// Entries whose base name holds more tokens come first, then those whose
// name starts with the first token, then shorter paths.
func (s *Snapshot) Search(q string, limit int, accept func(Entry) bool) []Entry {
	tokens := strings.Fields(strings.ToLower(q))
	if len(tokens) == 0 || limit <= 0 {
		return nil
	}
	type hit struct {
		idx   int
		score int64
	}
	var hits []hit
	for i := range s.Entries {
		e := &s.Entries[i]
		inName := 0
		ok := true
		for _, t := range tokens {
			if !strings.Contains(e.lower, t) {
				ok = false
				break
			}
			if strings.Contains(e.lower[e.name:], t) {
				inName++
			}
		}
		if !ok {
			continue
		}
		var notPrefix int64 = 1
		if strings.HasPrefix(e.lower[e.name:], tokens[0]) {
			notPrefix = 0
		}
		score := int64(len(tokens)-inName)<<40 | notPrefix<<39 | int64(len(e.Path))
		hits = append(hits, hit{i, score})
	}
	slices.SortFunc(hits, func(a, b hit) int {
		if a.score != b.score {
			if a.score < b.score {
				return -1
			}
			return 1
		}
		return strings.Compare(s.Entries[a.idx].Path, s.Entries[b.idx].Path)
	})
	out := make([]Entry, 0, min(limit, len(hits)))
	for _, h := range hits {
		if accept == nil || accept(s.Entries[h.idx]) {
			out = append(out, s.Entries[h.idx])
			if len(out) == limit {
				break
			}
		}
	}
	return out
}

var (
	current  atomic.Pointer[Snapshot]
	building atomic.Bool
	firstMu  sync.Mutex
)

// Get returns the current snapshot. The first call builds it and waits; later
// calls on a snapshot older than MaxAge rebuild it in the background and
// return the old one meanwhile.
func Get(roots func() []Root) *Snapshot {
	if s := current.Load(); s != nil {
		if time.Since(s.Built) > MaxAge && building.CompareAndSwap(false, true) {
			go func() {
				defer building.Store(false)
				current.Store(Build(roots()))
			}()
		}
		return s
	}
	firstMu.Lock()
	defer firstMu.Unlock()
	if s := current.Load(); s != nil {
		return s
	}
	s := Build(roots())
	current.Store(s)
	return s
}
