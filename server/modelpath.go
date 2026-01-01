package server

import (
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ollama/ollama/envconfig"
	"github.com/ollama/ollama/types/model"
)

type ModelPath struct {
	ProtocolScheme string
	Registry       string
	Namespace      string
	Repository     string
	Tag            string
}

const (
	DefaultRegistry       = "registry.ollama.ai"
	DefaultNamespace      = "library"
	DefaultTag            = "latest"
	DefaultProtocolScheme = "https"
)

var (
	ErrInvalidImageFormat  = errors.New("invalid image format")
	ErrInvalidDigestFormat = errors.New("invalid digest format")
	ErrInvalidProtocol     = errors.New("invalid protocol scheme")
	ErrInsecureProtocol    = errors.New("insecure protocol http")
	ErrModelPathInvalid    = errors.New("invalid model path")
)

func ParseModelPath(name string) ModelPath {
	mp := ModelPath{
		ProtocolScheme: DefaultProtocolScheme,
		Registry:       DefaultRegistry,
		Namespace:      DefaultNamespace,
		Repository:     "",
		Tag:            DefaultTag,
	}

	// Extract protocol scheme if present
	before, after, found := strings.Cut(name, "://")
	if found {
		mp.ProtocolScheme = before
		name = after
	}

	name = strings.ReplaceAll(name, string(os.PathSeparator), "/")

	// Use ParseNameBare to parse the name, then map the fields
	n := model.ParseNameBare(name)

	// Map parsed name fields to model path
	// ParseNameBare treats the first part as Host, but we need to check if it's actually a registry
	// A registry typically contains dots (domain), colons (port), or is "localhost"
	if n.Host != "" && (strings.Contains(n.Host, ".") || strings.Contains(n.Host, ":") || n.Host == "localhost") {
		mp.Registry = n.Host
		if n.Namespace != "" {
			mp.Namespace = n.Namespace
		}
	} else if n.Host != "" {
		// First part doesn't look like a registry, so it's part of the namespace
		if n.Namespace != "" {
			mp.Namespace = n.Host + "/" + n.Namespace
		} else {
			mp.Namespace = n.Host
		}
	} else if n.Namespace != "" {
		mp.Namespace = n.Namespace
	}

	if n.Model != "" {
		mp.Repository = n.Model
	}
	if n.Tag != "" {
		mp.Tag = n.Tag
	}

	return mp
}

func (mp ModelPath) GetNamespaceRepository() string {
	return fmt.Sprintf("%s/%s", mp.Namespace, mp.Repository)
}

func (mp ModelPath) GetFullTagname() string {
	return fmt.Sprintf("%s/%s/%s:%s", mp.Registry, mp.Namespace, mp.Repository, mp.Tag)
}

func (mp ModelPath) GetShortTagname() string {
	if mp.Registry == DefaultRegistry {
		if mp.Namespace == DefaultNamespace {
			return fmt.Sprintf("%s:%s", mp.Repository, mp.Tag)
		}
		return fmt.Sprintf("%s/%s:%s", mp.Namespace, mp.Repository, mp.Tag)
	}
	return fmt.Sprintf("%s/%s/%s:%s", mp.Registry, mp.Namespace, mp.Repository, mp.Tag)
}

// GetManifestPath returns the path to the manifest file for the given model path, it is up to the caller to create the directory if it does not exist.
func (mp ModelPath) GetManifestPath() (string, error) {
	name := model.Name{
		Host:      mp.Registry,
		Namespace: mp.Namespace,
		Model:     mp.Repository,
		Tag:       mp.Tag,
	}
	if !name.IsValid() {
		return "", fs.ErrNotExist
	}
	return filepath.Join(envconfig.Models(), "manifests", name.Filepath()), nil
}

func (mp ModelPath) BaseURL() *url.URL {
	return &url.URL{
		Scheme: mp.ProtocolScheme,
		Host:   mp.Registry,
	}
}

func GetManifestPath() (string, error) {
	path := filepath.Join(envconfig.Models(), "manifests")
	if err := os.MkdirAll(path, 0o755); err != nil {
		return "", fmt.Errorf("%w: ensure path elements are traversable", err)
	}

	return path, nil
}

func GetBlobsPath(digest string) (string, error) {
	// only accept actual sha256 digests
	pattern := "^sha256[:-][0-9a-fA-F]{64}$"
	re := regexp.MustCompile(pattern)

	if digest != "" && !re.MatchString(digest) {
		return "", ErrInvalidDigestFormat
	}

	digest = strings.ReplaceAll(digest, ":", "-")
	path := filepath.Join(envconfig.Models(), "blobs", digest)
	dirPath := filepath.Dir(path)
	if digest == "" {
		dirPath = path
	}

	if err := os.MkdirAll(dirPath, 0o755); err != nil {
		return "", fmt.Errorf("%w: ensure path elements are traversable", err)
	}

	return path, nil
}
