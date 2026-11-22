package oauth

import (
	"errors"
	"net"
	"net/url"
	"slices"
	"strings"
)

var (
	errRedirectInvalid  = errors.New("redirect_uri must be an absolute URI")
	errRedirectScheme   = errors.New("redirect_uri must use https, loopback http, or an app-specific scheme")
	errRedirectFragment = errors.New("redirect_uri must not contain a fragment or user info")
)

var blockedSchemes = []string{"javascript", "data", "file", "vbscript", "about", "blob", "ftp", "ws", "wss"}

func ValidateRedirectURI(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" {
		return errRedirectInvalid
	}

	if parsed.Fragment != "" || strings.Contains(raw, "#") || parsed.User != nil {
		return errRedirectFragment
	}

	scheme := strings.ToLower(parsed.Scheme)

	switch {
	case scheme == "https":
		if parsed.Host == "" {
			return errRedirectInvalid
		}

		return nil
	case scheme == "http":
		if !isLoopbackHost(parsed.Hostname()) {
			return errRedirectScheme
		}

		return nil
	case slices.Contains(blockedSchemes, scheme):
		return errRedirectScheme
	}

	return nil
}

func MatchRedirectURI(registered []string, requested string) bool {
	if slices.Contains(registered, requested) {
		return true
	}

	requestedURL, err := url.Parse(requested)
	if err != nil || requestedURL.Scheme != "http" || !isLoopbackHost(requestedURL.Hostname()) {
		return false
	}

	for _, candidate := range registered {
		candidateURL, err := url.Parse(candidate)
		if err != nil || candidateURL.Scheme != "http" || !isLoopbackHost(candidateURL.Hostname()) {
			continue
		}

		if candidateURL.Hostname() == requestedURL.Hostname() && candidateURL.Path == requestedURL.Path && candidateURL.RawQuery == requestedURL.RawQuery {
			return true
		}
	}

	return false
}

func isLoopbackHost(host string) bool {
	if host == "localhost" {
		return true
	}

	ip := net.ParseIP(host)

	return ip != nil && ip.IsLoopback()
}
