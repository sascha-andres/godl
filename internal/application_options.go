package internal

import "net/url"

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
