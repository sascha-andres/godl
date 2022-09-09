package main

import (
	"errors"
	"fmt"
	"github.com/sascha-andres/flag"
	"github.com/sascha-andres/godl/internal"
	"io/fs"
	"log"
	"os"
	"path"
	"strings"
)

var (
	printVersions, download       bool
	version, destinationDirectory string
)

func init() {
	flag.BoolVar(&printVersions, "print", false, "use to print all versions for current os & arch")
	flag.BoolVar(&download, "download", false, "download provided version")
	flag.StringVar(&version, "version", "", "download this version")
	flag.StringVar(&destinationDirectory, "destination", "", "save version in this directory")
}

func main() {
	log.SetFlags(log.LstdFlags | log.LUTC | log.Lshortfile)
	flag.Parse()

	a, err := internal.NewApplication()
	if err != nil {
		log.Printf("error constructing application: %s", err)
		os.Exit(1)
	}

	if printVersions {
		for i := range a.Downloads {
			log.Println(a.Downloads[i].Url.String())
		}
		return
	}

	if download {
		err = downloadGoVersion(a)
		if err != nil {
			log.Printf("error downloading go: %s", err)
			os.Exit(1)
		}
	}
}

// downloadGoVersion will download selected go version
func downloadGoVersion(a *internal.Application) error {
	if version == "" {
		return fmt.Errorf("no version provided")
	}
	if destinationDirectory == "" {
		return fmt.Errorf("no destination provided")
	}
	downloadDestination := ""
	saveDestination := ""
	if i, err := os.Stat(destinationDirectory); errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("%s does not exist", destinationDirectory)
	} else {
		if !i.IsDir() {
			return fmt.Errorf("%s is not a directory", destinationDirectory)
		}
		downloadDestination = path.Join(destinationDirectory, fmt.Sprintf("_%s", version))
		saveDestination = path.Join(destinationDirectory, fmt.Sprintf("%s", version))
	}
	if _, err := os.Stat(downloadDestination); !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("%s already exists, manual cleanup may be required", downloadDestination)
	}
	// TODO(#3)
	log.Printf("downloading to %s", downloadDestination)
	err := os.MkdirAll(downloadDestination, 0700)
	if err != nil {
		return fmt.Errorf("error creating directory: %s", err)
	}
	d, err := a.GetDownload(version)
	if err != nil {
		return fmt.Errorf("error selecting download: %s", err)
	}
	downloadFile := path.Join(downloadDestination, d.FileName)
	download, err := os.Create(downloadFile)
	if err = d.DownloadGoArchive(download); err != nil {
		return fmt.Errorf("error downloading: %s", err)
	}
	err = download.Close()
	if err != nil {
		log.Printf("error closing downloaded file: %s", err)
	}
	if strings.HasSuffix(downloadFile, ".tar.gz") {
		f, err := os.Open(downloadFile)
		if err != nil {
			return fmt.Errorf("error opening downloaded archive: %s", err)
		}
		defer func() {
			err := f.Close()
			if err != nil {
				log.Printf("error closing tar archive: %s", err)
			}
		}()
		err = a.Untar(downloadDestination, f)
		if err != nil {
			return fmt.Errorf("error extracting downloaded archive: %s", err)
		}
	}

	if strings.HasSuffix(downloadFile, ".zip") {
		err := a.Unzip(downloadFile, downloadDestination)
		if err != nil {
			return fmt.Errorf("error extracting downloaded archive: %s", err)
		}
	}

	goDirectory := path.Join(downloadDestination, "go")

	if _, err := os.Stat(goDirectory); errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("%s expected but not found", goDirectory)
	}

	err = os.Rename(goDirectory, saveDestination)
	if err != nil {
		return fmt.Errorf("could not move do directory: %s", err)
	}

	err = os.RemoveAll(downloadDestination)
	if err != nil {
		return fmt.Errorf("could not remove download destination: %s", err)
	}
	return nil
}
