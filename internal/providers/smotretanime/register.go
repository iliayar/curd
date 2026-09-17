package smotretanime

import "github.com/wraient/curd/internal/providers"

func init() {
	providers.Register(providers.Meta{
		Name:     "smotretanime",
		Aliases:  []string{"smotret-anime", "anime365"},
		Referrer: "https://smotret-anime.online/",
	}, func() providers.Provider {
		return &Provider{}
	})
}
