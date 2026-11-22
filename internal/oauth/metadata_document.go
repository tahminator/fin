package oauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/tahminator/fin/internal/domain"
)

const (
	metadataDocumentMaxBytes = 64 << 10
	metadataDocumentTimeout  = 5 * time.Second
	metadataDocumentMaxAge   = 24 * time.Hour
	metadataDocumentMinAge   = 5 * time.Minute
)

var errBlockedAddress = errors.New("client metadata document resolves to a non-public address")

type ClientMetadata struct {
	ClientID                          string   `json:"client_id"`
	ClientName                        string   `json:"client_name"`
	ClientURI                         string   `json:"client_uri"`
	LogoURI                           string   `json:"logo_uri"`
	RedirectURIs                      []string `json:"redirect_uris"`
	GrantTypes                        []string `json:"grant_types"`
	ResponseTypes                     []string `json:"response_types"`
	TokenEndpointAuthMethod           string   `json:"token_endpoint_auth_method"`
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported"`
	Scope                             string   `json:"scope"`
}

type FetchedClient struct {
	Client domain.OAuthClient
	MaxAge time.Duration
}

type MetadataDocumentFetcher struct {
	client *http.Client
}

func NewMetadataDocumentFetcher(allowed func(netip.Addr) bool) *MetadataDocumentFetcher {
	dialer := &net.Dialer{
		Timeout: metadataDocumentTimeout,
		Control: func(_, address string, _ syscall.RawConn) error {
			addrPort, err := netip.ParseAddrPort(address)
			if err != nil || !allowed(addrPort.Addr().Unmap()) {
				return errBlockedAddress
			}

			return nil
		},
	}

	transport := &http.Transport{
		Proxy:                 nil,
		DialContext:           dialer.DialContext,
		TLSHandshakeTimeout:   metadataDocumentTimeout,
		ResponseHeaderTimeout: metadataDocumentTimeout,
		MaxIdleConns:          4,
		IdleConnTimeout:       30 * time.Second,
	}

	return &MetadataDocumentFetcher{
		client: &http.Client{
			Timeout:   metadataDocumentTimeout,
			Transport: transport,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

func PublicAddressesOnly(addr netip.Addr) bool {
	if !addr.IsValid() || addr.IsLoopback() || addr.IsPrivate() || addr.IsLinkLocalUnicast() ||
		addr.IsLinkLocalMulticast() || addr.IsMulticast() || addr.IsUnspecified() || addr.IsInterfaceLocalMulticast() {
		return false
	}

	for _, blocked := range nonPublicPrefixes {
		if blocked.Contains(addr) {
			return false
		}
	}

	return true
}

var nonPublicPrefixes = []netip.Prefix{
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("64:ff9b::/96"),
	netip.MustParsePrefix("fc00::/7"),
}

func IsMetadataDocumentClientID(clientID string) bool {
	parsed, err := url.Parse(clientID)

	return err == nil && parsed.Scheme == "https" && parsed.Host != "" && parsed.Path != "" && parsed.Path != "/" &&
		parsed.Fragment == "" && parsed.User == nil
}

func (f *MetadataDocumentFetcher) Fetch(ctx context.Context, clientID string) (FetchedClient, error) {
	if !IsMetadataDocumentClientID(clientID) {
		return FetchedClient{}, NewError(ErrInvalidClient, "client_id is not a valid client metadata document URL")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, clientID, http.NoBody)
	if err != nil {
		return FetchedClient{}, NewError(ErrInvalidClient, "client_id is not a valid client metadata document URL")
	}

	req.Header.Set("Accept", "application/json")

	resp, err := f.client.Do(req)
	if err != nil {
		return FetchedClient{}, NewError(ErrInvalidClient, "could not fetch the client metadata document")
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return FetchedClient{}, NewError(ErrInvalidClient, fmt.Sprintf("client metadata document returned HTTP %d", resp.StatusCode))
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, metadataDocumentMaxBytes+1))
	if err != nil || len(body) > metadataDocumentMaxBytes {
		return FetchedClient{}, NewError(ErrInvalidClient, "client metadata document is too large or unreadable")
	}

	var metadata ClientMetadata

	if err := json.Unmarshal(body, &metadata); err != nil {
		return FetchedClient{}, NewError(ErrInvalidClient, "client metadata document is not valid JSON")
	}

	client, err := ClientFromMetadataDocument(clientID, metadata)
	if err != nil {
		return FetchedClient{}, err
	}

	return FetchedClient{Client: client, MaxAge: maxAgeOf(resp.Header.Get("Cache-Control"))}, nil
}

func ClientFromMetadataDocument(clientID string, metadata ClientMetadata) (domain.OAuthClient, error) {
	if metadata.ClientID != clientID {
		return domain.OAuthClient{}, NewError(ErrInvalidClient, "client metadata document client_id does not match its URL")
	}

	if strings.TrimSpace(metadata.ClientName) == "" || len(metadata.RedirectURIs) == 0 {
		return domain.OAuthClient{}, NewError(ErrInvalidClient, "client metadata document needs client_name and redirect_uris")
	}

	var method domain.TokenAuthMethod

	if len(metadata.TokenEndpointAuthMethodsSupported) == 0 {
		method = domain.TokenAuthMethod(metadata.TokenEndpointAuthMethod)
		if method == "" {
			method = domain.TokenAuthNone
		}
	} else {
		method = ""

		for _, supported := range metadata.TokenEndpointAuthMethodsSupported {
			if supported == string(domain.TokenAuthNone) {
				method = domain.TokenAuthNone

				break
			}
		}
	}

	if method != domain.TokenAuthNone {
		return domain.OAuthClient{}, NewError(ErrInvalidClient, "client metadata documents must support token_endpoint_auth_method none")
	}

	for _, redirectURI := range metadata.RedirectURIs {
		if err := ValidateRedirectURI(redirectURI); err != nil {
			return domain.OAuthClient{}, NewError(ErrInvalidClient, err.Error())
		}
	}

	return domain.OAuthClient{
		ID:                      clientID,
		Source:                  domain.ClientSourceMetadataDocument,
		ClientName:              strings.TrimSpace(metadata.ClientName),
		ClientURI:               optionalHTTPS(metadata.ClientURI),
		LogoURI:                 optionalHTTPS(metadata.LogoURI),
		RedirectURIs:            metadata.RedirectURIs,
		TokenEndpointAuthMethod: method,
	}, nil
}

func optionalHTTPS(raw string) *string {
	parsed, err := url.Parse(raw)
	if raw == "" || err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return nil
	}

	return &raw
}

func maxAgeOf(cacheControl string) time.Duration {
	for directive := range strings.SplitSeq(cacheControl, ",") {
		name, value, found := strings.Cut(strings.TrimSpace(directive), "=")
		if !found || !strings.EqualFold(name, "max-age") {
			continue
		}

		seconds, err := strconv.Atoi(strings.Trim(value, `"`))
		if err != nil {
			break
		}

		return min(max(time.Duration(seconds)*time.Second, metadataDocumentMinAge), metadataDocumentMaxAge)
	}

	return metadataDocumentMaxAge
}
