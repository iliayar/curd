package internal

import (
	"strings"
	"testing"

	_ "github.com/wraient/curd/internal/loadproviders"
)

func TestProviderMappingSearchStateNextProvider(t *testing.T) {
	t.Run("single provider hides next option", func(t *testing.T) {
		state := &providerMappingSearchState{allProviders: []string{"anineko"}}
		if got := state.nextProviderLabel(); got != "" {
			t.Fatalf("nextProviderLabel() = %q, want empty", got)
		}
		if state.advanceToNextProvider() {
			t.Fatal("advanceToNextProvider() = true, want false")
		}
	})

	t.Run("stacked search offers first provider", func(t *testing.T) {
		state := &providerMappingSearchState{allProviders: []string{"senshi", "anineko", "anipub"}}
		if got := state.nextProviderLabel(); got != "senshi" {
			t.Fatalf("nextProviderLabel() = %q, want senshi", got)
		}
	})

	t.Run("sequential advance walks stack", func(t *testing.T) {
		state := &providerMappingSearchState{allProviders: []string{"senshi", "anineko", "anipub"}}
		if !state.advanceToNextProvider() {
			t.Fatal("expected first advance to succeed")
		}
		if !state.sequential || state.providerIndex != 0 {
			t.Fatalf("expected sequential at index 0, got sequential=%v index=%d", state.sequential, state.providerIndex)
		}
		if got := state.nextProviderLabel(); got != "anineko" {
			t.Fatalf("nextProviderLabel() = %q, want anineko", got)
		}
		if !state.advanceToNextProvider() || state.providerIndex != 1 {
			t.Fatalf("expected index 1 after second advance, got %d", state.providerIndex)
		}
		if got := state.nextProviderLabel(); got != "anipub" {
			t.Fatalf("nextProviderLabel() = %q, want anipub", got)
		}
		if !state.advanceToNextProvider() || state.providerIndex != 2 {
			t.Fatalf("expected index 2 after third advance, got %d", state.providerIndex)
		}
		if got := state.nextProviderLabel(); got != "" {
			t.Fatalf("nextProviderLabel() = %q, want empty at end", got)
		}
		if state.advanceToNextProvider() {
			t.Fatal("expected final advance to fail")
		}
	})
}

func TestProviderNameFromSelectionUsesSequentialProvider(t *testing.T) {
	withAllProvidersEnabledForTest(t)
	config := &CurdConfig{Provider: `["senshi","anineko"]`}
	state := &providerMappingSearchState{
		allProviders:  []string{"senshi", "anineko"},
		sequential:    true,
		providerIndex: 1,
	}

	selected := SelectionOption{
		Key:   "frieren-beyond-journeys-end",
		Label: "Frieren: Beyond Journey's End",
		Title: "Frieren: Beyond Journey's End",
	}
	if got := providerNameFromSelection(config, state, selected); got != "anineko" {
		t.Fatalf("providerNameFromSelection() = %q, want anineko", got)
	}

	var anime Anime
	applySelectedProviderMapping(config, state, &anime, selected)
	if anime.ProviderName != "anineko" || anime.ProviderId != "frieren-beyond-journeys-end" {
		t.Fatalf("unexpected mapping %+v", anime)
	}
}

func TestProviderNameFromSelectionUsesQualifiedKey(t *testing.T) {
	withAllProvidersEnabledForTest(t)
	config := &CurdConfig{Provider: `["senshi","anineko"]`}
	state := &providerMappingSearchState{allProviders: []string{"senshi", "anineko"}}

	selected := SelectionOption{
		Key:   "anineko::frieren-beyond-journeys-end",
		Label: "Frieren: Beyond Journey's End [anineko]",
	}
	if got := providerNameFromSelection(config, state, selected); got != "anineko" {
		t.Fatalf("providerNameFromSelection() = %q, want anineko", got)
	}
}

func TestApplyMatchedProviderMappingUsesSequentialProvider(t *testing.T) {
	withAllProvidersEnabledForTest(t)
	config := &CurdConfig{Provider: `["senshi","anineko"]`}
	state := &providerMappingSearchState{
		allProviders:  []string{"senshi", "anineko"},
		sequential:    true,
		providerIndex: 1,
	}

	anime := Anime{ProviderId: "frieren-beyond-journeys-end"}
	applyMatchedProviderMapping(config, state, &anime)
	if anime.ProviderName != "anineko" || anime.ProviderId != "frieren-beyond-journeys-end" {
		t.Fatalf("unexpected mapping %+v", anime)
	}
}

// A multi-provider search runs one provider at a time and each can take up
// to the shared HTTP client's timeout, so a silent wait before the "no
// results, search again?" prompt can look like curd has hung. This locks in
// that a visible "Searching..." message goes out first, for terminal users.
func TestAnnounceProviderSearchStartPrintsForTerminalUsers(t *testing.T) {
	previousConfig := GetGlobalConfig()
	t.Cleanup(func() { SetGlobalConfig(previousConfig) })
	config := &CurdConfig{}
	SetGlobalConfig(config)

	state := &providerMappingSearchState{query: "one piece", allProviders: []string{"senshi", "anipub"}}

	output := captureStdout(t, func() {
		announceProviderSearchStart(config, state)
	})
	if !strings.Contains(output, `"one piece"`) {
		t.Fatalf("expected the announcement to mention the query, got %q", output)
	}
	if !strings.Contains(output, "senshi") || !strings.Contains(output, "anipub") {
		t.Fatalf("expected the announcement to mention the providers being searched, got %q", output)
	}
}

// Rofi mode already avoids notify-send spam for this flow (see
// selectWithOptionalMessage, which uses -mesg instead) — the search-start
// announcement should stay silent there rather than fire an extra
// notification on every search and every "search with a different name" retry.
func TestAnnounceProviderSearchStartSkipsNotifySpamUnderRofi(t *testing.T) {
	previousConfig := GetGlobalConfig()
	t.Cleanup(func() { SetGlobalConfig(previousConfig) })
	config := &CurdConfig{RofiSelection: true}
	SetGlobalConfig(config)

	state := &providerMappingSearchState{query: "one piece", allProviders: []string{"senshi", "anipub"}}

	output := captureStdout(t, func() {
		announceProviderSearchStart(config, state)
	})
	if output != "" {
		t.Fatalf("expected no stdout output under Rofi, got %q", output)
	}
}
