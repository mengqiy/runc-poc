package images

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/google/go-containerregistry/pkg/crane"
	cranev1 "github.com/google/go-containerregistry/pkg/v1"

	"k8s.io/klog/v2"
)

func (s *Store) extractImage(ctx context.Context, imageName string, img cranev1.Image, destDir string) error {
	stat, err := os.Stat(destDir)
	if err != nil {
		if os.IsNotExist(err) {
			stat = nil
		} else {
			return fmt.Errorf("error checking stat of %q: %w", destDir, err)
		}
	}

	if stat == nil {
		klog.Infof("extracting image %s", imageName)

		exportErr := make(chan error)
		pr, pw := io.Pipe()
		go func() {
			defer pw.Close()
			exportErr <- crane.Export(img, pw)
		}()

		defer pr.Close()

		tr := tar.NewReader(pr)
		if err := os.MkdirAll(filepath.Dir(destDir), 0755); err != nil {
			return fmt.Errorf("failed to create directory %q: %w", filepath.Dir(destDir), err)
		}
		tempDir, err := ioutil.TempDir(filepath.Dir(destDir), "kontained")
		if err != nil {
			return fmt.Errorf("failed to create tempdir for image: %w", err)
		}

		if err := untar(tr, tempDir); err != nil {
			os.RemoveAll(tempDir)
			return fmt.Errorf("failed to extract image: %w", err)
		}

		if err := <-exportErr; err != nil {
			os.RemoveAll(tempDir)
			return fmt.Errorf("error extracting image: %w", err)
		}

		if err := os.Rename(tempDir, destDir); err != nil {
			os.RemoveAll(tempDir)
			return fmt.Errorf("failed to rename extraction tempdir %q -> %q: %w", tempDir, destDir, err)
		}
	}
	return nil
}

// Based on https://pkg.go.dev/golang.org/x/build/internal/untar#Untar
func untar(tr *tar.Reader, dir string) (err error) {
	dirAbs, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for %q: %w", dir, err)
	}
	dirAbs = filepath.Clean(dirAbs)
	dirAbs = dirAbs + string(filepath.Separator)

	madeDir := map[string]bool{}

	for {
		f, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading tar entry: %w", err)
		}

		if !validRelPath(f.Name) {
			return fmt.Errorf("tar contained invalid name error %q", f.Name)
		}
		rel := filepath.FromSlash(f.Name)
		abs := filepath.Join(dir, rel)

		fi := f.FileInfo()
		mode := fi.Mode()
		switch {
		case mode.IsRegular():
			// Make the directory. This is redundant because it should
			// already be made by a directory entry in the tar
			// beforehand. Thus, don't check for errors; the next
			// write will fail with the same error.
			dir := filepath.Dir(abs)
			if !madeDir[dir] {
				if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
					return fmt.Errorf("failed to make directory %q: %w", abs, err)
				}
				madeDir[dir] = true
			}
			wf, err := os.OpenFile(abs, os.O_RDWR|os.O_CREATE|os.O_TRUNC, mode.Perm())
			if err != nil {
				return err
			}
			n, err := io.Copy(wf, tr)
			if closeErr := wf.Close(); closeErr != nil && err == nil {
				err = closeErr
			}
			if err != nil {
				return fmt.Errorf("error writing to %s: %w", abs, err)
			}
			if n != f.Size {
				return fmt.Errorf("only wrote %d bytes to %s; expected %d", n, abs, f.Size)
			}

		case mode.IsDir():
			if err := os.MkdirAll(abs, 0755); err != nil {
				return fmt.Errorf("failed to make directory %q: %w", abs, err)
			}
			madeDir[abs] = true

		default:
			if mode.Type() == fs.ModeSymlink {
				targetRel := filepath.FromSlash(f.Linkname)
				// This is relatively safe because we will chroot

				targetAbs := filepath.Clean(filepath.Join(dir, filepath.Dir(f.Name), targetRel))
				if !strings.HasPrefix(targetAbs, dirAbs) {
					return fmt.Errorf("symlink %q -> %q (=> %q) was outside of target directory %q", f.Name, targetRel, targetAbs, dir)
				}

				if err := os.Symlink(targetRel, abs); err != nil {
					return fmt.Errorf("failed to make symlink %q -> %q: %w", abs, targetRel, err)
				}
			} else {
				klog.Warningf("skipping tar file entry %s with unsupported file type %v", f.Name, mode)
			}
		}
	}
	return nil
}

func validRelativeDir(dir string) bool {
	if strings.Contains(dir, `\`) || path.IsAbs(dir) {
		return false
	}
	dir = path.Clean(dir)
	if strings.HasPrefix(dir, "../") || strings.HasSuffix(dir, "/..") || dir == ".." {
		return false
	}
	return true
}

func validRelPath(p string) bool {
	if p == "" || strings.Contains(p, `\`) || strings.HasPrefix(p, "/") || strings.Contains(p, "../") {
		return false
	}
	return true
}
