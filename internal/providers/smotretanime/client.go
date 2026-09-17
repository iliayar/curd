package smotretanime

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/wraient/curd/internal/curdhost"
)

// baseURL is a var (not const) so tests can point it at an httptest server.
var baseURL = "https://smotret-anime.online"

const userAgent = "curd"

// apiErr is returned when the API answers HTTP 200 with an `error` envelope,
// most commonly {"code":403,"message":"You should login first."} when no
// (or an invalid) access_token was sent.
type apiErr struct {
	Code    int
	Message string
}

func (e *apiErr) Error() string {
	return fmt.Sprintf("smotretanime API error %d: %s", e.Code, e.Message)
}

// token returns the user's personal API access token, configured via
// SmotretAnimeToken in curd.conf. Streaming and subtitle endpoints require
// it; catalog search and listing do not.
func token() string {
	if curdhost.SmotretAnimeToken == nil {
		return ""
	}
	return strings.TrimSpace(curdhost.SmotretAnimeToken())
}

// errNoToken is returned by calls that need a personal access token when
// none is configured, without making a network request.
var errNoToken = fmt.Errorf("smotretanime: set SmotretAnimeToken in curd.conf to use this provider (your personal API token from your smotret-anime.online account settings)")

// languageOrder returns which typeLang to try first, then fall back to, when
// picking a translation. Reuses the existing SubsLanguage setting in
// curd.conf ("english", the default, or "russian"); either way the other
// language is used as a fallback so playback still works for titles missing
// a track in the preferred one.
func languageOrder() []string {
	lang := ""
	if curdhost.SubsLanguage != nil {
		lang = strings.ToLower(strings.TrimSpace(curdhost.SubsLanguage()))
	}
	switch lang {
	case "ru", "rus", "russian":
		return []string{"ru", "en"}
	default:
		return []string{"en", "ru"}
	}
}

func fetchJSON(rawURL string, dest any) error {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := curdhost.HTTPClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if !curdhost.HTTPStatusOK(resp.StatusCode) {
		return curdhost.HTTPStatusError("smotretanime request", resp.StatusCode, raw)
	}

	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return fmt.Errorf("parse smotretanime response: %w", err)
	}
	if env.Error != nil {
		return &apiErr{Code: env.Error.Code, Message: env.Error.Message}
	}
	if dest == nil || len(env.Data) == 0 {
		return nil
	}
	if err := json.Unmarshal(env.Data, dest); err != nil {
		return fmt.Errorf("parse smotretanime response data: %w", err)
	}
	return nil
}

// withToken appends the caller's access_token to a URL that already has
// query parameters (or not).
func withToken(rawURL string) (string, error) {
	tok := token()
	if tok == "" {
		return "", errNoToken
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	query := parsed.Query()
	query.Set("access_token", tok)
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func posterURL(item seriesItem) string {
	return strings.TrimSpace(item.PosterURLSmall)
}
