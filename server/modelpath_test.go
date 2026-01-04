package server

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetBlobsPath(t *testing.T) {
	// GetBlobsPath expects an actual directory to exist
	tempDir := t.TempDir()

	tests := []struct {
		name     string
		digest   string
		expected string
		err      error
	}{
		{
			"empty digest",
			"",
			filepath.Join(tempDir, "blobs"),
			nil,
		},
		{
			"valid with colon",
			"sha256:456402914e838a953e0cf80caa6adbe75383d9e63584a964f504a7bbb8f7aad9",
			filepath.Join(tempDir, "blobs", "sha256-456402914e838a953e0cf80caa6adbe75383d9e63584a964f504a7bbb8f7aad9"),
			nil,
		},
		{
			"valid with dash",
			"sha256-456402914e838a953e0cf80caa6adbe75383d9e63584a964f504a7bbb8f7aad9",
			filepath.Join(tempDir, "blobs", "sha256-456402914e838a953e0cf80caa6adbe75383d9e63584a964f504a7bbb8f7aad9"),
			nil,
		},
		{
			"digest too short",
			"sha256-45640291",
			"",
			ErrInvalidDigestFormat,
		},
		{
			"digest too long",
			"sha256-456402914e838a953e0cf80caa6adbe75383d9e63584a964f504a7bbb8f7aad9aaaaaaaaaa",
			"",
			ErrInvalidDigestFormat,
		},
		{
			"digest invalid chars",
			"../sha256-456402914e838a953e0cf80caa6adbe75383d9e63584a964f504a7bbb8f7a",
			"",
			ErrInvalidDigestFormat,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("OLLAMA_MODELS", tempDir)

			got, err := GetBlobsPath(tc.digest)

			require.ErrorIs(t, tc.err, err, tc.name)
			assert.Equal(t, tc.expected, got, tc.name)
		})
	}
}

func TestParseModelPath(t *testing.T) {
	tests := []struct {
		name string
		arg  string
		want ModelPath
	}{
		{
			"full path https",
			"https://example.com/ns/repo:tag",
			ModelPath{
				ProtocolScheme: "https",
				Registry:       "example.com",
				Namespace:      "ns",
				Repository:     "repo",
				Tag:            "tag",
			},
		},
		{
			"full path http",
			"http://example.com/ns/repo:tag",
			ModelPath{
				ProtocolScheme: "http",
				Registry:       "example.com",
				Namespace:      "ns",
				Repository:     "repo",
				Tag:            "tag",
			},
		},
		{
			"no protocol",
			"example.com/ns/repo:tag",
			ModelPath{
				ProtocolScheme: "https",
				Registry:       "example.com",
				Namespace:      "ns",
				Repository:     "repo",
				Tag:            "tag",
			},
		},
		{
			"no registry",
			"ns/repo:tag",
			ModelPath{
				ProtocolScheme: "https",
				Registry:       DefaultRegistry,
				Namespace:      "ns",
				Repository:     "repo",
				Tag:            "tag",
			},
		},
		{
			"no namespace",
			"repo:tag",
			ModelPath{
				ProtocolScheme: "https",
				Registry:       DefaultRegistry,
				Namespace:      DefaultNamespace,
				Repository:     "repo",
				Tag:            "tag",
			},
		},
		{
			"no tag",
			"repo",
			ModelPath{
				ProtocolScheme: "https",
				Registry:       DefaultRegistry,
				Namespace:      DefaultNamespace,
				Repository:     "repo",
				Tag:            DefaultTag,
			},
		},
		{
			"multi-level namespace with registry IP",
			"192.168.1.1/proxycache/library/qwen:0.5b",
			ModelPath{
				ProtocolScheme: "https",
				Registry:       "192.168.1.1",
				Namespace:      "proxycache",
				Repository:     "library/qwen",
				Tag:            "0.5b",
			},
		},
		{
			"multi-level namespace with registry domain",
			"registry.example.com/team/project/model:v1.0",
			ModelPath{
				ProtocolScheme: "https",
				Registry:       "registry.example.com",
				Namespace:      "team",
				Repository:     "project/model",
				Tag:            "v1.0",
			},
		},
		{
			"multi-level namespace with localhost and port",
			"localhost:5000/a/b/c/model:latest",
			ModelPath{
				ProtocolScheme: "https",
				Registry:       "localhost:5000",
				Namespace:      "a",
				Repository:     "b/c/model",
				Tag:            "latest",
			},
		},
		{
			"multi-level namespace with protocol",
			"https://registry.example.com/ns1/ns2/ns3/repo:tag",
			ModelPath{
				ProtocolScheme: "https",
				Registry:       "registry.example.com",
				Namespace:      "ns1",
				Repository:     "ns2/ns3/repo",
				Tag:            "tag",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ParseModelPath(tc.arg)

			if got != tc.want {
				t.Errorf("got: %q want: %q", got, tc.want)
			}
		})
	}
}
