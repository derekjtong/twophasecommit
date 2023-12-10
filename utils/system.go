package utils

import (
	"bufio"
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

func WriteNodeInfoToFile(nodesInfo []string, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, info := range nodesInfo {
		_, err := file.WriteString(info + "\n")
		if err != nil {
			return err
		}
	}

	return nil
}

func ReadNodeInfoFromFile(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return lines, scanner.Err()
}
