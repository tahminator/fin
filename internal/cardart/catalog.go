package cardart

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/tahminator/fin/internal/domain"
)

const (
	openCardHost         = "opencard-db.davidweng.workers.dev"
	clientName           = "fin"
	placeholderImagePath = "/v1/assets/default-card.webp"
	maxResponseBytes     = 4 << 20
)

type Catalog interface {
	Search(ctx context.Context, query string) ([]domain.CatalogCard, error)
	Card(ctx context.Context, id string) (domain.CatalogCard, error)
}

type OpenCard struct {
	httpClient *http.Client
	baseURL    *url.URL
}

func NewOpenCard(httpClient *http.Client) *OpenCard {
	return &OpenCard{
		httpClient: httpClient,
		baseURL:    &url.URL{Scheme: "https", Host: openCardHost, Path: "/v1/"},
	}
}

type openCardRecord struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Issuer *string `json:"issuer"`
	Image  *struct {
		URL string `json:"url"`
	} `json:"image"`
}

type openCardSearchResponse struct {
	Data []json.RawMessage `json:"data"`
}

func (c *OpenCard) Search(ctx context.Context, query string) ([]domain.CatalogCard, error) {
	var response openCardSearchResponse

	if err := c.get(ctx, "search?q="+url.QueryEscape(query), &response); err != nil {
		return nil, err
	}

	cards := make([]domain.CatalogCard, 0, len(response.Data))

	for _, raw := range response.Data {
		var record openCardRecord

		if err := json.Unmarshal(raw, &record); err != nil {
			continue
		}

		if card, ok := c.toCatalogCard(record); ok {
			cards = append(cards, card)
		}
	}

	return cards, nil
}

func (c *OpenCard) Card(ctx context.Context, id string) (domain.CatalogCard, error) {
	var record openCardRecord

	if err := c.get(ctx, "cards/"+url.PathEscape(id), &record); err != nil {
		return domain.CatalogCard{}, err
	}

	card, ok := c.toCatalogCard(record)
	if !ok {
		return domain.CatalogCard{}, fmt.Errorf("%w: card %q has no artwork", domain.ErrNotFound, id)
	}

	return card, nil
}

func (c *OpenCard) get(ctx context.Context, path string, target any) error {
	endpoint, err := c.baseURL.Parse(path)
	if err != nil {
		return fmt.Errorf("build card catalog url: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), http.NoBody)
	if err != nil {
		return fmt.Errorf("build card catalog request: %w", err)
	}

	request.Header.Set("Accept", "application/json")
	request.Header.Set("X-Client-Name", clientName)

	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("%w: card catalog request: %w", domain.ErrUnavailable, err)
	}

	defer func() { _ = response.Body.Close() }()

	switch {
	case response.StatusCode == http.StatusNotFound:
		return fmt.Errorf("%w: card catalog has no %s", domain.ErrNotFound, endpoint.Path)
	case response.StatusCode != http.StatusOK:
		return fmt.Errorf("%w: card catalog returned status %d", domain.ErrUnavailable, response.StatusCode)
	}

	if err := json.NewDecoder(io.LimitReader(response.Body, maxResponseBytes)).Decode(target); err != nil {
		return fmt.Errorf("%w: decode card catalog response: %w", domain.ErrUnavailable, err)
	}

	return nil
}

func (c *OpenCard) toCatalogCard(record openCardRecord) (domain.CatalogCard, bool) {
	if record.ID == "" || record.Name == "" || record.Image == nil || record.Image.URL == "" {
		return domain.CatalogCard{}, false
	}

	imageURL, err := c.baseURL.Parse(record.Image.URL)
	if err != nil || imageURL.Scheme != "https" || strings.HasSuffix(imageURL.Path, placeholderImagePath) {
		return domain.CatalogCard{}, false
	}

	return domain.CatalogCard{
		ID:       record.ID,
		Name:     record.Name,
		Issuer:   record.Issuer,
		ImageURL: imageURL.String(),
	}, true
}
