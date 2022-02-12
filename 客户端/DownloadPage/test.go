package main

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
)

func main() {
	err := Unzip1("./DownloadPage.zip", "./DownloadPage")
	if err != nil {
		return
	}
}

func Unzip1(zipFile string, destDir string) error {
	zipReader, err := zip.OpenReader(zipFile)
	if err != nil {
		return err
	}
	defer func(zipReader *zip.ReadCloser) {
		err := zipReader.Close()
		if err != nil {

		}
	}(zipReader)

	for _, f := range zipReader.File {
		fpath := filepath.Join(destDir, f.Name)
		if f.FileInfo().IsDir() {
			err := os.MkdirAll(fpath, os.ModePerm)
			if err != nil {
				return err
			}
		} else {
			if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
				return err
			}

			inFile, err := f.Open()
			if err != nil {
				return err
			}
			defer func(inFile io.ReadCloser) {
				err := inFile.Close()
				if err != nil {

				}
			}(inFile)

			outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer func(outFile *os.File) {
				err := outFile.Close()
				if err != nil {

				}
			}(outFile)

			_, err = io.Copy(outFile, inFile)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
