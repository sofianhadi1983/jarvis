// Package clipboard provides functionality for reading images from the system clipboard.
package clipboard

import (
	"errors"
	"sync"

	"golang.design/x/clipboard"
)

var (
	initOnce    sync.Once
	initErr     error
	initialized bool
)

// Init initializes the clipboard. This must be called before using ReadImage.
// It is safe to call multiple times; initialization only happens once.
func Init() error {
	initOnce.Do(func() {
		initErr = clipboard.Init()
		if initErr == nil {
			initialized = true
		}
	})
	return initErr
}

// IsInitialized returns true if the clipboard has been successfully initialized.
func IsInitialized() bool {
	return initialized
}

// ErrNotInitialized is returned when clipboard operations are attempted
// before calling Init().
var ErrNotInitialized = errors.New("clipboard not initialized")

// ErrNoImage is returned when the clipboard does not contain image data.
var ErrNoImage = errors.New("no image in clipboard")

// ReadImage reads PNG image data from the clipboard.
// Returns nil, nil if the clipboard is empty or contains non-image data.
// Returns an error if the clipboard cannot be read.
func ReadImage() ([]byte, error) {
	if !initialized {
		if err := Init(); err != nil {
			return nil, err
		}
	}

	data := clipboard.Read(clipboard.FmtImage)
	if data == nil || len(data) == 0 {
		return nil, nil
	}

	return data, nil
}
