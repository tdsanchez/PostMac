package cache

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/tdsanchez/PostMac/internal/models"
)

const schema = `
CREATE TABLE IF NOT EXISTS files (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    abs_path TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    size_bytes INTEGER NOT NULL,
    mtime_ns INTEGER NOT NULL,
    created INTEGER NOT NULL,
    comment TEXT,
    os_mod_time INTEGER,
    os_birth_time INTEGER,
    exif_create_date INTEGER,
    exif_modify_date INTEGER,
    earliest_date INTEGER,
    needs_date_correction INTEGER,
    large_discrepancy INTEGER
);

CREATE INDEX IF NOT EXISTS idx_files_path ON files(abs_path);

CREATE TABLE IF NOT EXISTS tags (
    file_id INTEGER NOT NULL,
    tag_name TEXT NOT NULL,
    PRIMARY KEY (file_id, tag_name),
    FOREIGN KEY (file_id) REFERENCES files(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_tags_name ON tags(tag_name);
CREATE INDEX IF NOT EXISTS idx_tags_file ON tags(file_id);

CREATE TABLE IF NOT EXISTS scan_metadata (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    last_scan_time INTEGER NOT NULL,
    total_files INTEGER NOT NULL,
    total_tags INTEGER NOT NULL
);
`

const mlSchema = `
CREATE TABLE IF NOT EXISTS date_decisions (
    rel_path TEXT PRIMARY KEY,

    -- Feature vector (input variables)
    os_mod_time INTEGER NOT NULL,
    os_birth_time INTEGER NOT NULL,
    exif_create_time INTEGER,
    exif_modify_time INTEGER,
    earliest_time INTEGER NOT NULL,
    max_diff_hours INTEGER NOT NULL,
    has_exif INTEGER NOT NULL,

    -- Target label (output variable)
    decision TEXT NOT NULL,

    -- Metadata
    decided_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_date_decisions_decision ON date_decisions(decision);
`

type Cache struct {
	db   *sql.DB
	mlDB *sql.DB
}

// New creates or opens a port-isolated cache database in ~/.media-server-conf/
// Each port gets its own cache-PORT.db, preventing data stomping when multiple
// instances share the host.
func New(port string) (*Cache, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	confDir := filepath.Join(homeDir, ".media-server-conf")
	if err := os.MkdirAll(confDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create conf directory: %w", err)
	}

	dbName := "cache.db"
	if port != "" {
		dbName = "cache-" + port + ".db"
	}
	dbPath := filepath.Join(confDir, dbName)
	db, err := sql.Open("sqlite3", dbPath+"?cache=shared&mode=rwc&_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Create schema
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	mlDBName := "ml-training.db"
	if port != "" {
		mlDBName = "ml-training-" + port + ".db"
	}
	mlDBPath := filepath.Join(confDir, mlDBName)
	mlDB, err := sql.Open("sqlite3", mlDBPath+"?cache=shared&mode=rwc&_journal_mode=WAL")
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to open ML database: %w", err)
	}

	// Create ML schema
	if _, err := mlDB.Exec(mlSchema); err != nil {
		db.Close()
		mlDB.Close()
		return nil, fmt.Errorf("failed to create ML schema: %w", err)
	}

	return &Cache{
		db:   db,
		mlDB: mlDB,
	}, nil
}

// Close closes the database connections
func (c *Cache) Close() error {
	err1 := c.db.Close()
	err2 := c.mlDB.Close()
	if err1 != nil {
		return err1
	}
	return err2
}

