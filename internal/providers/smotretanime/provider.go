package smotretanime

import "github.com/wraient/curd/internal/providers"

// Provider implements smotret-anime.online (Anime 365) catalog search and
// direct MP4/CDN stream resolution. Streaming and subtitle endpoints require
// a personal access token (SmotretAnimeToken in curd.conf); search does not.
type Provider struct{}

func (p *Provider) Name() string {
	return "smotretanime"
}

func (p *Provider) SearchAnime(query, mode string) ([]providers.SelectionOption, error) {
	return searchAnime(query, mode)
}

func (p *Provider) EpisodesList(showID, mode string) ([]string, error) {
	return episodesList(showID, mode)
}

func (p *Provider) GetEpisodeURL(config providers.PlaybackConfig, id string, epNo int) ([]string, error) {
	links, _, err := p.GetEpisodeURLForModeWithHints(config, id, epNo, config.SubOrDub)
	return links, err
}

func (p *Provider) GetEpisodeURLForMode(config providers.PlaybackConfig, id string, epNo int, mode string) ([]string, error) {
	links, _, err := p.GetEpisodeURLForModeWithHints(config, id, epNo, mode)
	return links, err
}

func (p *Provider) GetEpisodeURLForModeWithHints(config providers.PlaybackConfig, id string, epNo int, mode string) ([]string, map[string]providers.StreamPlaybackHint, error) {
	playback := config
	playback.SubOrDub = providers.NormalizeTranslationType(mode)
	return getEpisodeStreamsForMode(id, playback, epNo)
}
