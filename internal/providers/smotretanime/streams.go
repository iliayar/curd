package smotretanime

import (
	"fmt"

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

	tr, err := bestTranslation(episodeID, wantKind)
	if err != nil {
		return nil, nil, err
	}

	embed, err := getEmbed(tr.ID)
	if err != nil {
		return nil, nil, err
	}
	if len(embed.Stream) == 0 {
		return nil, nil, fmt.Errorf("no stream urls for translation %d", tr.ID)
	}

	urls := bestQualityURLs(embed.Stream)
	if len(urls) == 0 {
		return nil, nil, fmt.Errorf("no stream urls for translation %d", tr.ID)
	}

	subtitle := ""
	if mode == "sub" && embed.SubtitlesVttURL != "" {
		if withTok, err := withToken(embed.SubtitlesVttURL); err == nil {
			subtitle = withTok
		}
	}

	hints := make(map[string]providers.StreamPlaybackHint, len(urls))
	for _, u := range urls {
		hints[u] = providers.StreamPlaybackHint{
			Referrer: baseURL + "/",
			Subtitle: subtitle,
		}
	}

	return urls, hints, nil
}

// bestTranslation picks the highest-priority active translation matching
// typeKind ("sub" or "voice") in the configured language, falling back to
// the other language (see languageOrder) when no track exists in it.
func bestTranslation(episodeID int, wantKind string) (translation, error) {
	rawURL := fmt.Sprintf("%s/api/episodes/%d", baseURL, episodeID)
	var detail episodeDetail
	if err := fetchJSON(rawURL, &detail); err != nil {
		return translation{}, err
	}

	for _, lang := range languageOrder() {
		var best *translation
		for i := range detail.Translations {
			tr := &detail.Translations[i]
			if tr.IsActive == 0 || tr.TypeKind != wantKind || tr.TypeLang != lang {
				continue
			}
			if best == nil || tr.Priority > best.Priority {
				best = tr
			}
		}
		if best != nil {
			return *best, nil
		}
	}

	return translation{}, fmt.Errorf("no active %s translation found for episode id %d", wantKind, episodeID)
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