// GetScanMetadata retrieves the last scan metadata
func (c *Cache) GetScanMetadata() (lastScan time.Time, totalFiles int, totalTags int, err error) {
	row := c.db.QueryRow(`
		SELECT last_scan_time, total_files, total_tags
		FROM scan_metadata
		WHERE id = 1
	`)

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
		SELECT id, abs_path, name, size_bytes, mtime_ns, created, comment,
		       os_mod_time, os_birth_time, exif_create_date, exif_modify_date,
		       earliest_date, needs_date_correction, large_discrepancy
		FROM files
		ORDER BY abs_path
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
		var mtimeNs int64
		var created int64
		var comment sql.NullString
		var osModTime, osBirthTime, exifCreateDate, exifModifyDate, earliestDate sql.NullInt64
		var needsDateCorrection, largeDiscrepancy sql.NullInt64

		err := rows.Scan(&id, &file.Path, &file.Name, &file.Size, &mtimeNs, &created, &comment,
			&osModTime, &osBirthTime, &exifCreateDate, &exifModifyDate,
			&earliestDate, &needsDateCorrection, &largeDiscrepancy)
		if err != nil {
			return nil, err
		}

		file.Created = time.Unix(created, 0)
		if comment.Valid {
			file.Comment = comment.String
		}

		// Load date analysis fields
		if osModTime.Valid {
			file.OSModTime = time.Unix(osModTime.Int64, 0)
		}
		if osBirthTime.Valid {
			file.OSBirthTime = time.Unix(osBirthTime.Int64, 0)
		}
		if exifCreateDate.Valid {
			file.EXIFCreateDate = time.Unix(exifCreateDate.Int64, 0)
		}
		if exifModifyDate.Valid {
			file.EXIFModifyDate = time.Unix(exifModifyDate.Int64, 0)
		}
		if earliestDate.Valid {
			file.EarliestDate = time.Unix(earliestDate.Int64, 0)
		}
		if needsDateCorrection.Valid {
			file.NeedsDateCorrection = needsDateCorrection.Int64 == 1
		}
		if largeDiscrepancy.Valid {
			file.LargeDiscrepancy = largeDiscrepancy.Int64 == 1
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
		INSERT INTO files (abs_path, name, size_bytes, mtime_ns, created, comment,
		                   os_mod_time, os_birth_time, exif_create_date, exif_modify_date,
		                   earliest_date, needs_date_correction, large_discrepancy)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
		// Convert date analysis fields to nullable integers
		var osModTime, osBirthTime, exifCreateDate, exifModifyDate, earliestDate sql.NullInt64
		var needsDateCorrection int64

		if !file.OSModTime.IsZero() {
			osModTime = sql.NullInt64{Int64: file.OSModTime.Unix(), Valid: true}
		}
		if !file.OSBirthTime.IsZero() {
			osBirthTime = sql.NullInt64{Int64: file.OSBirthTime.Unix(), Valid: true}
		}
		if !file.EXIFCreateDate.IsZero() {
			exifCreateDate = sql.NullInt64{Int64: file.EXIFCreateDate.Unix(), Valid: true}
		}
		if !file.EXIFModifyDate.IsZero() {
			exifModifyDate = sql.NullInt64{Int64: file.EXIFModifyDate.Unix(), Valid: true}
		}
		if !file.EarliestDate.IsZero() {
			earliestDate = sql.NullInt64{Int64: file.EarliestDate.Unix(), Valid: true}
		}
		if file.NeedsDateCorrection {
			needsDateCorrection = 1
		}

		var largeDiscrepancy int64
		if file.LargeDiscrepancy {
			largeDiscrepancy = 1
		}

		// Get mtime in nanoseconds for freshness check
		mtimeNs := file.OSModTime.UnixNano()

		result, err := fileStmt.Exec(
			file.Path,
			file.Name,
			file.Size,
			mtimeNs,
			file.Created.Unix(),
			file.Comment,
			osModTime,
			osBirthTime,
			exifCreateDate,
			exifModifyDate,
			earliestDate,
			needsDateCorrection,
			largeDiscrepancy,
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
		INSERT OR REPLACE INTO scan_metadata (id, last_scan_time, total_files, total_tags)
		VALUES (1, ?, ?, ?)
	`, time.Now().Unix(), len(files), totalTags)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// UpdateFileComment updates a file's comment in the cache
func (c *Cache) UpdateFileComment(absPath, comment string) error {
	_, err := c.db.Exec(`
		UPDATE files SET comment = ? WHERE abs_path = ?
	`, comment, absPath)
	return err
}

// UpdateFileTags updates a file's tags in the cache
func (c *Cache) UpdateFileTags(absPath string, tags []string) error {
	tx, err := c.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Get file ID
	var fileID int64
	err = tx.QueryRow("SELECT id FROM files WHERE abs_path = ?", absPath).Scan(&fileID)
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
func (c *Cache) DeleteFile(absPath string) error {
	// Tags are automatically deleted via CASCADE foreign key constraint
	_, err := c.db.Exec("DELETE FROM files WHERE abs_path = ?", absPath)
	return err
}

// UpsertFile inserts or updates a file in the cache
func (c *Cache) UpsertFile(f models.FileInfo) error {
	tx, err := c.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Convert date analysis fields to nullable integers
	var osModTime, osBirthTime, exifCreateDate, exifModifyDate, earliestDate sql.NullInt64
	var needsDateCorrection int64

	if !f.OSModTime.IsZero() {
		osModTime = sql.NullInt64{Int64: f.OSModTime.Unix(), Valid: true}
	}
	if !f.OSBirthTime.IsZero() {
		osBirthTime = sql.NullInt64{Int64: f.OSBirthTime.Unix(), Valid: true}
	}
	if !f.EXIFCreateDate.IsZero() {
		exifCreateDate = sql.NullInt64{Int64: f.EXIFCreateDate.Unix(), Valid: true}
	}
	if !f.EXIFModifyDate.IsZero() {
		exifModifyDate = sql.NullInt64{Int64: f.EXIFModifyDate.Unix(), Valid: true}
	}
	if !f.EarliestDate.IsZero() {
		earliestDate = sql.NullInt64{Int64: f.EarliestDate.Unix(), Valid: true}
	}
	if f.NeedsDateCorrection {
		needsDateCorrection = 1
	}

	var largeDiscrepancy int64
	if f.LargeDiscrepancy {
		largeDiscrepancy = 1
	}

	mtimeNs := f.OSModTime.UnixNano()

	// Insert or replace file
	result, err := tx.Exec(`
		INSERT OR REPLACE INTO files (abs_path, name, size_bytes, mtime_ns, created, comment,
		                              os_mod_time, os_birth_time, exif_create_date, exif_modify_date,
		                              earliest_date, needs_date_correction, large_discrepancy)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, f.Path, f.Name, f.Size, mtimeNs, f.Created.Unix(), f.Comment,
		osModTime, osBirthTime, exifCreateDate, exifModifyDate,
		earliestDate, needsDateCorrection, largeDiscrepancy)
	if err != nil {
		return err
	}

	fileID, err := result.LastInsertId()
	if err != nil {
		return err
	}

	// Delete existing tags and insert new ones
	if _, err := tx.Exec("DELETE FROM tags WHERE file_id = ?", fileID); err != nil {
		return err
	}

	for _, tag := range f.Tags {
		if _, err := tx.Exec("INSERT INTO tags (file_id, tag_name) VALUES (?, ?)", fileID, tag); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetFile retrieves a file by its absolute path
func (c *Cache) GetFile(absPath string) *models.FileInfo {
	row := c.db.QueryRow(`
		SELECT id, abs_path, name, size_bytes, mtime_ns, created, comment,
		       os_mod_time, os_birth_time, exif_create_date, exif_modify_date,
		       earliest_date, needs_date_correction, large_discrepancy
		FROM files
		WHERE abs_path = ?
	`, absPath)

	var id int64
	var file models.FileInfo
	var mtimeNs int64
	var created int64
	var comment sql.NullString
	var osModTime, osBirthTime, exifCreateDate, exifModifyDate, earliestDate sql.NullInt64
	var needsDateCorrection, largeDiscrepancy sql.NullInt64

	err := row.Scan(&id, &file.Path, &file.Name, &file.Size, &mtimeNs, &created, &comment,
		&osModTime, &osBirthTime, &exifCreateDate, &exifModifyDate,
		&earliestDate, &needsDateCorrection, &largeDiscrepancy)
	if err != nil {
		return nil
	}

	file.Created = time.Unix(created, 0)
	if comment.Valid {
		file.Comment = comment.String
	}

	if osModTime.Valid {
		file.OSModTime = time.Unix(osModTime.Int64, 0)
	}
	if osBirthTime.Valid {
		file.OSBirthTime = time.Unix(osBirthTime.Int64, 0)
	}
	if exifCreateDate.Valid {
		file.EXIFCreateDate = time.Unix(exifCreateDate.Int64, 0)
	}
	if exifModifyDate.Valid {
		file.EXIFModifyDate = time.Unix(exifModifyDate.Int64, 0)
	}
	if earliestDate.Valid {
		file.EarliestDate = time.Unix(earliestDate.Int64, 0)
	}
	if needsDateCorrection.Valid {
		file.NeedsDateCorrection = needsDateCorrection.Int64 == 1
	}
	if largeDiscrepancy.Valid {
		file.LargeDiscrepancy = largeDiscrepancy.Int64 == 1
	}

	// Load tags
	tagRows, err := c.db.Query("SELECT tag_name FROM tags WHERE file_id = ?", id)
	if err != nil {
		return &file
	}
	defer tagRows.Close()

	for tagRows.Next() {
		var tag string
		if err := tagRows.Scan(&tag); err == nil {
			file.Tags = append(file.Tags, tag)
		}
	}

	return &file
}

// GetFileMtime returns the stored mtime_ns for freshness checking
func (c *Cache) GetFileMtime(absPath string) (int64, bool) {
	var mtimeNs int64
	err := c.db.QueryRow("SELECT mtime_ns FROM files WHERE abs_path = ?", absPath).Scan(&mtimeNs)
	if err != nil {
		return 0, false
	}
	return mtimeNs, true
}

// SaveDateDecision saves a user's decision for date correction with complete feature vector
func (c *Cache) SaveDateDecision(relPath, decision string, osModTime, osBirthTime, exifCreateTime, exifModifyTime, earliestTime int64, maxDiffHours int, hasExif bool) error {
	hasExifInt := 0
	if hasExif {
		hasExifInt = 1
	}

	_, err := c.mlDB.Exec(`
		INSERT OR REPLACE INTO date_decisions (
			rel_path, os_mod_time, os_birth_time, exif_create_time, exif_modify_time,
			earliest_time, max_diff_hours, has_exif, decision, decided_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, relPath, osModTime, osBirthTime, exifCreateTime, exifModifyTime, earliestTime, maxDiffHours, hasExifInt, decision, time.Now().Unix())
	return err
}

// GetDateDecision retrieves a user's decision for a file (if any)
func (c *Cache) GetDateDecision(relPath string) (decision string, exists bool, err error) {
	err = c.mlDB.QueryRow(`
		SELECT decision FROM date_decisions WHERE rel_path = ?
	`, relPath).Scan(&decision)

	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return decision, true, nil
}

// DateDecisionStats represents training progress statistics
type DateDecisionStats struct {
	TotalDecisions     int
	UseOSMod           int // Use OS Modification Time
	UseOSBirth         int // Use OS Birth Time
	UseEXIFCreate      int // Use EXIF CreateDate
	UseEXIFModify      int // Use EXIF ModifyDate
	Skip               int // No change
	NotChosen          int
	UseOSModPct        float64
	UseOSBirthPct      float64
	UseEXIFCreatePct   float64
	UseEXIFModifyPct   float64
	SkipPct            float64
}

// GetDateDecisionStats retrieves statistics about date correction decisions
func (c *Cache) GetDateDecisionStats() (*DateDecisionStats, error) {
	stats := &DateDecisionStats{}

	rows, err := c.mlDB.Query(`
		SELECT decision, COUNT(*) as count
		FROM date_decisions
		WHERE decision != 'not_chosen'
		GROUP BY decision
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var decision string
		var count int
		if err := rows.Scan(&decision, &count); err != nil {
			return nil, err
		}

		stats.TotalDecisions += count
		switch decision {
		case "use_os_mod":
			stats.UseOSMod = count
		case "use_os_birth":
			stats.UseOSBirth = count
		case "use_exif_create":
			stats.UseEXIFCreate = count
		case "use_exif_modify":
			stats.UseEXIFModify = count
		case "skip":
			stats.Skip = count
		}
	}

	// Calculate percentages
	if stats.TotalDecisions > 0 {
		total := float64(stats.TotalDecisions)
		stats.UseOSModPct = float64(stats.UseOSMod) / total * 100
		stats.UseOSBirthPct = float64(stats.UseOSBirth) / total * 100
		stats.UseEXIFCreatePct = float64(stats.UseEXIFCreate) / total * 100
		stats.UseEXIFModifyPct = float64(stats.UseEXIFModify) / total * 100
		stats.SkipPct = float64(stats.Skip) / total * 100
	}

	return stats, nil
}

// DatePrediction represents a model prediction for date correction
type DatePrediction struct {
	SuggestedDecision string  // "earliest", "exif", "os", "skip"
	Confidence        float64 // 0.0 to 1.0
	MatchCount        int     // Number of similar training examples
	IsReady           bool    // Whether model has enough training data
}

// PredictDateDecision uses training data to predict the best decision for a file
// Simple pattern-matching approach: find similar examples and return most common decision
func (c *Cache) PredictDateDecision(osModTime, osBirthTime, exifCreateTime, exifModifyTime, earliestTime int64, maxDiffHours int, hasExif bool) (*DatePrediction, error) {
	prediction := &DatePrediction{
		IsReady: false,
	}

	// Check if we have enough training data (minimum 5 examples)
	stats, err := c.GetDateDecisionStats()
	if err != nil {
		return nil, err
	}
	if stats.TotalDecisions < 5 {
		return prediction, nil // Not ready yet
	}
	prediction.IsReady = true

	// Convert hasExif to int for comparison
	hasExifInt := 0
	if hasExif {
		hasExifInt = 1
	}

	// Query all training examples
	rows, err := c.mlDB.Query(`
		SELECT decision, max_diff_hours, has_exif
		FROM date_decisions
		WHERE decision != 'not_chosen'
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Pattern matching: group by decision and count similar examples
	decisionCounts := make(map[string]int)
	totalMatches := 0

	for rows.Next() {
		var decision string
		var trainMaxDiff int
		var trainHasExif int

		if err := rows.Scan(&decision, &trainMaxDiff, &trainHasExif); err != nil {
			continue
		}

		// Simple similarity: match on has_exif and similar max_diff_hours
		// Bucket max_diff_hours: 0, 1-24, 25-168 (week), 169+ (more than week)
		bucket := func(hours int) int {
			if hours == 0 {
				return 0
			} else if hours <= 24 {
				return 1
			} else if hours <= 168 {
				return 2
			}
			return 3
		}

		// Match if same EXIF status and similar time difference bucket
		if trainHasExif == hasExifInt && bucket(trainMaxDiff) == bucket(maxDiffHours) {
			decisionCounts[decision]++
			totalMatches++
		}
	}

	// No similar examples found - return most common overall decision
	if totalMatches == 0 {
		// Fallback: return most common decision from all training data
		mostCommon := "use_os_mod"
		maxCount := stats.UseOSMod
		if stats.UseOSBirth > maxCount {
			mostCommon = "use_os_birth"
			maxCount = stats.UseOSBirth
		}
		if stats.UseEXIFCreate > maxCount {
			mostCommon = "use_exif_create"
			maxCount = stats.UseEXIFCreate
		}
		if stats.UseEXIFModify > maxCount {
			mostCommon = "use_exif_modify"
			maxCount = stats.UseEXIFModify
		}
		if stats.Skip > maxCount {
			mostCommon = "skip"
			maxCount = stats.Skip
		}

		prediction.SuggestedDecision = mostCommon
		prediction.Confidence = float64(maxCount) / float64(stats.TotalDecisions)
		prediction.MatchCount = 0
		return prediction, nil
	}

	// Find the most common decision among similar examples
	maxCount := 0
	for decision, count := range decisionCounts {
		if count > maxCount {
			maxCount = count
			prediction.SuggestedDecision = decision
		}
	}

	// Calculate confidence as percentage of similar examples with same decision
	prediction.Confidence = float64(maxCount) / float64(totalMatches)
	prediction.MatchCount = totalMatches

	return prediction, nil
}
