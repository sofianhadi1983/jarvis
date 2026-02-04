package clipboard

import (
	"os"
	"testing"
)

func TestInit(t *testing.T) {
	// Skip if no display available (CI environments)
	if os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
		// On macOS, clipboard should work without DISPLAY
		if os.Getenv("GITHUB_ACTIONS") == "true" {
			t.Skip("Skipping clipboard test in CI without display")
		}
	}

	err := Init()
	// Init may fail on systems without clipboard support (headless servers, etc.)
	// This is expected behavior, so we just check it doesn't panic
	if err != nil {
		t.Logf("Init returned error (expected on headless systems): %v", err)
	}
}

func TestIsInitialized(t *testing.T) {
	// Test that IsInitialized reflects the init state
	// Note: This test depends on the global state from TestInit
	_ = Init()
	// Just verify it returns a boolean without panicking
	_ = IsInitialized()
}

func TestReadImage_Empty(t *testing.T) {
	// Skip if clipboard not available
	if err := Init(); err != nil {
		t.Skip("Clipboard not available on this system")
	}

	// ReadImage should return nil when clipboard doesn't contain an image
	// We can't easily put an image in the clipboard from a test,
	// so we just verify the function doesn't panic
	data, err := ReadImage()
	if err != nil {
		t.Logf("ReadImage returned error: %v", err)
	}
	// data may be nil (no image) or contain data (if user has image in clipboard)
	_ = data
}
