package internal

import (
	"net/url"
	"regexp"
)

type (
	Download struct {
		Url      *url.URL
		Version  string
		GoOs     string
		GoArch   string
		FileName string
	}

	Application struct {
		baseUrl      *url.URL
		Downloads    []Download
		versionRegex *regexp.Regexp
	}

	ApplicationOption func(application *Application) error
)

const (
	BaseUrl = "https://go.dev/dl/"
)
