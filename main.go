package main

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path"
	"strings"

	"github.com/sascha-andres/flag"
	"github.com/sascha-andres/godl/internal"
)

var (
	printVersions, download, link, forceDownload    bool
	skipDownload, verbose, includeReleaseCandidates bool
	version, destinationDirectory, linkName         string
)

func init() {
	flag.SetEnvPrefix("GODL")

	flag.BoolVar(&printVersions, "print", false, "use to print all versions for current os & arch")
	flag.BoolVar(&verbose, "verbose", false, "more verbose output")
	flag.BoolVar(&download, "download", false, "download provided version")
	flag.BoolVar(&forceDownload, "force-download", false, "force new download")
	flag.BoolVar(&skipDownload, "skip-download", false, "skip download if it exists")
	flag.BoolVar(&link, "link", false, "link go version as linkname")
	flag.StringVar(&linkName, "link-name", "current", "name (path) of symlink")
	flag.StringVar(&version, "version", "", "download this version")
	flag.StringVar(&destinationDirectory, "destination", "", "save version in this directory")
	flag.BoolVar(&includeReleaseCandidates, "include-release-candidates", false, "specify to include release candidates")
}

func main() {
	log.SetFlags(log.LstdFlags | log.LUTC | log.Lshortfile)
	flag.Parse()

	var opts []internal.ApplicationOption
	if includeReleaseCandidates {
		opts = append(opts, internal.WithIncludeReleaseCandidates())
	}

	a, err := internal.NewApplication(opts...)
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
		if version == "" {
			log.Print("no version provided")
			os.Exit(1)
		}
		if destinationDirectory == "" {
			log.Print("no destination provided")
			os.Exit(1)
		}

		var downloadDestination, saveDestination string

		downloadDestination, saveDestination, canSkip, err := getDestinationDirectories()
		if err != nil {
			log.Printf("error getting destination directories: %s", err)
			os.Exit(1)
		}

		if !canSkip {
			logOnVerbose("Starting download")
			err = downloadGoVersion(a, downloadDestination, saveDestination)
			if err != nil {
				log.Printf("error downloading go: %s", err)
				os.Exit(1)
			}
			logOnVerbose("Download done")
		} else {
			logOnVerbose("version exists, download skipped")
		}
	}

	if link {
		logOnVerbose("Creating symlink")
		err = createSymLink()
		if err != nil {
			log.Printf("error creating symbolic link: %s", err)
			os.Exit(1)
		}
		logOnVerbose("Symlink done")
	}
}

// logOnVerbose calls log.Print to print to the standard logger.
// Arguments are handled in the manner of fmt.Print. It calls only
// // if verbose is active
func logOnVerbose(val string) {
	log.Print(val)
}

// logOnVerbosef calls log.Printf to print to the standard logger.
// Arguments are handled in the manner of fmt.Printf. It calls only
// if verbose is active
func logOnVerbosef(format string, v ...any) {
	log.Printf(format, v...)
}

// createSymLink creates a symlink for go version
func createSymLink() error {
	if "" == version {
		return errors.New("no version provided")
	}
	if destinationDirectory == "" {
		return errors.New("no destination provided")
	}

	saveDestination := path.Join(destinationDirectory, fmt.Sprintf("%s", version))
	if _, err := os.Stat(saveDestination); errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("no go version %s in %s", version, destinationDirectory)
	}

	return internal.Link(saveDestination, internal.CreateSymlinkPath(destinationDirectory, linkName))
}

// downloadGoVersion will download selected go version
func downloadGoVersion(a *internal.Application, downloadDestination string, saveDestination string) error {
	var err error
	var downloadFile *os.File
	var goDownload *internal.Download

	if err != nil {
		return err
	}
	logOnVerbosef("downloading to %s", downloadDestination)
	err = os.MkdirAll(downloadDestination, 0700)
	if err != nil {
		return fmt.Errorf("error creating directory: %s", err)
	}
	goDownload, err = a.GetDownload(version)
	if err != nil {
		return fmt.Errorf("error selecting download: %s", err)
	}
	downloadFileName := path.Join(downloadDestination, goDownload.FileName)
	downloadFile, err = os.Create(downloadFileName)
	if err = goDownload.DownloadGoArchive(downloadFile); err != nil {
		return fmt.Errorf("error downloading: %s", err)
	}
	err = downloadFile.Close()
	if err != nil {
		return fmt.Errorf("error closing downloaded file: %s", err)
	}
	if strings.HasSuffix(downloadFileName, ".tar.gz") {
		f, err := os.Open(downloadFileName)
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

	if strings.HasSuffix(downloadFileName, ".zip") {
		err := a.Unzip(downloadFileName, downloadDestination)
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

// getDestinationDirectories calculates destination directories: download dir and save dir, save to link and optionally an error
func getDestinationDirectories() (string, string, bool, error) {
	var downloadDestination, saveDestination string
	if i, err := os.Stat(destinationDirectory); errors.Is(err, fs.ErrNotExist) {
		return "", "", false, fmt.Errorf("%s does not exist", destinationDirectory)
	} else {
		if !i.IsDir() {
			return "", "", false, fmt.Errorf("%s is not a directory", destinationDirectory)
		}
		downloadDestination = path.Join(destinationDirectory, fmt.Sprintf("_%s", version))
		saveDestination = path.Join(destinationDirectory, fmt.Sprintf("%s", version))
	}
	if _, err := os.Stat(downloadDestination); !errors.Is(err, fs.ErrNotExist) {
		return "", "", false, fmt.Errorf("%s already exists, manual cleanup may be required", downloadDestination)
	}
	if _, err := os.Stat(saveDestination); !errors.Is(err, fs.ErrNotExist) {
		if !forceDownload {
			if skipDownload {
				logOnVerbose("skipping download, already downloaded")
				return "", "", true, nil
			}
			return "", "", false, fmt.Errorf("%s already exists, not downloading. To set symbolic link, call without -download", saveDestination)
		}
		err = os.RemoveAll(saveDestination)
		if err != nil {
			return "", "", false, fmt.Errorf("%s already existed and could not be removed", downloadDestination)
		}
	}
	return downloadDestination, saveDestination, false, nil
}
