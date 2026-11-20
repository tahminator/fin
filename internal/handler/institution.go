package handler

import "net/url"

func institutionLogoURL(institutionURL *string) *string {
	if institutionURL == nil || *institutionURL == "" {
		return nil
	}

	host := institutionHost(*institutionURL)
	if host == "" {
		return nil
	}

	logoURL := "https://www.google.com/s2/favicons?domain=" + host + "&sz=64"

	return &logoURL
}

func institutionHost(raw string) string {
	if parsed, err := url.Parse(raw); err == nil && parsed.Host != "" {
		return parsed.Host
	}

	parsed, err := url.Parse("https://" + raw)
	if err != nil {
		return ""
	}

	return parsed.Host
}
