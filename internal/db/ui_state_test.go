package db_test

import (
	"testing"

	"github.com/OpenListTeam/OpenList/v4/internal/db"
)

func TestUIStateRoundTripPerUser(t *testing.T) {
	setupViewHistoryDB(t)
	if _, ok, err := db.GetUIState(1, "tabs.layout"); err != nil || ok {
		t.Fatalf("empty store: ok=%v err=%v", ok, err)
	}
	for _, v := range []string{`{"a":1}`, `{"a":2}`} {
		if err := db.SetUIState(1, "tabs.layout", v); err != nil {
			t.Fatalf("set %s: %v", v, err)
		}
	}
	if err := db.SetUIState(2, "tabs.layout", `{"other":true}`); err != nil {
		t.Fatalf("set user 2: %v", err)
	}
	if v, ok, err := db.GetUIState(1, "tabs.layout"); err != nil || !ok || v != `{"a":2}` {
		t.Fatalf("user 1 = %q ok=%v err=%v, want the latest value", v, ok, err)
	}
	if v, _, _ := db.GetUIState(2, "tabs.layout"); v != `{"other":true}` {
		t.Fatalf("user 2 = %q", v)
	}
}
