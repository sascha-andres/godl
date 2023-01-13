package internal

import (
	"net/url"
	"regexp"
)

type (
	// Download is a single Go download
	Download struct {
		// Url is pointing to the archive
		Url *url.URL
		// Version of download
		Version string
		// GoOs of download
		GoOs string
		// GoArch of download
		GoArch string
		// FileName of download
		FileName string
	}

	// Application is the base for all business logic
	Application struct {
		// baseUrl of download page
		baseUrl *url.URL
		// Downloads found
		Downloads []Download
		// versionRegex is a regular expression that extracts the single values
		versionRegex *regexp.Regexp
		// verbose is used to control verbosity
		verbose bool
		// includeReleaseCandidates will show release candidates as something to install
		includeReleaseCandidates bool
	}

	// ApplicationOption can be used to control behavior
	ApplicationOption func(application *Application) error
)

const (
	BaseUrl = "https://go.dev/dl/"
)
