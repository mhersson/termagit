package platform

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCopyToClipboard_UnsupportedPlatform(t *testing.T) {
	if runtime.GOOS == "darwin" || runtime.GOOS == "linux" || runtime.GOOS == "windows" {
		t.Skip("test only runs on unsupported platforms")
	}
	err := CopyToClipboard("test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "clipboard not supported")
}

func TestOpen_UnsupportedPlatform(t *testing.T) {
	if runtime.GOOS == "darwin" || runtime.GOOS == "linux" || runtime.GOOS == "windows" {
		t.Skip("test only runs on unsupported platforms")
	}
	err := Open("https://example.com")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "open not supported")
}
