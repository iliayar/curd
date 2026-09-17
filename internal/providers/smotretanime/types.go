package smotretanime

import (
	"bytes"
	"encoding/json"
	"strconv"
)

// SearchItem is stored in SelectionOption.ExtraData for tracker matching hints.
type SearchItem struct {
	SeriesID  int
	MalID     int
	AnilistID int
	Title     string
	Type      string
	Episodes  int
	Year      int
}

// flexInt unmarshals a JSON number that the API sometimes encodes as a
// quoted string (e.g. "episodeInt") and sometimes as a bare number. Long
// franchises (e.g. One Piece) also use half-integer values like "1004.5"
// for interstitial episodes; those are truncated to the preceding whole
// episode number rather than failing to parse (a parse failure would fail
// the entire episode list response, not just that one entry).
type flexInt int

func (f *flexInt) UnmarshalJSON(data []byte) error {
	trimmed := bytes.Trim(data, `"`)
	if len(trimmed) == 0 {
		*f = 0
		return nil
	}
	if n, err := strconv.Atoi(string(trimmed)); err == nil {
		*f = flexInt(n)
		return nil
	}
	v, err := strconv.ParseFloat(string(trimmed), 64)
	if err != nil {
		return err
	}
	*f = flexInt(int(v))
	return nil
}

type apiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// envelope wraps every API response. The API can answer HTTP 200 with an
// `error` object instead of `data` (e.g. when the caller has no access_token),
// so callers must check Error independently of the HTTP status code.
type envelope struct {
	Error *apiError       `json:"error"`
	Data  json.RawMessage `json:"data"`
}

type titles struct {
	RU     string `json:"ru"`
	Romaji string `json:"romaji"`
	JA     string `json:"ja"`
	EN     string `json:"en"`
}

type seriesItem struct {
	ID               int    `json:"id"`
	MyAnimeListID    int    `json:"myAnimeListId"`
	AnilistID        int    `json:"anilistId"`
	NumberOfEpisodes int    `json:"numberOfEpisodes"`
	Type             string `json:"type"`
	TypeTitle        string `json:"typeTitle"`
	Year             int    `json:"year"`
	Season           string `json:"season"`
	Titles           titles `json:"titles"`
	Title            string `json:"title"`
	PosterURLSmall   string `json:"posterUrlSmall"`
}

type episodeSummary struct {
	ID          int     `json:"id"`
	EpisodeInt  flexInt `json:"episodeInt"`
	EpisodeFull string  `json:"episodeFull"`
	EpisodeType string  `json:"episodeType"`
	IsActive    int     `json:"isActive"`
}

type seriesDetail struct {
	Episodes []episodeSummary `json:"episodes"`
}

type translation struct {
	ID             int    `json:"id"`
	Type           string `json:"type"`
	TypeKind       string `json:"typeKind"`
	TypeLang       string `json:"typeLang"`
	QualityType    string `json:"qualityType"`
	Priority       int    `json:"priority"`
	IsActive       int    `json:"isActive"`
	AuthorsSummary string `json:"authorsSummary"`
}

type episodeDetail struct {
	Translations []translation `json:"translations"`
}

type embedStream struct {
	Height int      `json:"height"`
	URLs   []string `json:"urls"`
}

type embedData struct {
	Stream          []embedStream `json:"stream"`
	SubtitlesVttURL string        `json:"subtitlesVttUrl"`
}
