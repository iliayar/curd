package providers_test

import (
	"net/http"
	"os"
	"testing"

	"github.com/wraient/curd/internal/curdhost"
	"github.com/wraient/curd/internal/providers"
	"github.com/wraient/curd/internal/providers/smotretanime"
)

// Exercises the real smotret-anime.online API end to end. Requires a
// personal access token (Settings on the site) since stream/subtitle
// resolution is gated behind login:
//
//	CURD_LIVE_SMOTRETANIME=1 CURD_SMOTRETANIME_TOKEN=... go test ./internal/providers -run TestLiveSmotretAnime -v
func TestLiveSmotretAnime(t *testing.T) {
	if os.Getenv("CURD_LIVE_SMOTRETANIME") == "" {
		t.Skip("set CURD_LIVE_SMOTRETANIME=1 (and CURD_SMOTRETANIME_TOKEN) to run the live smotretanime test")
	}
	token := os.Getenv("CURD_SMOTRETANIME_TOKEN")
	if token == "" {
		t.Fatal("CURD_SMOTRETANIME_TOKEN is required for the live smotretanime test")
	}

	curdhost.HTTPClient = func() *http.Client { return http.DefaultClient }
	curdhost.SmotretAnimeToken = func() string { return token }

	p := &smotretanime.Provider{}

	options, err := p.SearchAnime("frieren", "sub")
	if err != nil {
		t.Fatalf("SearchAnime: %v", err)
	}
	if len(options) == 0 {
		t.Fatal("expected at least one search result")
	}
	t.Logf("first result: key=%s title=%s", options[0].Key, options[0].Title)

	episodes, err := p.EpisodesList(options[0].Key, "sub")
	if err != nil {
		t.Fatalf("EpisodesList: %v", err)
	}
	if len(episodes) == 0 {
		t.Fatal("expected at least one episode")
	}
	t.Logf("episodes: %v", episodes[:min(5, len(episodes))])

	links, err := p.GetEpisodeURL(providers.PlaybackConfig{SubOrDub: "sub"}, options[0].Key, 1)
	if err != nil {
		t.Fatalf("GetEpisodeURL(sub): %v", err)
	}
	if len(links) == 0 {
		t.Fatal("expected at least one sub stream url")
	}
	t.Logf("sub link: %s", links[0])

	dubLinks, err := p.GetEpisodeURL(providers.PlaybackConfig{SubOrDub: "dub"}, options[0].Key, 1)
	if err != nil {
		t.Logf("GetEpisodeURL(dub): %v (dub may be unavailable for this title)", err)
	} else if len(dubLinks) > 0 {
		t.Logf("dub link: %s", dubLinks[0])
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
