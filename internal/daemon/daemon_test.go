package daemon

import (
	"testing"
	"time"

	"github.com/archhaeondlg/aiusage/internal/adapter"
	_ "github.com/archhaeondlg/aiusage/internal/adapter/all"
)

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{30 * time.Second, "30s"},
		{5 * time.Minute, "5m0s"},
		{90 * time.Minute, "1h30m"},
		{2*time.Hour + 5*time.Minute, "2h5m"},
		{0, "0s"},
		{1 * time.Minute, "1m0s"},
		{61 * time.Second, "1m1s"},
	}
	for _, tc := range tests {
		got := formatDuration(tc.d)
		if got != tc.want {
			t.Errorf("formatDuration(%v) = %q, want %q", tc.d, got, tc.want)
		}
	}
}

func TestAgentDisplayNamesCoverAll(t *testing.T) {
	registered := adapter.AllAdapters()
	for _, a := range registered {
		name := a.Name()
		if _, ok := agentDisplayNames[name]; !ok {
			t.Errorf("agent %q missing from agentDisplayNames map", name)
		}
	}
}

func TestAgentRegistryReturnsAll(t *testing.T) {
	entries := agentRegistry()
	registered := adapter.AllAdapters()
	if len(entries) != len(registered) {
		t.Errorf("agentRegistry returned %d entries, want %d", len(entries), len(registered))
	}
}

func TestAgentRegistryDisplayNames(t *testing.T) {
	entries := agentRegistry()
	for _, e := range entries {
		if e.displayName == "" {
			t.Errorf("agent %q has empty display name", e.adapter.Name())
		}
	}
}
