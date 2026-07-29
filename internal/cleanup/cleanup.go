package cleanup

import (
	"log"
	"os"
	"path/filepath"
	"time"
)

// Start begins a background goroutine that runs every `interval`.
// It scans the specified storage directories and deletes any UUID-named
// folders (or files) that are older than `maxAge`.
func Start(storageDir string, maxAge time.Duration, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			runCleanup(storageDir, maxAge)
		}
	}()
}

// runCleanup executes the actual deletion logic.
func runCleanup(storageDir string, maxAge time.Duration) {
	threshold := time.Now().Add(-maxAge)
	dirsToClean := []string{"uploads", "output", "tools"}

	for _, subDir := range dirsToClean {
		targetDir := filepath.Join(storageDir, subDir)
		
		// Read all entries in the target directory
		entries, err := os.ReadDir(targetDir)
		if err != nil {
			if !os.IsNotExist(err) {
				log.Printf("cleanup: error reading dir %s: %v", targetDir, err)
			}
			continue
		}

		for _, entry := range entries {
			// Skip files (we expect UUID folders here, but maybe zip files too in tools)
			// But wait, the handler saves `tools/<id>/...` so entry is mostly a directory.
			// However, even if it's a file, we can check its age.
			info, err := entry.Info()
			if err != nil {
				continue
			}

			if info.ModTime().Before(threshold) {
				fullPath := filepath.Join(targetDir, entry.Name())
				if err := os.RemoveAll(fullPath); err != nil {
					log.Printf("cleanup: failed to delete %s: %v", fullPath, err)
				} else {
					log.Printf("cleanup: deleted old item %s", fullPath)
				}
			}
		}
	}
}
