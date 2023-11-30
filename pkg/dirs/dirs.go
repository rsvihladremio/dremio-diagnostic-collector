package dirs

import (
	"fmt"
	"io/fs"
	"os"
)

// CheckDirectory checks if a directory exists and contains files.
// It returns an error if the directory is empty, doesn't exist, isn't a directory,
// or if there's an error reading it.
func CheckDirectory(dirPath string, fileCheck func([]fs.DirEntry) bool) error {
	// Check if the directory exists
	fileInfo, err := os.Stat(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("directory does not exist")
		}
		return fmt.Errorf("error checking directory: %w", err)
	}

	// Check if the path is a directory
	if !fileInfo.IsDir() {
		return fmt.Errorf("the path is not a directory")
	}

	// Read the contents of the directory
	files, err := os.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("error reading directory: %w", err)
	}

	// Check if the directory is empty
	if len(files) == 0 {
		return fmt.Errorf("directory is empty")
	}

	if !fileCheck(files) {
		return fmt.Errorf("file check function failed")
	}
	return nil
}
