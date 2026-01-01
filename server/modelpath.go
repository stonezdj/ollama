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

	before, after, found := strings.Cut(name, "://")
	if found {
		mp.ProtocolScheme = before
		name = after
	}

	name = strings.ReplaceAll(name, string(os.PathSeparator), "/")

	// First, extract tag from the end (after last ":")
	// We need to ensure the ":" is after the last "/" to distinguish it from port numbers
	lastSlashIdx := strings.LastIndex(name, "/")
	lastColonIdx := strings.LastIndex(name, ":")
	if lastColonIdx > lastSlashIdx && lastColonIdx >= 0 {
		// The ":" is after the last "/", so it's a tag separator
		mp.Tag = name[lastColonIdx+1:]
		name = name[:lastColonIdx]
	}

	parts := strings.Split(name, "/")

	switch len(parts) {
	case 0:
		// Empty, shouldn't happen but handle gracefully
		return mp
	case 1:
		// Only repository name
		mp.Repository = parts[0]
	case 2:
		// Could be "namespace/repository" or "registry/repository"
		// Check if first part looks like a registry (has dots, colons, or is localhost)
		if strings.Contains(parts[0], ".") || strings.Contains(parts[0], ":") || parts[0] == "localhost" {
			mp.Registry = parts[0]
			mp.Repository = parts[1]
		} else {
			// Treat as namespace/repository
			mp.Namespace = parts[0]
			mp.Repository = parts[1]
		}
	default:
		// 3 or more parts: registry/namespace1/namespace2/.../repository
		// First part is registry if it looks like one
		if strings.Contains(parts[0], ".") || strings.Contains(parts[0], ":") || parts[0] == "localhost" {
			mp.Registry = parts[0]
			// Last part is repository, everything in between is namespace
			mp.Repository = parts[len(parts)-1]
			mp.Namespace = strings.Join(parts[1:len(parts)-1], "/")
		} else {
			// No registry, all parts before last are namespace
			mp.Repository = parts[len(parts)-1]
			mp.Namespace = strings.Join(parts[0:len(parts)-1], "/")
		}
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
