package internal

import "testing"

// attachExtraSubtitleTracks skips index 0 (the already-selected primary
// track) via tracks[1:]; this locks in that short/empty inputs don't panic.
func TestAttachExtraSubtitleTracksNoopsOnShortInput(t *testing.T) {
	attachExtraSubtitleTracks("", nil)
	attachExtraSubtitleTracks("/tmp/does-not-exist.sock", nil)
	attachExtraSubtitleTracks("/tmp/does-not-exist.sock", []SubtitleTrack{{URL: "https://example.com/only.vtt"}})
}
