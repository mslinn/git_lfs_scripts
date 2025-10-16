package checksum

import (
	"encoding/json"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/mslinn/git_lfs_scripts/pkg/database"
)

// FileChecksum represents a file's checksum and metadata
type FileChecksum struct {
	Path      string
	CRC32     uint32
	SizeBytes int64
}

// ComputeFile computes the CRC32 checksum for a single file
func ComputeFile(path string) (*FileChecksum, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	hash := crc32.NewIEEE()
	if _, err := io.Copy(hash, file); err != nil {
		return nil, fmt.Errorf("failed to compute checksum: %w", err)
	}

	return &FileChecksum{
		Path:      path,
		CRC32:     hash.Sum32(),
		SizeBytes: info.Size(),
	}, nil
}

// ComputeDirectory recursively computes checksums for all files in a directory
// It skips .git directories and the .checksums file
func ComputeDirectory(dir string) ([]*FileChecksum, error) {
	var checksums []*FileChecksum

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			// Skip .git directories
			if info.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip .checksums file
		if info.Name() == ".checksums" {
			return nil
		}

		// Compute checksum for regular files
		cs, err := ComputeFile(path)
		if err != nil {
			return fmt.Errorf("failed to compute checksum for %s: %w", path, err)
		}

		// Store relative path
		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			relPath = path
		}
		cs.Path = relPath

		checksums = append(checksums, cs)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	// Sort by path for consistent ordering
	sort.Slice(checksums, func(i, j int) bool {
		return checksums[i].Path < checksums[j].Path
	})

	return checksums, nil
}

// StoreChecksums stores checksums in the database
func StoreChecksums(db *database.DB, runID int64, stepNumber int, checksums []*FileChecksum) error {
	now := time.Now()

	for _, cs := range checksums {
		dbChecksum := &database.Checksum{
			RunID:      runID,
			StepNumber: stepNumber,
			FilePath:   cs.Path,
			CRC32:      fmt.Sprintf("%08x", cs.CRC32),
			SizeBytes:  cs.SizeBytes,
			ComputedAt: now,
		}

		if err := db.CreateChecksum(dbChecksum); err != nil {
			return fmt.Errorf("failed to store checksum for %s: %w", cs.Path, err)
		}
	}

	return nil
}

// Difference represents a checksum difference between two steps
type Difference struct {
	FilePath    string
	OldCRC32    string
	OldSize     int64
	NewCRC32    string
	NewSize     int64
	ChangeType  string // "added", "modified", "deleted", "size-changed"
}

// CompareChecksums compares checksums between two steps
func CompareChecksums(db *database.DB, runID int64, oldStep, newStep int) ([]*Difference, error) {
	oldChecksums, err := db.ListChecksums(runID, oldStep)
	if err != nil {
		return nil, fmt.Errorf("failed to get checksums for step %d: %w", oldStep, err)
	}

	newChecksums, err := db.ListChecksums(runID, newStep)
	if err != nil {
		return nil, fmt.Errorf("failed to get checksums for step %d: %w", newStep, err)
	}

	// Create maps for easy lookup
	oldMap := make(map[string]*database.Checksum)
	for _, cs := range oldChecksums {
		oldMap[cs.FilePath] = cs
	}

	newMap := make(map[string]*database.Checksum)
	for _, cs := range newChecksums {
		newMap[cs.FilePath] = cs
	}

	var diffs []*Difference

	// Find modified and deleted files
	for path, oldCS := range oldMap {
		newCS, exists := newMap[path]
		if !exists {
			// File was deleted
			diffs = append(diffs, &Difference{
				FilePath:   path,
				OldCRC32:   oldCS.CRC32,
				OldSize:    oldCS.SizeBytes,
				ChangeType: "deleted",
			})
		} else if oldCS.CRC32 != newCS.CRC32 {
			// File was modified
			changeType := "modified"
			if oldCS.SizeBytes != newCS.SizeBytes {
				changeType = "size-changed"
			}
			diffs = append(diffs, &Difference{
				FilePath:   path,
				OldCRC32:   oldCS.CRC32,
				OldSize:    oldCS.SizeBytes,
				NewCRC32:   newCS.CRC32,
				NewSize:    newCS.SizeBytes,
				ChangeType: changeType,
			})
		}
	}

	// Find added files
	for path, newCS := range newMap {
		if _, exists := oldMap[path]; !exists {
			diffs = append(diffs, &Difference{
				FilePath:   path,
				NewCRC32:   newCS.CRC32,
				NewSize:    newCS.SizeBytes,
				ChangeType: "added",
			})
		}
	}

	// Sort by path for consistent output
	sort.Slice(diffs, func(i, j int) bool {
		return diffs[i].FilePath < diffs[j].FilePath
	})

	return diffs, nil
}

// FormatSize formats bytes in human-readable format
func FormatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// ChecksumExport represents checksums in JSON format for export
type ChecksumExport struct {
	RunID      int64            `json:"run_id"`
	StepNumber int              `json:"step_number"`
	Checksums  []*FileChecksum  `json:"checksums"`
	ComputedAt time.Time        `json:"computed_at"`
}

// ExportJSON exports checksums to JSON format
func ExportJSON(runID int64, stepNumber int, checksums []*FileChecksum) ([]byte, error) {
	export := &ChecksumExport{
		RunID:      runID,
		StepNumber: stepNumber,
		Checksums:  checksums,
		ComputedAt: time.Now(),
	}

	data, err := json.MarshalIndent(export, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return data, nil
}

// ImportJSON imports checksums from JSON format and stores in database
func ImportJSON(db *database.DB, data []byte) error {
	var export ChecksumExport
	if err := json.Unmarshal(data, &export); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	// Convert to database checksums
	dbChecksums := make([]*database.Checksum, len(export.Checksums))
	for i, cs := range export.Checksums {
		dbChecksums[i] = &database.Checksum{
			RunID:      export.RunID,
			StepNumber: export.StepNumber,
			FilePath:   cs.Path,
			CRC32:      fmt.Sprintf("%08x", cs.CRC32),
			SizeBytes:  cs.SizeBytes,
			ComputedAt: export.ComputedAt,
		}
	}

	// Store in database
	for _, cs := range dbChecksums {
		if err := db.CreateChecksum(cs); err != nil {
			return fmt.Errorf("failed to store checksum for %s: %w", cs.FilePath, err)
		}
	}

	return nil
}
