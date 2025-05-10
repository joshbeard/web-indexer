package webindexer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"
)

type LocalBackend struct {
	path string
	cfg  Config
}

var _ FileSource = &LocalBackend{}

func (l *LocalBackend) Read(path string) ([]*Item, bool, error) {
	var items []*Item
	log.Debugf("Listing files in %s", path)
	files, err := os.ReadDir(path)
	if err != nil {
		return nil, false, fmt.Errorf("unable to read source path %s: %w", path, err)
	}

	// First check for noindex files before processing anything else
	for _, file := range files {
		if !file.IsDir() {
			// Check for noindex files (skip directory entirely)
			if len(l.cfg.NoIndexFiles) > 0 && contains(l.cfg.NoIndexFiles, file.Name()) {
				log.Infof("Skipping %s (found noindex file %s)", path, file.Name())
				return nil, true, nil
			}

			// Check for skipindex files (skip indexing but include in parent)
			if len(l.cfg.SkipIndexFiles) > 0 && contains(l.cfg.SkipIndexFiles, file.Name()) {
				log.Infof("Skipping indexing of %s (found skipindex file %s), will include in parent directory", path, file.Name())
				// Return empty items but mark as not having noindex file
				// This will prevent indexing this directory but still include it in the parent
				return []*Item{}, false, nil
			}
		}
	}

	// Process all other files
	for _, file := range files {
		if shouldSkip(file.Name(), l.cfg.IndexFile, l.cfg.Skips) {
			continue
		}

		fullPath := filepath.Join(path, file.Name())
		stat, err := os.Stat(fullPath)
		if err != nil {
			return nil, false, fmt.Errorf("unable to stat file %s: %w", file.Name(), err)
		}

		// If it's a directory, check if it contains a noindex or skipindex file before adding it
		if stat.IsDir() {
			subFiles, err := os.ReadDir(fullPath)
			if err != nil {
				return nil, false, fmt.Errorf("unable to read directory %s: %w", fullPath, err)
			}

			// Skip this directory if it contains a noindex file
			skipDir := false
			for _, subFile := range subFiles {
				if !subFile.IsDir() && len(l.cfg.NoIndexFiles) > 0 && contains(l.cfg.NoIndexFiles, subFile.Name()) {
					log.Infof("Skipping %s (found noindex file %s)", fullPath, subFile.Name())
					skipDir = true
					break
				}
			}
			if skipDir {
				continue
			}
		}

		itemName := file.Name()
		item := &Item{
			Name:         itemName,
			Size:         stat.Size(),
			LastModified: stat.ModTime(),
			IsDir:        stat.IsDir(),
			HasMetadata:  true,
		}

		items = append(items, item)
	}

	return items, false, nil
}

func (l *LocalBackend) EnsureDirExists(relativePath string) error {
	localPath := filepath.Join(l.cfg.Target, relativePath)
	if err := os.MkdirAll(localPath, 0o750); err != nil {
		return fmt.Errorf("failed to ensure directory exists %s: %w", localPath, err)
	}
	log.Debugf("Ensured directory exists: %s", localPath)
	return nil
}

func (l *LocalBackend) Write(data Data, content string) error {
	prefix := data.RelativePath
	prefix = strings.TrimPrefix(prefix, l.cfg.BasePath)

	// Remove any leading slashes to avoid creating unnecessary subdirectories
	prefix = strings.TrimPrefix(prefix, "/")

	// For the root directory, don't create an additional subdirectory
	if prefix == "" || prefix == "/" {
		localPath := l.cfg.Target
		if err := os.MkdirAll(localPath, 0o750); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", localPath, err)
		}

		filePath := filepath.Join(localPath, l.cfg.IndexFile)
		file, err := os.Create(filePath) // #nosec
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = file.WriteString(content)
		if err != nil {
			return err
		}

		log.Infof("Generated %s", filePath)
		return nil
	}

	// For subdirectories, create the necessary structure
	localPath := filepath.Join(l.cfg.Target, prefix)
	if err := os.MkdirAll(localPath, 0o750); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", localPath, err)
	}

	filePath := filepath.Join(localPath, l.cfg.IndexFile)
	file, err := os.Create(filePath) // #nosec
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(content)
	if err != nil {
		return err
	}

	log.Infof("Generated %s", filePath)
	return nil
}
