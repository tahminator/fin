package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
)

type envReader struct {
	errs []error
}

func (e *envReader) required(key string) string {
	value := strings.TrimSpace(os.Getenv(key))

	if value == "" {
		e.errs = append(e.errs, fmt.Errorf("%s is required", key))
	}

	return value
}

func (e *envReader) optional(key, fallback string) string {
	value, ok := os.LookupEnv(key)

	if !ok || strings.TrimSpace(value) == "" {
		return fallback
	}

	return strings.TrimSpace(value)
}

func (e *envReader) boolean(key string, fallback bool) bool {
	raw := e.optional(key, strconv.FormatBool(fallback))

	value, err := strconv.ParseBool(raw)
	if err != nil {
		e.errs = append(e.errs, fmt.Errorf("%s must be true or false: %w", key, err))
	}

	return value
}

func (e *envReader) duration(key string, fallback time.Duration) time.Duration {
	raw := e.optional(key, fallback.String())

	value, err := time.ParseDuration(raw)
	if err != nil {
		e.errs = append(e.errs, fmt.Errorf("%s must be a duration such as 6h: %w", key, err))
	}

	return value
}

func (e *envReader) schedule(key, fallback string) cron.Schedule {
	raw := e.optional(key, fallback)

	if raw == "off" {
		return nil
	}

	schedule, err := cron.ParseStandard(raw)
	if err != nil {
		e.errs = append(e.errs, fmt.Errorf("%s must be a 5-field cron expression or off: %w", key, err))
	}

	return schedule
}

func (e *envReader) list(key string, fallback []string) []string {
	raw := e.optional(key, "")

	if raw == "" {
		return fallback
	}

	var values []string

	for item := range strings.SplitSeq(raw, ",") {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			values = append(values, trimmed)
		}
	}

	return values
}
