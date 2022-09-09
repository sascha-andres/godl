package internal

import (
	"fmt"
	"io"
	"log"
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
			log.Printf("error closing http body: %s", err)
		}
	}()
	if res.StatusCode != 200 {
		return fmt.Errorf("status code error: %d %s", res.StatusCode, res.Status)
	}
	// TODO(#3)
	log.Println("downloaded")
	_, err = io.Copy(writer, res.Body)
	if err != nil {
		return fmt.Errorf("error writing bytes to file: %s", err)
	}
	return nil
}
