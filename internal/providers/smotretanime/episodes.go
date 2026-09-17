package smotretanime

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

func episodesList(showID, mode string) ([]string, error) {
	seriesID, err := parseSeriesID(showID)
	if err != nil {
		return nil, err
	}
	_ = mode

	episodes, err := fetchSeriesEpisodes(seriesID)
	if err != nil {
		return nil, err
	}

	seen := map[int]struct{}{}
	for _, ep := range episodes {
		if !isCanonicalEpisode(ep) {
			continue
		}
		if n := int(ep.EpisodeInt); n > 0 {
			seen[n] = struct{}{}
		}
	}
	if len(seen) == 0 {
		return nil, fmt.Errorf("no episodes found for series id %d", seriesID)
	}

	nums := make([]int, 0, len(seen))
	for n := range seen {
		nums = append(nums, n)
	}
	sort.Ints(nums)

	result := make([]string, 0, len(nums))
	for _, n := range nums {
		result = append(result, strconv.Itoa(n))
	}
	return result, nil
}

func fetchSeriesEpisodes(seriesID int) ([]episodeSummary, error) {
	rawURL := fmt.Sprintf("%s/api/series/%d?fields=episodes", baseURL, seriesID)
	var detail seriesDetail
	if err := fetchJSON(rawURL, &detail); err != nil {
		return nil, err
	}
	return detail.Episodes, nil
}

// episodeIDForNumber resolves the internal episode id for a canonical
// (AniList-aligned) 1-based TV episode number.
func episodeIDForNumber(seriesID, epNo int) (int, error) {
	episodes, err := fetchSeriesEpisodes(seriesID)
	if err != nil {
		return 0, err
	}
	for _, ep := range episodes {
		if isCanonicalEpisode(ep) && int(ep.EpisodeInt) == epNo {
			return ep.ID, nil
		}
	}
	return 0, fmt.Errorf("episode %d not found for series id %d", epNo, seriesID)
}

// isCanonicalEpisode reports whether an episode is real watchable content
// rather than a trailer/preview. The site uses episodeType to mirror the
// series' own type (tv, movie, ova, ona, special, tv_special, ...) for every
// real release, and "preview" specifically for promotional trailers mixed
// into an otherwise normal episode list (e.g. a "Трейлер" entry ahead of a
// TV series' episode 1, or the sole entry for a movie/OVA/special release).
func isCanonicalEpisode(ep episodeSummary) bool {
	return ep.IsActive != 0 && ep.EpisodeType != "preview" && ep.EpisodeType != ""
}

func parseSeriesID(showID string) (int, error) {
	showID = strings.TrimSpace(showID)
	if showID == "" {
		return 0, fmt.Errorf("empty show id")
	}
	id, err := strconv.Atoi(showID)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid smotretanime series id %q", showID)
	}
	return id, nil
}
