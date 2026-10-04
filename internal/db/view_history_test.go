package db_test

import (
	"testing"
	"time"

	"github.com/OpenListTeam/OpenList/v4/internal/conf"
	"github.com/OpenListTeam/OpenList/v4/internal/db"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupViewHistoryDB(t *testing.T) {
	t.Helper()
	dB, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	conf.Conf = conf.DefaultConfig("data")
	db.Init(dB)
}

func historyPaths(t *testing.T, userID uint) []string {
	t.Helper()
	rows, err := db.ListViewHistory(userID, 0)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	paths := make([]string, len(rows))
	for i, r := range rows {
		paths[i] = r.Path
	}
	return paths
}

func equalPaths(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestViewHistoryReopenMovesToTop(t *testing.T) {
	setupViewHistoryDB(t)
	t0 := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	for i, p := range []string{"/a.pdf", "/b.png", "/a.pdf"} {
		if err := db.RecordViewHistory(1, p, t0.Add(time.Duration(i)*time.Minute), 10); err != nil {
			t.Fatalf("record %s: %v", p, err)
		}
	}
	if got, want := historyPaths(t, 1), []string{"/a.pdf", "/b.png"}; !equalPaths(got, want) {
		t.Fatalf("history = %v, want %v", got, want)
	}
	rows, _ := db.ListViewHistory(1, 1)
	if len(rows) != 1 || rows[0].Count != 2 {
		t.Fatalf("top row = %+v, want /a.pdf with count 2", rows)
	}
}

func TestViewHistoryKeepsNewest(t *testing.T) {
	setupViewHistoryDB(t)
	t0 := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	for i, p := range []string{"/1", "/2", "/3", "/4"} {
		if err := db.RecordViewHistory(1, p, t0.Add(time.Duration(i)*time.Minute), 2); err != nil {
			t.Fatalf("record %s: %v", p, err)
		}
	}
	if got, want := historyPaths(t, 1), []string{"/4", "/3"}; !equalPaths(got, want) {
		t.Fatalf("history = %v, want %v", got, want)
	}
}

func TestViewHistoryDeleteIsPerUser(t *testing.T) {
	setupViewHistoryDB(t)
	now := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	for _, rec := range []struct {
		user uint
		path string
	}{{1, "/x"}, {1, "/y"}, {2, "/x"}} {
		if err := db.RecordViewHistory(rec.user, rec.path, now, 10); err != nil {
			t.Fatalf("record: %v", err)
		}
	}
	if err := db.DeleteViewHistory(1, "/x"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if got, want := historyPaths(t, 1), []string{"/y"}; !equalPaths(got, want) {
		t.Fatalf("user 1 = %v, want %v", got, want)
	}
	if got, want := historyPaths(t, 2), []string{"/x"}; !equalPaths(got, want) {
		t.Fatalf("user 2 = %v, want %v", got, want)
	}
	if err := db.DeleteViewHistory(1, ""); err != nil {
		t.Fatalf("clear: %v", err)
	}
	if got := historyPaths(t, 1); len(got) != 0 {
		t.Fatalf("user 1 after clear = %v, want empty", got)
	}
	if got := historyPaths(t, 2); len(got) != 1 {
		t.Fatalf("user 2 after clearing user 1 = %v, want untouched", got)
	}
}
