package internal

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"errors"
	"github.com/PuerkitoBio/goquery"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
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
	a.queryVersions()
	return a, nil
}

// queryVersions connects to go.dev to gather all known go versions
func (a *Application) queryVersions() {
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

// processSelection is transforming a download link to out internal version representation
// it will skip over go versions that are not runtime OS or arch
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

// GetDownload will return download data
func (a *Application) GetDownload(version string) (*Download, error) {
	for i := range a.Downloads {
		if a.Downloads[i].Version == version {
			return &a.Downloads[i], nil
		}
	}
	return nil, errors.New("no such go version")
}

// Untar takes a destination path and a reader; a tar reader loops over the tarfile
// creating the file structure at 'dst' along the way, and writing any files
func (a *Application) Untar(dst string, r io.Reader) error {

	gzr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()

		switch {

		// if no more files are found return
		case err == io.EOF:
			return nil

		// return any other error
		case err != nil:
			return err

		// if the header is nil, just skip it (not sure how this happens)
		case header == nil:
			continue
		}

		target := filepath.Join(dst, header.Name)

		// TODO(#3)
		log.Printf("tar content: %s", target)

		// the following switch could also be done using fi.Mode(), not sure if there
		// a benefit of using one vs. the other.
		// fi := header.FileInfo()

		// check the file type
		switch header.Typeflag {

		// if its a dir and it doesn't exist create it
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0755); err != nil {
					return err
				}
			}

		// if it's a file create it
		case tar.TypeReg:
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			// copy over contents
			if _, err := io.Copy(f, tr); err != nil {
				return err
			}

			// manually close here after each file operation; defering would cause each file close
			// to wait until all operations have completed.
			err = f.Close()
			if err != nil {
				log.Printf("error closing created file: %s", err)
			}
		}
	}
}

// Unzip takes a destination path and a file and extracts it
func (a *Application) Unzip(zipFile, dst string) error {
	archive, err := zip.OpenReader(zipFile)
	if err != nil {
		panic(err)
	}
	defer func() {
		err := archive.Close()
		if err != nil {
			log.Printf("error closing zip archive: %s", err)
		}
	}()

	for _, f := range archive.File {
		filePath := filepath.Join(dst, f.Name)
		// TODO(#3)
		log.Println("unzipping file ", filePath)

		if !strings.HasPrefix(filePath, filepath.Clean(dst)+string(os.PathSeparator)) {
			return errors.New("invalid file path")
		}
		if f.FileInfo().IsDir() {
			// TODO(#3)
			log.Println("creating directory...")
			_ = os.MkdirAll(filePath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
			panic(err)
		}

		dstFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		fileInArchive, err := f.Open()
		if err != nil {
			return err
		}

		if _, err := io.Copy(dstFile, fileInArchive); err != nil {
			panic(err)
		}

		err = dstFile.Close()
		if err != nil {
			log.Printf("error closing extracted file: %s", err)
		}
		err = fileInArchive.Close()
		if err != nil {
			log.Printf("error closing in archive file: %s", err)
		}
	}
	return nil
}
