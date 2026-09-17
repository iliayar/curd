package providers

// SelectionOption is a provider search or menu result item.
type SelectionOption struct {
	Key       string
	Label     string
	Title     string
	Thumbnail string
	ExtraData any
}

// PlaybackConfig carries playback preferences passed into providers.
type PlaybackConfig struct {
	SubOrDub string
	// SubStyle controls subtitle delivery for sub-mode playback: ask, soft, or hard.
	SubStyle string
}

// SubtitleTrack is one selectable external subtitle for mpv, with enough
// metadata (Title, Lang) for it to show up as a distinguishable option in
// mpv's subtitle track menu instead of a bare URL.
type SubtitleTrack struct {
	URL   string
	Title string
	Lang  string
}

// StreamPlaybackHint carries MPV playback metadata for a resolved stream URL.
type StreamPlaybackHint struct {
	Referrer string
	// Subtitle is the primary/default subtitle URL, kept for providers and
	// callers that only deal with a single track. When Subtitles is set,
	// its first entry's URL should match Subtitle.
	Subtitle string
	// Subtitles lists every subtitle track the provider found for this
	// stream (e.g. multiple fansub groups or languages), so the host can
	// hand them all to mpv and let the viewer switch if the default one
	// has a problem. Optional — providers that only support one track can
	// leave this nil and rely on Subtitle alone.
	Subtitles []SubtitleTrack
}

// Provider resolves catalog search, episode lists, and stream URLs.
type Provider interface {
	Name() string
	SearchAnime(query, mode string) ([]SelectionOption, error)
	EpisodesList(showID, mode string) ([]string, error)
	GetEpisodeURL(config PlaybackConfig, id string, epNo int) ([]string, error)
}

// ModeResolver resolves episode URLs for an explicit sub/dub mode.
type ModeResolver interface {
	GetEpisodeURLForMode(config PlaybackConfig, id string, epNo int, mode string) ([]string, error)
}

// HintResolver resolves episode URLs with MPV playback hints.
type HintResolver interface {
	GetEpisodeURLForModeWithHints(config PlaybackConfig, id string, epNo int, mode string) ([]string, map[string]StreamPlaybackHint, error)
}

// IDResolver refreshes or validates a provider-specific show ID.
type IDResolver interface {
	ResolveProviderID(providerID, query string) (string, error)
}
