package internal

import (
	"fmt"
	"io"
	"net/http"
)

// DownloadGoArchive saves a Go release archive to given
func (d *Download) DownloadGoArchive(writer io.Writer) error {
	res, err := http.Get(d.Url.String())
	if err != nil {
		return fmt.Errorf("error in http: %s", err)
	}
	defer func() {
		err := res.Body.Close()
		if err != nil {
			d.Logger.Warn("error closing http body", "err", err)
		}
	}()
	if res.StatusCode != 200 {
		return fmt.Errorf("status code error: %d %s", res.StatusCode, res.Status)
	}
	_, err = io.Copy(writer, res.Body)
	if err != nil {
		return fmt.Errorf("error writing bytes to file: %s", err)
	}
	return nil
}
