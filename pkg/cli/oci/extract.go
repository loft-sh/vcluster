package oci

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/layout"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

func Extract(archive, prefix, outDir string) error {
	lp := layout.Path(archive)
	idx, err := lp.ImageIndex()
	if err != nil {
		return fmt.Errorf("layout.ImageIndex: %w", err)
	}
	img, err := selectImageForRef(idx)
	if err != nil {
		return fmt.Errorf("select image: %w", err)
	}

	layers, err := img.Layers()
	if err != nil {
		return fmt.Errorf("get layers: %w", err)
	}

	// Normalize prefix like "/kubernetes" -> "kubernetes/"
	wantPrefix := strings.TrimPrefix(prefix, "/")
	if wantPrefix != "" && !strings.HasSuffix(wantPrefix, "/") {
		wantPrefix += "/"
	}

	// Process layers from TOP to BASE so newer wins, track whiteouts & seen paths.
	seen := map[string]bool{}
	var blockedPrefixes []string      // from .wh..wh..opq or dir deletes
	deletedExact := map[string]bool{} // from .wh.<name>

	shouldBlock := func(p string) bool {
		// exact deletions and directory-prefix blocks
		if _, ok := deletedExact[p]; ok {
			return true
		}
		for _, pre := range blockedPrefixes {
			if strings.HasPrefix(p, pre) {
				return true
			}
		}
		return false
	}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("create out dir: %w", err)
	}

	for i := len(layers) - 1; i >= 0; i-- {
		rc, err := layers[i].Uncompressed()
		if err != nil {
			return fmt.Errorf("get uncompressed layer: %w", err)
		}
		tr := tar.NewReader(rc)

		for {
			h, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				rc.Close()
				return fmt.Errorf("next: %w", err)
			}
			name := path.Clean(h.Name) // tar paths use forward slashes
			if strings.HasPrefix(path.Base(name), ".wh.") {
				base := path.Base(name)
				dir := path.Dir(name)
				if base == ".wh..wh..opq" {
					// block everything under this directory in lower layers
					prefix := strings.TrimSuffix(dir, "/")
					if prefix != "" {
						prefix += "/"
					}
					blockedPrefixes = append(blockedPrefixes, prefix)
				} else {
					target := path.Join(dir, strings.TrimPrefix(base, ".wh."))
					// block the exact path and anything under it (if it's a dir in lower layers)
					deletedExact[target] = true
					dirPrefix := strings.TrimSuffix(target, "/")
					if dirPrefix != "" {
						dirPrefix += "/"
						blockedPrefixes = append(blockedPrefixes, dirPrefix)
					}
				}
				continue
			}

			// We only care about entries under desired prefix.
			if wantPrefix != "" && !(name == strings.TrimSuffix(wantPrefix, "/") || strings.HasPrefix(name, wantPrefix)) {
				continue
			}

			// Normalize to absolute-ish for tracking, but keep tar-style
			key := "/" + name
			if _, ok := seen[key]; ok {
				continue // newer layer already materialized this path
			}
			if shouldBlock(name) {
				continue
			}

			switch h.Typeflag {
			case tar.TypeDir:
				// No need to create dirs unless we later place a file; mark as seen to avoid lower layers overriding perms.
				seen[key] = true
			case tar.TypeReg:
				rel := strings.TrimPrefix(name, strings.TrimSuffix(wantPrefix, "/"))
				rel = strings.TrimPrefix(rel, "/")
				dest := filepath.Join(outDir, rel)
				if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
					rc.Close()
					return fmt.Errorf("ensure dir: %w", err)
				}
				out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(h.Mode))
				if err != nil {
					rc.Close()
					return fmt.Errorf("open file: %w", err)
				}
				if _, err := io.Copy(out, tr); err != nil {
					out.Close()
					rc.Close()
					return fmt.Errorf("copy file: %w", err)
				}
				out.Close()
				seen[key] = true
			case tar.TypeSymlink:
				// Skip symlinks for "binaries" by default. If you want them, handle here.
				seen[key] = true
			default:
				seen[key] = true
			}
		}
		rc.Close()
	}

	return nil
}

