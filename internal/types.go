package internal

import (
	"errors"
	"github.com/PuerkitoBio/goquery"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"runtime"
	"strings"
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
	//DownloadRegex =
)

// NewApplication returns an instance of the application
func NewApplication(opts ...ApplicationOption) (*Application, error) {
	a := &Application{}
	r, err := regexp.Compile(`^go(?P<version>[1-9]\.[0-9]{1,3}(\.[0-9]{1,3})?)\.(?P<goos>[^-]*)-(?P<goarch>[^\\.]*)`)
	if err != nil {
		return nil, err
	}
	a.versionRegex = r
	_ = WithBaseUrl(BaseUrl)(a)
	for i := range opts {
		err := opts[i](a)
		if err != nil {
			log.Printf("error setting option: %s", err)
		}
	}
	return a, nil
}

func (a *Application) QueryVersions() {
	// Request the HTML page.
	res, err := http.Get(a.baseUrl.String())
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Fatalf("status code error: %d %s", res.StatusCode, res.Status)
	}

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	// Find the review items
	doc.Find(".download").Each(func(i int, s *goquery.Selection) {
		a.processSelection(s)
	})
}

func (a *Application) processSelection(s *goquery.Selection) {
	title := s.Text()
	if !strings.HasSuffix(title, ".zip") && !strings.HasSuffix(title, ".tar.gz") {
		return
	}
	if !a.versionRegex.Match([]byte(title)) {
		return
	}
	href, exists := s.Attr("href")
	if !exists {
		return
	}

	match := a.versionRegex.FindStringSubmatch(title)
	result := make(map[string]string)
	for i, name := range a.versionRegex.SubexpNames() {
		if i != 0 && name != "" {
			result[name] = match[i]
		}
	}

	if result["goos"] != runtime.GOOS || result["goarch"] != runtime.GOARCH {
		return
	}

	u := url.URL{
		Scheme: a.baseUrl.Scheme,
		Host:   a.baseUrl.Host,
	}

	d := Download{
		Url:      u.JoinPath(href),
		Version:  result["version"],
		GoOs:     result["goos"],
		GoArch:   result["goarch"],
		FileName: title,
	}

	a.Downloads = append(a.Downloads, d)
}

func (a *Application) GetDownload(version string) (*Download, error) {
	for i := range a.Downloads {
		if a.Downloads[i].Version == version {
			return &a.Downloads[i], nil
		}
	}
	return nil, errors.New("no such go version")
}
