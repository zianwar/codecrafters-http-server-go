package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func getFileContent(filename string) ([]byte, error, bool) {
	path := filepath.Join(flDirectory, filename)

	// Return false if file doesn't exist
	if _, err := os.Stat(path); err != nil {
		return nil, nil, false
	}

	// Open file
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("unable to open file %s", path), false
	}
	defer file.Close()

	// Read file
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("unable to read file %s", path), false
	}

	return fileBytes, nil, true
}

func writeFileContent(filename string, content string) error {
	path := filepath.Join(flDirectory, filename)

	// Create file
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("unable to create file %s: %v", path, err)
	}
	defer file.Close()

	// Write to file
	_, err = file.Write([]byte(content))
	if err != nil {
		return fmt.Errorf("unable to write file %s", path)
	}

	return nil
}
