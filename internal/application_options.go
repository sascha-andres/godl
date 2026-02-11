package internal

import (
	"log/slog"
	"net/url"
)

func WithIncludeReleaseCandidates() ApplicationOption {
	return func(application *Application) error {
		application.includeReleaseCandidates = true
		return nil
	}
}

// WithBaseUrl allows overriding the base url
func WithBaseUrl(baseUrl string) ApplicationOption {
	return func(application *Application) error {
		parsed, err := url.Parse(baseUrl)
		if err != nil {
			return err
		}
		application.baseUrl = parsed
		return nil
	}
}

func WithVerbose() ApplicationOption {
	return func(application *Application) error {
		application.verbose = true
		return nil
	}
}

// WithLogger allows setting the logger
func WithLogger(logger *slog.Logger) ApplicationOption {
	return func(application *Application) error {
		application.logger = logger
		return nil
	}
}
