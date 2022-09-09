package internal

import "net/url"

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
