package smotretanime

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wraient/curd/internal/curdhost"
	"github.com/wraient/curd/internal/providers"
)

func withTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	previousClient := curdhost.HTTPClient
	curdhost.HTTPClient = func() *http.Client { return server.Client() }
	t.Cleanup(func() { curdhost.HTTPClient = previousClient })

	previousBase := baseURL
	baseURL = server.URL
	t.Cleanup(func() { baseURL = previousBase })

	return server
}

func withToken_(t *testing.T, tok string) {
	t.Helper()
	previous := curdhost.SmotretAnimeToken
	curdhost.SmotretAnimeToken = func() string { return tok }
	t.Cleanup(func() { curdhost.SmotretAnimeToken = previous })
}

func withLanguage(t *testing.T, lang string) {
	t.Helper()
	previous := curdhost.SubsLanguage
	curdhost.SubsLanguage = func() string { return lang }
	t.Cleanup(func() { curdhost.SubsLanguage = previous })
}

func writeEnvelope(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"error": nil, "data": data})
}

// writeRawEnvelope writes a literal "data" JSON fragment, for response
// shapes (like a half-integer episodeInt) that flexInt's Go value can't
// represent and so can't round-trip through writeEnvelope's json.Marshal.
func writeRawEnvelope(w http.ResponseWriter, rawData string) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"error":null,"data":` + rawData + `}`))
}

func TestSearchAnimeParsesResults(t *testing.T) {
	withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/series/" {
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("query") != "frieren" {
			t.Fatalf("unexpected query params %q", r.URL.RawQuery)
		}
		writeEnvelope(w, []seriesItem{{
			ID:               30414,
			MyAnimeListID:    52991,
			AnilistID:        154587,
			NumberOfEpisodes: 28,
			Type:             "tv",
			TypeTitle:        "TV",
			Year:             2023,
			Titles:           titles{EN: "Frieren: Beyond Journey's End", Romaji: "Sousou no Frieren"},
			Title:            "Sousou no Frieren",
			PosterURLSmall:   "https://smotret-anime.online/posters/30414.jpg",
		}})
	})

	options, err := searchAnime("frieren", "sub")
	if err != nil {
		t.Fatalf("searchAnime: %v", err)
	}
	if len(options) != 1 {
		t.Fatalf("expected 1 option, got %d", len(options))
	}
	if options[0].Key != "30414" {
		t.Fatalf("unexpected key %q", options[0].Key)
	}
	if options[0].Title != "Frieren: Beyond Journey's End" {
		t.Fatalf("unexpected title %q", options[0].Title)
	}
	item, ok := options[0].ExtraData.(SearchItem)
	if !ok || item.MalID != 52991 || item.AnilistID != 154587 {
		t.Fatalf("unexpected extra data %+v", options[0].ExtraData)
	}
}

func TestSearchAnimeReturnsErrorEnvelope(t *testing.T) {
	withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"code": 500, "message": "boom"},
		})
	})

	if _, err := searchAnime("frieren", "sub"); err == nil {
		t.Fatal("expected error")
	} else if !strings.Contains(err.Error(), "boom") {
		t.Fatalf("unexpected error %v", err)
	}
}

func TestEpisodesListFiltersToTVEpisodes(t *testing.T) {
	withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/series/30414" {
			http.NotFound(w, r)
			return
		}
		writeEnvelope(w, seriesDetail{Episodes: []episodeSummary{
			{ID: 306115, EpisodeInt: 1, EpisodeType: "preview", IsActive: 1},
			{ID: 291395, EpisodeInt: 1, EpisodeType: "tv", IsActive: 1},
			{ID: 312552, EpisodeInt: 2, EpisodeType: "tv", IsActive: 1},
			{ID: 999999, EpisodeInt: 3, EpisodeType: "tv", IsActive: 0},
		}})
	})

	episodes, err := episodesList("30414", "sub")
	if err != nil {
		t.Fatalf("episodesList: %v", err)
	}
	if len(episodes) != 2 || episodes[0] != "1" || episodes[1] != "2" {
		t.Fatalf("unexpected episodes %#v", episodes)
	}
}

// Movies, OVAs, and specials use their own episodeType (matching the
// series' own "type") instead of "tv" — e.g. a movie like "Evangelion 3.0"
// has a single episode with episodeType "movie". Only "preview" (trailer)
// entries should be excluded.
func TestEpisodesListIncludesNonTVCanonicalTypes(t *testing.T) {
	withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/series/584" {
			http.NotFound(w, r)
			return
		}
		writeEnvelope(w, seriesDetail{Episodes: []episodeSummary{
			{ID: 89186, EpisodeInt: 1, EpisodeType: "movie", IsActive: 1},
		}})
	})

	episodes, err := episodesList("584", "sub")
	if err != nil {
		t.Fatalf("episodesList: %v", err)
	}
	if len(episodes) != 1 || episodes[0] != "1" {
		t.Fatalf("unexpected episodes %#v", episodes)
	}
}

// Regression: long-running shows (e.g. One Piece) use half-integer
// episodeInt values like "1004.5" for interstitial episodes. That used to
// fail flexInt's strconv.Atoi and abort parsing the *entire* episode list,
// breaking every episode's lookup, not just the half-numbered one.
func TestEpisodesListToleratesHalfIntegerEpisodeInt(t *testing.T) {
	withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/series/8762" {
			http.NotFound(w, r)
			return
		}
		writeRawEnvelope(w, `{"episodes":[
			{"id":1,"episodeInt":1,"episodeType":"tv","isActive":1},
			{"id":2,"episodeInt":2,"episodeType":"tv","isActive":1},
			{"id":3,"episodeInt":1004.5,"episodeType":"tv","isActive":1}
		]}`)
	})

	episodes, err := episodesList("8762", "sub")
	if err != nil {
		t.Fatalf("episodesList: %v", err)
	}
	if len(episodes) != 3 || episodes[0] != "1" || episodes[1] != "2" || episodes[2] != "1004" {
		t.Fatalf("unexpected episodes %#v", episodes)
	}
}

func TestGetEpisodeStreamsForModeRequiresToken(t *testing.T) {
	withToken_(t, "")

	if _, _, err := getEpisodeStreamsForMode("30414", providers.PlaybackConfig{SubOrDub: "sub"}, 1); err != errNoToken {
		t.Fatalf("expected errNoToken, got %v", err)
	}
}

func TestGetEpisodeStreamsForModePicksHighestPriorityEnglishSub(t *testing.T) {
	withToken_(t, "test-token")
	withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/series/30414":
			writeEnvelope(w, seriesDetail{Episodes: []episodeSummary{
				{ID: 291395, EpisodeInt: 1, EpisodeType: "tv", IsActive: 1},
			}})
		case r.URL.Path == "/api/episodes/291395":
			writeEnvelope(w, episodeDetail{Translations: []translation{
				{ID: 1, TypeKind: "sub", TypeLang: "ru", IsActive: 1, Priority: 999999},
				{ID: 2, TypeKind: "sub", TypeLang: "en", IsActive: 1, Priority: 100},
				{ID: 3, TypeKind: "sub", TypeLang: "en", IsActive: 1, Priority: 200},
				{ID: 4, TypeKind: "voice", TypeLang: "en", IsActive: 1, Priority: 999999},
			}})
		case r.URL.Path == "/api/translations/embed/3":
			if r.URL.Query().Get("access_token") != "test-token" {
				t.Fatalf("missing access_token on embed request: %s", r.URL.RawQuery)
			}
			writeEnvelope(w, embedData{
				Stream: []embedStream{
					{Height: 360, URLs: []string{"https://cdn.example/360.mp4"}},
					{Height: 1080, URLs: []string{"https://cdn.example/1080.mp4"}},
				},
				SubtitlesVttURL: "https://smotret-anime.online/translations/vtt/3",
			})
		default:
			http.NotFound(w, r)
		}
	})

	links, hints, err := getEpisodeStreamsForMode("30414", providers.PlaybackConfig{SubOrDub: "sub"}, 1)
	if err != nil {
		t.Fatalf("getEpisodeStreamsForMode: %v", err)
	}
	if len(links) != 1 || links[0] != "https://cdn.example/1080.mp4" {
		t.Fatalf("unexpected links %#v", links)
	}
	hint := hints[links[0]]
	if !strings.Contains(hint.Subtitle, "access_token=test-token") {
		t.Fatalf("expected subtitle url to carry access_token, got %q", hint.Subtitle)
	}
}

func TestGetEpisodeStreamsForModePrefersRussianWhenConfigured(t *testing.T) {
	withToken_(t, "test-token")
	withLanguage(t, "ru")
	withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/series/30414":
			writeEnvelope(w, seriesDetail{Episodes: []episodeSummary{
				{ID: 291395, EpisodeInt: 1, EpisodeType: "tv", IsActive: 1},
			}})
		case r.URL.Path == "/api/episodes/291395":
			writeEnvelope(w, episodeDetail{Translations: []translation{
				{ID: 1, TypeKind: "sub", TypeLang: "en", IsActive: 1, Priority: 999999},
				{ID: 2, TypeKind: "sub", TypeLang: "ru", IsActive: 1, Priority: 100},
			}})
		case r.URL.Path == "/api/translations/embed/2":
			writeEnvelope(w, embedData{
				Stream:          []embedStream{{Height: 1080, URLs: []string{"https://cdn.example/ru.mp4"}}},
				SubtitlesVttURL: "https://smotret-anime.online/translations/vtt/2",
			})
		default:
			http.NotFound(w, r)
		}
	})

	links, hints, err := getEpisodeStreamsForMode("30414", providers.PlaybackConfig{SubOrDub: "sub"}, 1)
	if err != nil {
		t.Fatalf("getEpisodeStreamsForMode: %v", err)
	}
	if len(links) != 1 || links[0] != "https://cdn.example/ru.mp4" {
		t.Fatalf("expected the lower-priority Russian translation to win, got %#v", links)
	}
	if !strings.Contains(hints[links[0]].Subtitle, "/vtt/2") {
		t.Fatalf("expected the Russian translation's own subtitle track, got %q", hints[links[0]].Subtitle)
	}
}

func TestGetEpisodeStreamsForModeFallsBackWhenPreferredLanguageMissing(t *testing.T) {
	withToken_(t, "test-token")
	withLanguage(t, "ru")
	withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/series/30414":
			writeEnvelope(w, seriesDetail{Episodes: []episodeSummary{
				{ID: 291395, EpisodeInt: 1, EpisodeType: "tv", IsActive: 1},
			}})
		case r.URL.Path == "/api/episodes/291395":
			writeEnvelope(w, episodeDetail{Translations: []translation{
				{ID: 1, TypeKind: "sub", TypeLang: "en", IsActive: 1, Priority: 100},
			}})
		case r.URL.Path == "/api/translations/embed/1":
			writeEnvelope(w, embedData{
				Stream: []embedStream{{Height: 1080, URLs: []string{"https://cdn.example/en.mp4"}}},
			})
		default:
			http.NotFound(w, r)
		}
	})

	links, _, err := getEpisodeStreamsForMode("30414", providers.PlaybackConfig{SubOrDub: "sub"}, 1)
	if err != nil {
		t.Fatalf("getEpisodeStreamsForMode: %v", err)
	}
	if len(links) != 1 || links[0] != "https://cdn.example/en.mp4" {
		t.Fatalf("expected fallback to the only available (English) translation, got %#v", links)
	}
}

// Regression: selecting a movie search result (e.g. "Evangelion 3.0") and
// resolving its single episode used to fail with "episode 1 not found"
// because episodeIDForNumber only matched episodeType "tv".
func TestGetEpisodeStreamsForModeResolvesMovieEpisode(t *testing.T) {
	withToken_(t, "test-token")
	withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/series/584":
			writeEnvelope(w, seriesDetail{Episodes: []episodeSummary{
				{ID: 89186, EpisodeInt: 1, EpisodeType: "movie", IsActive: 1},
			}})
		case r.URL.Path == "/api/episodes/89186":
			writeEnvelope(w, episodeDetail{Translations: []translation{
				{ID: 1, TypeKind: "sub", TypeLang: "en", IsActive: 1, Priority: 100},
			}})
		case r.URL.Path == "/api/translations/embed/1":
			writeEnvelope(w, embedData{
				Stream: []embedStream{{Height: 1080, URLs: []string{"https://cdn.example/movie.mp4"}}},
			})
		default:
			http.NotFound(w, r)
		}
	})

	links, _, err := getEpisodeStreamsForMode("584", providers.PlaybackConfig{SubOrDub: "sub"}, 1)
	if err != nil {
		t.Fatalf("getEpisodeStreamsForMode: %v", err)
	}
	if len(links) != 1 || links[0] != "https://cdn.example/movie.mp4" {
		t.Fatalf("unexpected links %#v", links)
	}
}

func TestGetEpisodeStreamsForModeDubDoesNotAttachSubtitle(t *testing.T) {
	withToken_(t, "test-token")
	withTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/series/30414":
			writeEnvelope(w, seriesDetail{Episodes: []episodeSummary{
				{ID: 291395, EpisodeInt: 1, EpisodeType: "tv", IsActive: 1},
			}})
		case r.URL.Path == "/api/episodes/291395":
			writeEnvelope(w, episodeDetail{Translations: []translation{
				{ID: 4, TypeKind: "voice", TypeLang: "en", IsActive: 1, Priority: 100},
			}})
		case r.URL.Path == "/api/translations/embed/4":
			writeEnvelope(w, embedData{
				Stream:          []embedStream{{Height: 1080, URLs: []string{"https://cdn.example/dub.mp4"}}},
				SubtitlesVttURL: "https://smotret-anime.online/translations/vtt/4",
			})
		default:
			http.NotFound(w, r)
		}
	})

	links, hints, err := getEpisodeStreamsForMode("30414", providers.PlaybackConfig{SubOrDub: "dub"}, 1)
	if err != nil {
		t.Fatalf("getEpisodeStreamsForMode: %v", err)
	}
	if hints[links[0]].Subtitle != "" {
		t.Fatalf("expected no subtitle for dub mode, got %q", hints[links[0]].Subtitle)
	}
}
