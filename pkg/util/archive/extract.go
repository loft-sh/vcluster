package archive

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func ExtractTarGz(bundlePath, targetDir string) error {
	bundleReader, err := os.Open(bundlePath)
	if err != nil {
		return fmt.Errorf("failed to open bundle: %w", err)
	}

	uncompressedStream, err := gzip.NewReader(bundleReader)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader to extract bundle: %w", err)
	}

	return extract(uncompressedStream, targetDir)
}

func ExtractTar(bundlePath, targetDir string) error {
	bundleReader, err := os.Open(bundlePath)
	if err != nil {
		return fmt.Errorf("failed to open bundle: %w", err)
	}

	return extract(bundleReader, targetDir)
}

func extract(reader io.Reader, targetDir string) error {
	tarReader := tar.NewReader(reader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return fmt.Errorf("failed to get next tar header: %w", err)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.Mkdir(filepath.Join(targetDir, header.Name), 0755); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
		case tar.TypeReg:
			outFile, err := os.Create(filepath.Join(targetDir, header.Name))
			if err != nil {
				return fmt.Errorf("failed to create file %s: %w", header.Name, err)
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				return fmt.Errorf("failed to copy file %s: %w", header.Name, err)
			}
			outFile.Close()
			if err := os.Chmod(filepath.Join(targetDir, header.Name), header.FileInfo().Mode()); err != nil {
				return fmt.Errorf("failed to chmod file %s: %w", header.Name, err)
			}

		default:
			return fmt.Errorf("unknown type: %d in %s", header.Typeflag, header.Name)
		}
	}

	return nil
}
