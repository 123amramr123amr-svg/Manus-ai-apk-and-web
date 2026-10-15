// Command syncassets copies the standalone agent frontend (web/agent) into the
// Android application assets, so the APK can run the agent without any server.
//
// Usage:
//
//	go run ./tools/syncassets
//
// The Go test TestAgentAssetsMatchAndroidCopy fails when the copies drift.
package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

const (
	sourceDir = "web/agent"
	targetDir = "android/app/src/main/assets/agent"
)

func main() {
	copied, err := sync()
	if err != nil {
		fmt.Fprintf(os.Stderr, "syncassets: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("synced %d files: %s -> %s\n", copied, sourceDir, targetDir)
}

func sync() (int, error) {
	if _, err := os.Stat(sourceDir); err != nil {
		return 0, fmt.Errorf("source directory: %w", err)
	}
	if err := os.RemoveAll(targetDir); err != nil {
		return 0, fmt.Errorf("clean target: %w", err)
	}

	copied := 0
	err := filepath.WalkDir(sourceDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		destination := filepath.Join(targetDir, relative)

		if entry.IsDir() {
			return os.MkdirAll(destination, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(destination, data, 0o644); err != nil {
			return err
		}
		copied++
		return nil
	})
	if err != nil {
		return copied, err
	}
	return copied, nil
}
