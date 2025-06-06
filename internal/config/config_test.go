package config_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/atinyakov/go-url-shortener/internal/config"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	t.Run("no env, no config", func(t *testing.T) {
		os.Clearenv()
		opts := config.Parse()
		require.Equal(t, "localhost:8080", opts.Port)
		require.Equal(t, "http://localhost:8080", opts.ResultHostname)
		require.Equal(t, "", opts.FilePath)
		require.False(t, opts.EnableHTTPS)
		require.False(t, opts.EnablePprof)
		require.Equal(t, "config.json", opts.Config)
	})

	t.Run("env overrides", func(t *testing.T) {
		os.Clearenv()
		os.Setenv("SERVER_ADDRESS", "127.0.0.1:9999")
		os.Setenv("BASE_URL", "http://example.com")
		os.Setenv("FILE_STORAGE_PATH", "/tmp/data")
		os.Setenv("ENABLE_HTTPS", "true")
		os.Setenv("TRUSTED_SUBNET", "192.168.0.0/24")
		os.Unsetenv("CONFIG")

		opts := config.Parse()
		require.Equal(t, "127.0.0.1:9999", opts.Port)
		require.Equal(t, "http://example.com", opts.ResultHostname)
		require.Equal(t, "/tmp/data", opts.FilePath)
		require.True(t, opts.EnableHTTPS)
		require.Equal(t, "192.168.0.0/24", opts.TrustedSubnet)
	})

	t.Run("config file overrides", func(t *testing.T) {
		os.Clearenv() // ✅ This clears previous env vars

		tmpDir, err := tryMkdirTemp()
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		cfgPath := filepath.Join(tmpDir, "cfg.json")
		cfg := config.Options{
			Port:           "10.0.0.1:8081",
			ResultHostname: "http://testhost",
			FilePath:       "/config/path",
			DatabaseDSN:    "postgres://test",
			EnablePprof:    true,
			EnableHTTPS:    true,
			TrustedSubnet:  "10.10.0.0/16",
		}
		content, _ := json.Marshal(cfg)
		require.NoError(t, os.WriteFile(cfgPath, content, 0644))
		os.Setenv("CONFIG", cfgPath)

		opts := config.Parse()
		require.Equal(t, "10.0.0.1:8081", opts.Port)
		require.Equal(t, "http://testhost", opts.ResultHostname)
		require.Equal(t, "/config/path", opts.FilePath)
		require.Equal(t, "postgres://test", opts.DatabaseDSN)
		require.True(t, opts.EnablePprof)
		require.True(t, opts.EnableHTTPS)
		require.Equal(t, "10.10.0.0/16", opts.TrustedSubnet)
	})
}

// tryMkdirTemp attempts to create a temporary directory in various fallback locations.
func tryMkdirTemp() (string, error) {
	// First try the system temp dir
	tmpDir, err := os.MkdirTemp("", "testconfig")
	if err == nil {
		return tmpDir, nil
	}

	// Try user cache dir
	if fallbackBase, err2 := os.UserCacheDir(); err2 == nil {
		tmpDir, err := os.MkdirTemp(fallbackBase, "testconfig")
		if err == nil {
			return tmpDir, nil
		}
	}

	// Fall back to current directory
	return os.MkdirTemp(".", "testconfig")
}
