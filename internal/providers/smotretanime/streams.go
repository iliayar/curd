package smotretanime

import (
	"fmt"
	"sort"
	"strings"

	"github.com/wraient/curd/internal/providers"
)

func getEpisodeStreamsForMode(showID string, config providers.PlaybackConfig, epNo int) ([]string, map[string]providers.StreamPlaybackHint, error) {
	if token() == "" {
		return nil, nil, errNoToken
	}

	seriesID, err := parseSeriesID(showID)
	if err != nil {
		return nil, nil, err
	}
	if epNo <= 0 {
		return nil, nil, fmt.Errorf("invalid episode number %d", epNo)
	}

	mode := providers.NormalizeTranslationType(config.SubOrDub)
	wantKind := "sub"
	if mode == "dub" {
		wantKind = "voice"
	}

	episodeID, err := episodeIDForNumber(seriesID, epNo)
	if err != nil {
		return nil, nil, err
	}

	detail, err := fetchEpisodeDetail(episodeID)
	if err != nil {
		return nil, nil, err
	}

	candidates := rankedTranslations(detail.Translations, wantKind)
	if len(candidates) == 0 {
		return nil, nil, fmt.Errorf("no active %s translation found", wantKind)
	}

	var tr translation
	var urls []string
	var lastErr error
	for _, candidate := range candidates {
		embed, embedErr := getEmbed(candidate.ID)
		if embedErr != nil {
			lastErr = embedErr
			continue
		}
		if got := bestQualityURLs(embed.Stream); len(got) > 0 {
			tr = candidate
			urls = got
			break
		}
		lastErr = fmt.Errorf("no stream urls for translation %d", candidate.ID)
	}
	if len(urls) == 0 {
		if lastErr != nil {
			return nil, nil, lastErr
		}
		return nil, nil, fmt.Errorf("no stream urls found for episode id %d", episodeID)
	}

	var subtitle string
	var subtitles []providers.SubtitleTrack
	if mode == "sub" {
		subtitles = subtitleTracksForEpisode(detail.Translations, tr.ID)
		if len(subtitles) > 0 {
			subtitle = subtitles[0].URL
		}
	}

	hints := make(map[string]providers.StreamPlaybackHint, len(urls))
	for _, u := range urls {
		hints[u] = providers.StreamPlaybackHint{
			Referrer:  baseURL + "/",
			Subtitle:  subtitle,
			Subtitles: subtitles,
		}
	}

	return urls, hints, nil
}

func fetchEpisodeDetail(episodeID int) (episodeDetail, error) {
	rawURL := fmt.Sprintf("%s/api/episodes/%d", baseURL, episodeID)
	var detail episodeDetail
	if err := fetchJSON(rawURL, &detail); err != nil {
		return episodeDetail{}, err
	}
	return detail, nil
}

// qualityRank scores a translation's source master so a Blu-ray remux is
// always preferred over a TV-broadcast rip at the same (or even a nominally
// higher) resolution — a bd source is cleaner/less compressed at any given
// height. Unrecognized values rank with "tv" rather than erroring, since the
// site may add quality tiers we don't know about yet.
func qualityRank(qualityType string) int {
	if strings.EqualFold(qualityType, "bd") {
		return 1
	}
	return 0
}

// rankedTranslations returns every active translation matching typeKind
// ("sub" or "voice") in the first language from languageOrder() that has
// any, best candidate first: bd-sourced before tv-sourced, then by the
// site's own priority. Callers should try candidates in order and fall
// through to the next one if a "best" pick's stream/embed lookup fails,
// rather than erroring out while a working, still-high-quality alternative
// is available.
func rankedTranslations(translations []translation, wantKind string) []translation {
	for _, lang := range languageOrder() {
		var matches []translation
		for _, tr := range translations {
			if tr.IsActive == 0 || tr.TypeKind != wantKind || tr.TypeLang != lang {
				continue
			}
			matches = append(matches, tr)
		}
		if len(matches) == 0 {
			continue
		}
		sort.SliceStable(matches, func(i, j int) bool {
			qi, qj := qualityRank(matches[i].QualityType), qualityRank(matches[j].QualityType)
			if qi != qj {
				return qi > qj
			}
			return matches[i].Priority > matches[j].Priority
		})
		return matches
	}
	return nil
}

// subtitleTracksForEpisode returns every active subtitle translation (any
// language, e.g. multiple fansub groups per language) as an mpv-ready track
// list, so a viewer can switch away from the chosen default if it turns out
// broken or incomplete. primaryID's track is placed first so it's what mpv
// selects by default; the rest are ordered by priority, highest first.
func subtitleTracksForEpisode(translations []translation, primaryID int) []providers.SubtitleTrack {
	var primary *translation
	rest := make([]translation, 0, len(translations))
	for i := range translations {
		tr := translations[i]
		if tr.IsActive == 0 || tr.TypeKind != "sub" {
			continue
		}
		if tr.ID == primaryID {
			t := tr
			primary = &t
			continue
		}
		rest = append(rest, tr)
	}
	sort.SliceStable(rest, func(i, j int) bool {
		qi, qj := qualityRank(rest[i].QualityType), qualityRank(rest[j].QualityType)
		if qi != qj {
			return qi > qj
		}
		return rest[i].Priority > rest[j].Priority
	})

	tracks := make([]providers.SubtitleTrack, 0, len(rest)+1)
	if primary != nil {
		if track, ok := subtitleTrackFor(*primary); ok {
			tracks = append(tracks, track)
		}
	}
	for _, tr := range rest {
		if track, ok := subtitleTrackFor(tr); ok {
			tracks = append(tracks, track)
		}
	}
	return tracks
}

func subtitleTrackFor(tr translation) (providers.SubtitleTrack, bool) {
	url, err := withToken(fmt.Sprintf("%s/translations/vtt/%d", baseURL, tr.ID))
	if err != nil {
		return providers.SubtitleTrack{}, false
	}
	title := tr.TypeLang
	if summary := strings.TrimSpace(tr.AuthorsSummary); summary != "" {
		title = fmt.Sprintf("%s - %s", tr.TypeLang, summary)
	}
	return providers.SubtitleTrack{URL: url, Title: title, Lang: tr.TypeLang}, true
}

func getEmbed(translationID int) (embedData, error) {
	rawURL, err := withToken(fmt.Sprintf("%s/api/translations/embed/%d", baseURL, translationID))
	if err != nil {
		return embedData{}, err
	}
	var embed embedData
	if err := fetchJSON(rawURL, &embed); err != nil {
		return embedData{}, err
	}
	return embed, nil
}

// bestQualityURLs returns the urls from the highest-height stream entry.
func bestQualityURLs(streams []embedStream) []string {
	var best *embedStream
	for i := range streams {
		s := &streams[i]
		if len(s.URLs) == 0 {
			continue
		}
		if best == nil || s.Height > best.Height {
			best = s
		}
	}
	if best == nil {
		return nil
	}
	return best.URLs
}
