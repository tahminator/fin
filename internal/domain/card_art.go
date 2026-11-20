package domain

type CardArtSource string

const (
	CardArtSourceMatched  CardArtSource = "matched"
	CardArtSourceSelected CardArtSource = "selected"
)

func (s CardArtSource) Valid() bool {
	return s == CardArtSourceMatched || s == CardArtSourceSelected
}

type CardArt struct {
	CardID   *string
	ImageURL *string
}

type CatalogCard struct {
	ID       string
	Name     string
	Issuer   *string
	ImageURL string
}
