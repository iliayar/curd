package smotretanime

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/wraient/curd/internal/curdhost"
	"github.com/wraient/curd/internal/providers"
)

const searchFields = "id,title,titles,posterUrlSmall,numberOfEpisodes,type,typeTitle,year,season,myAnimeListId,anilistId"

func searchAnime(query, mode string) ([]providers.SelectionOption, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("empty search query")
	}
	_ = mode

	rawURL := fmt.Sprintf("%s/api/series/?query=%s&fields=%s", baseURL, url.QueryEscape(query), searchFields)

	var items []seriesItem
	if err := fetchJSON(rawURL, &items); err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("no results for %q", query)
	}

	options := make([]providers.SelectionOption, 0, len(items))
	for _, item := range items {
		if item.ID <= 0 {
			continue
		}
		title := displayTitle(item)
		options = append(options, providers.SelectionOption{
			Key:       strconv.Itoa(item.ID),
			Label:     formatSearchLabel(item, title),
			Title:     title,
			Thumbnail: posterURL(item),
			ExtraData: SearchItem{
				SeriesID:  item.ID,
				MalID:     item.MyAnimeListID,
				AnilistID: item.AnilistID,
				Title:     title,
				Type:      item.Type,
				Episodes:  item.NumberOfEpisodes,
				Year:      item.Year,
			},
		})
	}
	if len(options) == 0 {
		return nil, fmt.Errorf("no results for %q", query)
	}
	return options, nil
}

func displayTitle(item seriesItem) string {
	if curdhost.AnimeNameLanguage != nil && strings.TrimSpace(curdhost.AnimeNameLanguage()) == "romaji" {
		if t := strings.TrimSpace(item.Titles.Romaji); t != "" {
			return t
		}
	}
	if t := strings.TrimSpace(item.Titles.EN); t != "" {
		return t
	}
	if t := strings.TrimSpace(item.Titles.Romaji); t != "" {
		return t
	}
	return strings.TrimSpace(item.Title)
}

func formatSearchLabel(item seriesItem, title string) string {
	parts := []string{title}
	if item.TypeTitle != "" {
		parts = append(parts, item.TypeTitle)
	}
	if item.Year > 0 {
		parts = append(parts, strconv.Itoa(item.Year))
	}
	if item.NumberOfEpisodes > 0 {
		parts = append(parts, fmt.Sprintf("%d eps", item.NumberOfEpisodes))
	}
	return strings.Join(parts, " · ")
}
