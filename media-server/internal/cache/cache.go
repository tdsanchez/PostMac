package cache

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/tdsanchez/PostMac/internal/models"
)

const schema = `
CREATE TABLE IF NOT EXISTS files (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    rel_path TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    size INTEGER NOT NULL,
    created INTEGER NOT NULL,
    comment TEXT
);

CREATE INDEX IF NOT EXISTS idx_files_path ON files(rel_path);

CREATE TABLE IF NOT EXISTS tags (
    file_id INTEGER NOT NULL,
    tag_name TEXT NOT NULL,
    PRIMARY KEY (file_id, tag_name),
    FOREIGN KEY (file_id) REFERENCES files(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_tags_name ON tags(tag_name);
CREATE INDEX IF NOT EXISTS idx_tags_file ON tags(file_id);

CREATE TABLE IF NOT EXISTS scan_metadata (
    directory_path TEXT PRIMARY KEY,
    last_scan_time INTEGER NOT NULL,
    total_files INTEGER NOT NULL,
    total_tags INTEGER NOT NULL
);
`

type Cache struct {
	db  *sql.DB
	dir string
}

// New creates or opens a cache database in the specified directory
func New(serveDir string) (*Cache, error) {
	dbPath := filepath.Join(serveDir, ".media-server-cache.db")

	db, err := sql.Open("sqlite3", dbPath+"?cache=shared&mode=rwc&_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Create schema
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	return &Cache{
		db:  db,
		dir: serveDir,
	}, nil
}

// Close closes the database connection
func (c *Cache) Close() error {
	return c.db.Close()
}

// GetScanMetadata retrieves the last scan metadata
func (c *Cache) GetScanMetadata() (lastScan time.Time, totalFiles int, totalTags int, err error) {
	row := c.db.QueryRow(`
		SELECT last_scan_time, total_files, total_tags
		FROM scan_metadata
		WHERE directory_path = ?
	`, c.dir)

	var unixTime int64
	err = row.Scan(&unixTime, &totalFiles, &totalTags)
	if err == sql.ErrNoRows {
		return time.Time{}, 0, 0, nil // No previous scan
	}
	if err != nil {
		return time.Time{}, 0, 0, err
	}

	lastScan = time.Unix(unixTime, 0)
	return lastScan, totalFiles, totalTags, nil
}

// LoadFiles loads all files and their tags from the cache
func (c *Cache) LoadFiles() ([]models.FileInfo, error) {
	rows, err := c.db.Query(`
		SELECT id, rel_path, name, size, created, comment
		FROM files
		ORDER BY rel_path
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	fileMap := make(map[int64]*models.FileInfo)
	fileOrder := []int64{} // Track insertion order

	for rows.Next() {
		var id int64
		var file models.FileInfo
		var created int64
		var comment sql.NullString

		err := rows.Scan(&id, &file.RelPath, &file.Name, &file.Size, &created, &comment)
		if err != nil {
			return nil, err
		}

		file.Created = time.Unix(created, 0)
		if comment.Valid {
			file.Comment = comment.String
		}
		file.Tags = []string{} // Will be populated below

		fileMap[id] = &file
		fileOrder = append(fileOrder, id)
	}

	// Load tags for all files
	tagRows, err := c.db.Query(`
		SELECT file_id, tag_name
		FROM tags
		ORDER BY file_id, tag_name
	`)
	if err != nil {
		return nil, err
	}
	defer tagRows.Close()

	for tagRows.Next() {
		var fileID int64
		var tagName string

		if err := tagRows.Scan(&fileID, &tagName); err != nil {
			return nil, err
		}

		if file, ok := fileMap[fileID]; ok {
			file.Tags = append(file.Tags, tagName)
		}
	}

	// Build final files slice from fileMap (now with tags loaded)
	files := make([]models.FileInfo, 0, len(fileOrder))
	for _, id := range fileOrder {
		if file, ok := fileMap[id]; ok {
			files = append(files, *file)
		}
	}

	return files, nil
}

// SaveFiles saves all files and their tags to the cache
func (c *Cache) SaveFiles(files []models.FileInfo, totalTags int) error {
	tx, err := c.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Clear existing data
	if _, err := tx.Exec("DELETE FROM tags"); err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM files"); err != nil {
		return err
	}

	// Prepare statements
	fileStmt, err := tx.Prepare(`
		INSERT INTO files (rel_path, name, size, created, comment)
		VALUES (?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer fileStmt.Close()

	tagStmt, err := tx.Prepare(`
		INSERT INTO tags (file_id, tag_name)
		VALUES (?, ?)
	`)
	if err != nil {
		return err
	}
	defer tagStmt.Close()

	// Insert files and tags
	for _, file := range files {
		result, err := fileStmt.Exec(
			file.RelPath,
			file.Name,
			file.Size,
			file.Created.Unix(),
			file.Comment,
		)
		if err != nil {
			return err
		}

		fileID, err := result.LastInsertId()
		if err != nil {
			return err
		}

		// Insert tags for this file
		for _, tag := range file.Tags {
			if _, err := tagStmt.Exec(fileID, tag); err != nil {
				return err
			}
		}
	}

	// Update scan metadata
	_, err = tx.Exec(`
		INSERT OR REPLACE INTO scan_metadata (directory_path, last_scan_time, total_files, total_tags)
		VALUES (?, ?, ?, ?)
	`, c.dir, time.Now().Unix(), len(files), totalTags)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// UpdateFileComment updates a file's comment in the cache
func (c *Cache) UpdateFileComment(relPath, comment string) error {
	_, err := c.db.Exec(`
		UPDATE files SET comment = ? WHERE rel_path = ?
	`, comment, relPath)
	return err
}

// UpdateFileTags updates a file's tags in the cache
func (c *Cache) UpdateFileTags(relPath string, tags []string) error {
	tx, err := c.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Get file ID
	var fileID int64
	err = tx.QueryRow("SELECT id FROM files WHERE rel_path = ?", relPath).Scan(&fileID)
	if err != nil {
		return err
	}

	// Delete existing tags
	if _, err := tx.Exec("DELETE FROM tags WHERE file_id = ?", fileID); err != nil {
		return err
	}

	// Insert new tags
	stmt, err := tx.Prepare("INSERT INTO tags (file_id, tag_name) VALUES (?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, tag := range tags {
		if _, err := stmt.Exec(fileID, tag); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// DeleteFile removes a file from the cache
func (c *Cache) DeleteFile(relPath string) error {
	// Tags are automatically deleted via CASCADE foreign key constraint
	_, err := c.db.Exec("DELETE FROM files WHERE rel_path = ?", relPath)
	return err
}
