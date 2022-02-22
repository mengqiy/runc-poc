package images

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	cranev1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/cache"
	"github.com/google/go-containerregistry/pkg/v1/remote"

	"k8s.io/klog/v2"
)

type Store struct {
	baseDir string

	layerCache cache.Cache
}

func NewStore(baseDir string) (*Store, error) {
	cacheDir := filepath.Join(baseDir, "cache")

	layerCache := cache.NewFilesystemCache(cacheDir)
	return &Store{
		baseDir:    baseDir,
		layerCache: layerCache,
	}, nil
}

type cachedImage struct {
	Name       string   `json:"name"`
	Version    string   `json:"version"`
	Digest     string   `json:"digest"`
	Env        []string `json:"env"`
	Command    []string `json:"command"`
	Entrypoint []string `json:"entrypoint"`
	WorkingDir string   `json:"workingDir"`
}

func (s *Store) pullImage(ctx context.Context, ref name.Reference) (cranev1.Image, error) {
	var options []remote.Option
	options = append(options, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	img, err := remote.Image(ref, options...) //, o.remote...)

	if err != nil {
		return nil, fmt.Errorf("error pulling %s: %w", ref, err)
	}
	img = cache.Image(img, s.layerCache)

	return img, nil
}

func sanitize(image string) string {
	var sanitized strings.Builder
	for _, r := range image {
		if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			sanitized.WriteRune(r)
			continue
		}
		switch r {
		case '-', '_':
			sanitized.WriteRune(r)
		default:
			sanitized.WriteRune('-')
		}
	}
	return sanitized.String()
}

type Extracted struct {
	ImageName    string
	ExtractedDir string
	info         *cachedImage
}

func (e *Extracted) Env() []string {
	return e.info.Env
}

func (e *Extracted) WorkingDir() string {
	return e.info.WorkingDir
}

func (e *Extracted) Command() []string {
	return e.info.Command
}

func (e *Extracted) Entrypoint() []string {
	return e.info.Entrypoint
}

const cachedImageFormatVersion = "0.0.1"

func (s *Store) checkCached(ctx context.Context, ref name.Reference) (*cachedImage, error) {
	p := filepath.Join(s.baseDir, sanitize(ref.Name()))
	b, err := ioutil.ReadFile(p)
	if err != nil {
		return nil, err
	}

	cached := &cachedImage{}
	if err := json.Unmarshal(b, &cached); err != nil {
		return nil, err
	}

	if cached.Name != ref.Name() {
		return nil, fmt.Errorf("name mismatch in %s", p)
	}

	if cached.Version != cachedImageFormatVersion {
		return nil, fmt.Errorf("version was not expected version in %s", p)
	}

	return cached, nil
}

func (s *Store) writeToCache(ctx context.Context, ref name.Reference, img cranev1.Image, configFile *cranev1.ConfigFile) (*cachedImage, error) {
	p := filepath.Join(s.baseDir, sanitize(ref.Name()))

	digest, err := img.Digest()
	if err != nil {
		return nil, fmt.Errorf("error getting digest of image: %w", err)
	}

	info := &cachedImage{
		Name:       ref.Name(),
		Version:    cachedImageFormatVersion,
		Digest:     digest.Hex,
		Command:    configFile.Config.Cmd,
		Entrypoint: configFile.Config.Entrypoint,
		WorkingDir: configFile.Config.WorkingDir,
	}

	info.Env = configFile.Config.Env

	b, err := json.Marshal(&info)
	if err != nil {
		return nil, fmt.Errorf("error converting image info to json: %w", err)
	}

	if err := ioutil.WriteFile(p, b, 0644); err != nil {
		return nil, fmt.Errorf("error writing file %q: %w", p, err)
	}
	return info, nil
}

func (s *Store) Extract(ctx context.Context, imageName string) (*Extracted, error) {
	ref, err := name.ParseReference(imageName) //, o.name...)
	if err != nil {
		return nil, fmt.Errorf("error parsing image %q: %w", imageName, err)
	}

	sanitized := sanitize(ref.Name())

	var cached *cachedImage

	tag := ref.Identifier()
	if tag != "latest" { // always lookup ":latest" image
		cached, err = s.checkCached(ctx, ref)
		if err != nil {
			klog.V(2).Infof("ignoring error looking up image in cache: %v", err)
			cached = nil
		}
	}

	if cached != nil {
		imgHex := cached.Digest

		imageExtracted := filepath.Join(s.baseDir, sanitized+"_"+imgHex)

		stat, err := os.Stat(imageExtracted)
		if err != nil {
			if os.IsNotExist(err) {
				stat = nil
			} else {
				return nil, fmt.Errorf("error doing stat(%q): %w", imageExtracted, err)
			}
		}

		if stat != nil && stat.IsDir() {
			klog.V(2).Infof("image %s is cached at %s", imageName, imageExtracted)

			return &Extracted{
				ImageName:    imageName,
				ExtractedDir: imageExtracted,
				info:         cached,
			}, nil
		}
	}

	klog.Infof("pulling image %s", ref.Name())
	img, err := s.pullImage(ctx, ref)
	if err != nil {
		return nil, err
	}

	configFile, err := img.ConfigFile()
	if err != nil {
		return nil, fmt.Errorf("could not get config for image: %w", err)
	}

	imgHash, err := img.Digest()
	if err != nil {
		return nil, fmt.Errorf("could not get digest for image: %w", err)
	}

	imageExtracted := filepath.Join(s.baseDir, sanitized+"_"+imgHash.Hex)

	if err := s.extractImage(ctx, imageName, img, imageExtracted); err != nil {
		return nil, err
	}

	klog.Infof("image %s is at %s", imageName, imageExtracted)

	info, err := s.writeToCache(ctx, ref, img, configFile)
	if err != nil {
		return nil, err
	}

	return &Extracted{
		ImageName:    imageName,
		ExtractedDir: imageExtracted,
		info:         info,
	}, nil
}

func (i *Extracted) ResolveInPath(bin string) (string, error) {
	if filepath.IsAbs(bin) {
		// TODO: check exists etc?
		return bin, nil
	}
	envpath := "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
	for _, env := range i.info.Env {
		if strings.HasPrefix(env, "PATH=") {
			envpath = strings.TrimPrefix(env, "PATH=")
			break
		}
	}

	for _, pathDir := range filepath.SplitList(envpath) {
		pathDir = strings.TrimLeft(pathDir, string(filepath.Separator))

		p := filepath.Join(i.ExtractedDir, pathDir, bin)
		stat, err := os.Stat(p)
		if err != nil {
			if os.IsNotExist(err) {
				stat = nil
			} else {
				return "", fmt.Errorf("error from stat(%q): %w", p, err)
			}
		}
		// TODO: Check executable?
		if stat != nil && !stat.IsDir() {
			return filepath.Join(string(filepath.Separator)+pathDir, bin), nil
		}
	}
	return "", fmt.Errorf("unable to find %q in path %q for image %q", bin, envpath, i.ImageName)
}

func (s *Store) Pull(name string, os string, arch string) (cranev1.Image, error) {
	img, err := crane.Pull(name, crane.WithPlatform(&cranev1.Platform{
		OS:           os,
		Architecture: arch,
	}))
	if err != nil {
		return nil, fmt.Errorf("failed to pull %q: %w", name, err)
	}
	img = cache.Image(img, s.layerCache)
	return img, nil
}
