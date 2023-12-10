package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

func ClearNodeDataDir() error {
	dir := "node_data"
	files, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("error reading node_data directory: %v", err)
	}

	for _, file := range files {
		err := os.Remove(filepath.Join(dir, file.Name()))
		if err != nil {
			return fmt.Errorf("error removing file %s: %v", file.Name(), err)
		}
	}

	return nil
}