// ExtractFile finds a single regular file at `filePath` inside the image and writes it
// to `outDir/<basename(filePath)>`. Whiteouts/opaque dirs are honored across layers.
// Returns os.ErrNotExist if the file does not exist (after considering deletions).
func ExtractFile(archive, filePath, outPath string) error {
	lp := layout.Path(archive)
	idx, err := lp.ImageIndex()
	if err != nil {
		return fmt.Errorf("layout.ImageIndex: %w", err)
	}
	img, err := selectImageForRef(idx)
	if err != nil {
		return fmt.Errorf("select image: %w", err)
	}

	layers, err := img.Layers()
	if err != nil {
		return fmt.Errorf("get layers: %w", err)
	}

	// Normalize wanted path like "/kubernetes/bin/kubelet" -> "kubernetes/bin/kubelet"
	want := strings.TrimPrefix(filePath, "/")
	want = path.Clean(want)
	if want == "." || want == "" {
		return fmt.Errorf("invalid file path")
	}

	// Process layers from TOP to BASE so newer wins.
	// Track whether higher layers delete/occlude our target so we can stop early.
	for i := len(layers) - 1; i >= 0; i-- {
		rc, err := layers[i].Uncompressed()
		if err != nil {
			return fmt.Errorf("get uncompressed layer: %w", err)
		}
		tr := tar.NewReader(rc)

		var deletedBelow bool // exact whiteout for the wanted path
		var blockedBelow bool // ancestor opq or dir deletion blocking lower layers

		for {
			h, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				rc.Close()
				return fmt.Errorf("next: %w", err)
			}

			name := path.Clean(h.Name) // tar paths use forward slashes

			// Handle whiteouts that affect ONLY lower layers.
			if strings.HasPrefix(path.Base(name), ".wh.") {
				base := path.Base(name)
				dir := path.Dir(name)

				if base == ".wh..wh..opq" {
					// This blocks all entries under `dir` in lower layers.
					prefix := strings.TrimPrefix(strings.TrimSuffix(dir, "/"), "/")
					if prefix != "" {
						prefix += "/"
					}
					if strings.HasPrefix(want, prefix) || want == strings.TrimSuffix(prefix, "/") {
						blockedBelow = true
					}
				} else {
					target := path.Join(dir, strings.TrimPrefix(base, ".wh."))
					target = strings.TrimPrefix(target, "/")

					// Exact deletion
					if target == want {
						deletedBelow = true
					}
					// If a whole directory was whiteouted, that blocks our wanted file in lower layers.
					dirPrefix := strings.TrimSuffix(target, "/")
					if dirPrefix != "" {
						dirPrefix += "/"
						if strings.HasPrefix(want, dirPrefix) {
							blockedBelow = true
						}
					}
				}
				continue
			}

			// If this layer contains the file as a regular file, extract and return immediately.
			if h.Typeflag == tar.TypeReg && name == want {
				// remove first to avoid still in use errors
				_ = os.Remove(outPath)
				out, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(h.Mode))
				if err != nil {
					rc.Close()
					return fmt.Errorf("open file: %w", err)
				}
				if _, err := io.Copy(out, tr); err != nil {
					out.Close()
					rc.Close()
					return fmt.Errorf("copy file: %w", err)
				}
				out.Close()
				rc.Close()
				return nil
			}

			// We ignore other entry types (dirs/symlinks/etc.) for a single-file extract.
		}

		rc.Close()

		// If a higher (newer) layer indicated our path/ancestor is deleted/opaque and
		// this layer didn't provide the file, lower layers can't provide it either.
		if deletedBelow || blockedBelow {
			return os.ErrNotExist
		}
	}

	// Never found it in any layer.
	return os.ErrNotExist
}

// pick the image for a given tag and prefer linux/amd64 if multi-platform
func selectImageForRef(idx v1.ImageIndex) (v1.Image, error) {
	im, err := idx.IndexManifest()
	if err != nil {
		return nil, err
	}

	if len(im.Manifests) == 0 {
		return nil, fmt.Errorf("no manifests in index")
	}

	desc := im.Manifests[0]
	if desc.Size == 0 {
		return nil, fmt.Errorf("no manifests in index")
	}

	switch desc.MediaType {
	case types.OCIImageIndex, types.DockerManifestList:
		child, err := idx.ImageIndex(desc.Digest)
		if err != nil {
			return nil, err
		}
		// recurse once: choose linux/amd64 child image
		cim, err := child.IndexManifest()
		if err != nil {
			return nil, err
		}
		desc2 := cim.Manifests[0]
		return child.Image(desc2.Digest)
	default: // image manifest
		return idx.Image(desc.Digest)
	}
}
