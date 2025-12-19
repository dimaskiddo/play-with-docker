package config

import (
	"log"
	"path/filepath"
)

func GetAbsoultePath(path string) string {
	if path == "" {
		log.Printf("Error getting absolute path")
		return path
	}

	if !filepath.IsAbs(path) {
		absPath, err := filepath.Abs(path)
		if err != nil {
			log.Printf("Error getting absolute path for %s: %v\n", path, err)
			return path
		} else {
			path = absPath
		}
	}

	return path
}
