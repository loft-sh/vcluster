/*
Copyright The ORAS Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package file

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/opencontainers/go-digest"
)

// tarDirectory walks the directory specified by path, and tar those files with a new
// path prefix.
func tarDirectory(ctx context.Context, root, prefix string, w io.Writer, removeTimes bool, buf []byte) (err error) {
	tw := tar.NewWriter(w)
	defer func() {
		closeErr := tw.Close()
		if err == nil {
			err = closeErr
		}
	}()

	return filepath.Walk(root, func(path string, info os.FileInfo, err error) (returnErr error) {
		if err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Rename path
		name, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		name = filepath.Join(prefix, name)
		name = filepath.ToSlash(name)

		// Generate header
		// NOTE: We don't support hard links and treat it as regular files
		var link string
		mode := info.Mode()
		if mode&os.ModeSymlink != 0 {
			if link, err = os.Readlink(path); err != nil {
				return err
			}
		}
		header, err := tar.FileInfoHeader(info, link)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		header.Name = name
		header.Uid = 0
		header.Gid = 0
		header.Uname = ""
		header.Gname = ""

		if removeTimes {
			header.ModTime = time.Time{}
			header.AccessTime = time.Time{}
			header.ChangeTime = time.Time{}
		}

		// Write file
		if err := tw.WriteHeader(header); err != nil {
			return fmt.Errorf("tar: %w", err)
		}
		if mode.IsRegular() {
			fp, err := os.Open(path)
			if err != nil {
				return err
			}
			defer func() {
				closeErr := fp.Close()
				if returnErr == nil {
					returnErr = closeErr
				}
			}()

			if _, err := io.CopyBuffer(tw, fp, buf); err != nil {
				return fmt.Errorf("failed to copy to %s: %w", path, err)
			}
		}

		return nil
	})
}

// extractTarGzip decompresses the gzip
// and extracts tar file to a directory specified by the `dir` parameter.
func extractTarGzip(dirPath, dirName, gzPath, checksum string, buf []byte, preservePermissions bool) (err error) {
	fp, err := os.Open(gzPath)
	if err != nil {
		return err
	}
	defer func() {
		closeErr := fp.Close()
		if err == nil {
			err = closeErr
		}
	}()

	gzr, err := gzip.NewReader(fp)
	if err != nil {
		return err
	}
	defer func() {
		closeErr := gzr.Close()
		if err == nil {
			err = closeErr
		}
	}()

	var r io.Reader = gzr
	var verifier digest.Verifier
	if checksum != "" {
		if digest, err := digest.Parse(checksum); err == nil {
			verifier = digest.Verifier()
			r = io.TeeReader(r, verifier)
		}
	}
	if err := extractTarDirectory(dirPath, dirName, r, buf, preservePermissions); err != nil {
		return err
	}
	if verifier != nil && !verifier.Verified() {
		return errors.New("content digest mismatch")
	}
	return nil
}

// extractTarDirectory extracts tar file to a directory specified by the `dir`
// parameter. The file name prefix is ensured to be the string specified by the
// `prefix` parameter and is trimmed.
func extractTarDirectory(dirPath, dirName string, r io.Reader, buf []byte, preservePermissions bool) error {
	tr := tar.NewReader(r)
	for {
		header, err := tr.Next()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		// Name check
		filename := header.Name
		filePathRel, err := resolveRelToBase(dirPath, dirName, filename)
		if err != nil {
			return err
		}
		filePath := filepath.Join(dirPath, filePathRel)

		// Create content
		switch header.Typeflag {
		case tar.TypeReg:
			err = writeFile(filePath, tr, header.FileInfo().Mode(), buf)
		case tar.TypeDir:
			err = os.MkdirAll(filePath, header.FileInfo().Mode())
		case tar.TypeLink:
			// NOTE: ORAS does not generate hard links when creating tarballs.
			// If a hard link is found in the tarball, it will be extracted.
			// If the target link already exists, os.Link will throw an error.
			// This is a known limitation and will not be addressed.
			var target string
			if target, err = ensureLinkPath(dirPath, dirName, filePath, header.Linkname); err == nil {
				err = os.Link(target, filePath)
			}
		case tar.TypeSymlink:
			var target string
			target, err = ensureLinkPath(dirPath, dirName, filePath, header.Linkname)
			if err != nil {
				return err
			}
			if err = os.Symlink(target, filePath); err != nil {
				if !errors.Is(err, fs.ErrExist) {
					return err
				}
				// link already exists, remove the old one and try again
				if err := os.Remove(filePath); err != nil {
					return err
				}
				err = os.Symlink(target, filePath)
			}
		default:
			continue // Non-regular files are skipped
		}
		if err != nil {
			return err
		}

		// Change access time and modification time if possible (error ignored)
		_ = os.Chtimes(filePath, header.AccessTime, header.ModTime)

		// Restore full mode bits
		if preservePermissions && (header.Typeflag == tar.TypeReg || header.Typeflag == tar.TypeDir) {
			if err := os.Chmod(filePath, os.FileMode(header.Mode)); err != nil {
				return err
			}
		}
	}
}

// resolveRelToBase ensures the target path is in the base path,
// returning its relative path to the base path.
// target can be either an absolute path or a relative path.
func resolveRelToBase(baseAbs, baseRel, target string) (string, error) {
	base := baseRel
	if filepath.IsAbs(target) {
		// ensure base and target are consistent
		base = baseAbs
	}
	path, err := filepath.Rel(base, target)
	if err != nil {
		return "", err
	}
	cleanPath := filepath.ToSlash(filepath.Clean(path))
	if cleanPath == ".." || strings.HasPrefix(cleanPath, "../") {
		return "", fmt.Errorf("%q is outside of %q", target, baseRel)
	}

	// No symbolic link allowed in the relative path
	dir := filepath.Dir(path)
	for dir != "." {
		if info, err := os.Lstat(filepath.Join(baseAbs, dir)); err != nil {
			if !os.IsNotExist(err) {
				return "", err
			}
		} else if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("no symbolic link allowed between %q and %q", baseRel, target)
		}
		dir = filepath.Dir(dir)
	}

	return path, nil
}

// ensureLinkPath ensures the target path pointed by the link is in the base
// path. It returns target path if validated.
func ensureLinkPath(baseAbs, baseRel, link, target string) (string, error) {
	// resolve link
	path := target
	if !filepath.IsAbs(target) {
		path = filepath.Join(filepath.Dir(link), target)
	}
	// ensure path is under baseAbs or baseRel
	if _, err := resolveRelToBase(baseAbs, baseRel, path); err != nil {
		return "", err
	}
	return target, nil
}

// writeFile writes content to the file specified by the `path` parameter.
func writeFile(path string, r io.Reader, perm os.FileMode, buf []byte) (err error) {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	defer func() {
		closeErr := file.Close()
		if err == nil {
			err = closeErr
		}
	}()

	_, err = io.CopyBuffer(file, r, buf)
	return err
}
