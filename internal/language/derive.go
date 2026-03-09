package language

import "github.com/infobloxopen/apx/internal/config"

// DeriveAllCoords iterates all available plugins for the given context
// and returns a map of language name → coordinates.
func DeriveAllCoords(ctx DerivationContext) (map[string]config.LanguageCoords, error) {
	coords := make(map[string]config.LanguageCoords)
	for _, p := range Available(ctx) {
		c, err := p.DeriveCoords(ctx)
		if err != nil {
			return nil, err
		}
		coords[p.Name()] = c
	}
	return coords, nil
}
