package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"errors"
	"fmt"
	"github.com/sascha-andres/flag"
	"github.com/sascha-andres/godl/internal"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
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
	log.SetFlags(0)
	flag.Parse()

	a, err := internal.NewApplication()
	if err != nil {
		log.Printf("error constructing application: %s", err)
		os.Exit(1)
	}

	a.QueryVersions()

	if printVersions {
		for i := range a.Downloads {
			log.Println(a.Downloads[i].Url.String())
		}
		return
	}

	if download {
		if version == "" {
			log.Printf("no version provided")
			return
		}
		if destinationDirectory == "" {
			log.Printf("no destination provided")
			return
		}
		downloadDestination := ""
		saveDestination := ""
		if i, err := os.Stat(destinationDirectory); errors.Is(err, fs.ErrNotExist) {
			log.Printf("%s does not exist", destinationDirectory)
			return
		} else {
			if !i.IsDir() {
				log.Printf("%s is not a directory", destinationDirectory)
				return
			}
			downloadDestination = path.Join(destinationDirectory, fmt.Sprintf("_%s", version))
			saveDestination = path.Join(destinationDirectory, fmt.Sprintf("%s", version))
		}
		if _, err := os.Stat(downloadDestination); !errors.Is(err, fs.ErrNotExist) {
			log.Printf("%s already exists", downloadDestination)
		}
		log.Printf("downloading to %s", downloadDestination)
		err := os.MkdirAll(downloadDestination, 0700)
		if err != nil {
			log.Printf("error creating directory: %s", err)
		}
		d, err := a.GetDownload(version)
		if err != nil {
			log.Printf("error selecting download: %s", err)
			return
		}
		downloadFile := path.Join(downloadDestination, d.FileName)
		if !downloadGoArchive(err, d, downloadFile) {
			return
		}
		if strings.HasSuffix(downloadFile, ".tar.gz") {
			f, err := os.Open(downloadFile)
			if err != nil {
				log.Printf("error opening downloaded archive: %s", err)
				return
			}
			defer f.Close()
			err = Untar(downloadDestination, f)
			if err != nil {
				log.Printf("error extracting downloaded archive: %s", err)
				return
			}
		}

		if strings.HasSuffix(downloadFile, ".zip") {
			err := Unzip(downloadFile, downloadDestination)
			if err != nil {
				log.Printf("error extracting downloaded archive: %s", err)
				return
			}
		}

		godir := path.Join(downloadDestination, "go")

		if _, err := os.Stat(godir); errors.Is(err, fs.ErrNotExist) {
			log.Printf("%s expected but not found", godir)
			return
		}

		err = os.Rename(godir, saveDestination)
		if err != nil {
			log.Printf("could not move do directory: %s", err)
			return
		}

		err = os.RemoveAll(downloadDestination)
		if err != nil {
			log.Printf("could not remove download destination: %s", err)
		}
	}
}

func downloadGoArchive(err error, d *internal.Download, downloadFile string) bool {
	res, err := http.Get(d.Url.String())
	if err != nil {
		log.Printf("error in http: %s", err)
		return false
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Printf("status code error: %d %s", res.StatusCode, res.Status)
		return false
	}
	log.Println("downloaded")
	f, err := os.Create(downloadFile)
	if err != nil {
		log.Printf("error writing file to disk: %s", err)
		return false
	}
	defer f.Close()
	_, err = io.Copy(f, res.Body)
	if err != nil {
		log.Printf("error writing bytes to files: %s", err)
		return false
	}
	return true
}

// Untar takes a destination path and a reader; a tar reader loops over the tarfile
// creating the file structure at 'dst' along the way, and writing any files
func Untar(dst string, r io.Reader) error {

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
			f.Close()
		}
	}
}

// Untar takes a destination path and a reader; a tar reader loops over the tarfile
// creating the file structure at 'dst' along the way, and writing any files
func Unzip(zipFile, dst string) error {
	archive, err := zip.OpenReader(zipFile)
	if err != nil {
		panic(err)
	}
	defer archive.Close()

	for _, f := range archive.File {
		filePath := filepath.Join(dst, f.Name)
		log.Println("unzipping file ", filePath)

		if !strings.HasPrefix(filePath, filepath.Clean(dst)+string(os.PathSeparator)) {
			return errors.New("invalid file path")
		}
		if f.FileInfo().IsDir() {
			log.Println("creating directory...")
			_ = os.MkdirAll(filePath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
			panic(err)
		}

		dstFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			panic(err)
		}

		fileInArchive, err := f.Open()
		if err != nil {
			return err
		}

		if _, err := io.Copy(dstFile, fileInArchive); err != nil {
			panic(err)
		}

		dstFile.Close()
		fileInArchive.Close()
	}
	return nil
}
