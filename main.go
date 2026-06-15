package main

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"log/slog"
	"os"
	"path"
	"runtime/debug"
	"strings"

	"github.com/sascha-andres/reuse/flag"

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

	var handlerOpts *slog.HandlerOptions
	if verbose {
		handlerOpts = &slog.HandlerOptions{Level: slog.LevelDebug}
	} else {
		handlerOpts = &slog.HandlerOptions{Level: slog.LevelInfo}
	}

	v := "unknown version"
	if i, ok := debug.ReadBuildInfo(); ok && i.Main.Version != "" {
		v = i.Main.Version
	}

	logger := slog.New(slog.NewJSONHandler(os.Stderr, handlerOpts)).With("project", "godl").With("version", v)
	slog.SetDefault(logger)

	logger.Debug("Starting importer")
	defer logger.Debug("Finished importer")

	var opts []internal.ApplicationOption
	if includeReleaseCandidates {
		opts = append(opts, internal.WithIncludeReleaseCandidates())
	}
	opts = append(opts, internal.WithLogger(logger))
	if verbose {
		opts = append(opts, internal.WithVerbose())
	}

	a, err := internal.NewApplication(opts...)
	if err != nil {
		logger.Error("error constructing application", "err", err)
		os.Exit(1)
	}

	if printVersions {
		for i := range a.Downloads {
			fmt.Println(a.Downloads[i].Url.String())
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

		downloadDestination, saveDestination, canSkip, err := getDestinationDirectories(logger)
		if err != nil {
			logger.Error("error getting destination directories", "err", err)
			os.Exit(1)
		}

		if !canSkip {
			logger.Debug("Starting download", "destination", downloadDestination)
			err = downloadGoVersion(a, downloadDestination, saveDestination)
			if err != nil {
				logger.Error("error downloading go", "err", err)
				os.Exit(1)
			}
			logger.Debug("download done")
		} else {
			logger.Debug("version exists, download skipped")
		}
	}

	if link {
		logger.Debug("Creating symlink")
		err = createSymLink()
		if err != nil {
			logger.Error("error creating symlink", "err", err)
			os.Exit(1)
		}
		logger.Debug("Symlink done")
	}
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
func getDestinationDirectories(logger *slog.Logger) (string, string, bool, error) {
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
				logger.Debug("skipping download, already downloaded")
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
