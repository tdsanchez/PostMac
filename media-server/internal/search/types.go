package search

import (
	"github.com/tdsanchez/PostMac/internal/models"
)

// QueryNode represents a node in the query AST
type QueryNode interface {
	Evaluate(filesByTag map[string][]models.FileInfo) []models.FileInfo
}

// TagNode represents a single tag in the query
type TagNode struct {
	TagName string
}

// Evaluate returns all files with this tag
func (n *TagNode) Evaluate(filesByTag map[string][]models.FileInfo) []models.FileInfo {
	if files, ok := filesByTag[n.TagName]; ok {
		// Return a copy to avoid modifying the original
		result := make([]models.FileInfo, len(files))
		copy(result, files)
		return result
	}
	return []models.FileInfo{}
}

// AndNode represents an AND operation (intersection)
type AndNode struct {
	Left  QueryNode
	Right QueryNode
}

// Evaluate returns files that match both left AND right
func (n *AndNode) Evaluate(filesByTag map[string][]models.FileInfo) []models.FileInfo {
	leftFiles := n.Left.Evaluate(filesByTag)

	// Short-circuit: if left is empty, no need to evaluate right
	if len(leftFiles) == 0 {
		return []models.FileInfo{}
	}

	rightFiles := n.Right.Evaluate(filesByTag)

	// Build a set from right files for O(1) lookup
	rightSet := make(map[string]bool)
	for _, file := range rightFiles {
		rightSet[file.Path] = true
	}

	// Return intersection
	result := []models.FileInfo{}
	for _, file := range leftFiles {
		if rightSet[file.Path] {
			result = append(result, file)
		}
	}

	return result
}

// OrNode represents an OR operation (union)
type OrNode struct {
	Left  QueryNode
	Right QueryNode
}

// Evaluate returns files that match left OR right (union, deduplicated)
func (n *OrNode) Evaluate(filesByTag map[string][]models.FileInfo) []models.FileInfo {
	leftFiles := n.Left.Evaluate(filesByTag)
	rightFiles := n.Right.Evaluate(filesByTag)

	// Build result with deduplication
	seen := make(map[string]bool)
	result := []models.FileInfo{}

	for _, file := range leftFiles {
		if !seen[file.Path] {
			seen[file.Path] = true
			result = append(result, file)
		}
	}

	for _, file := range rightFiles {
		if !seen[file.Path] {
			seen[file.Path] = true
			result = append(result, file)
		}
	}

	return result
}

// NotNode represents a NOT operation (difference)
type NotNode struct {
	Child QueryNode
}

// Evaluate returns all files EXCEPT those matching the child query
func (n *NotNode) Evaluate(filesByTag map[string][]models.FileInfo) []models.FileInfo {
	// Get all files
	allFiles := []models.FileInfo{}
	if files, ok := filesByTag["All"]; ok {
		allFiles = files
	}

	// Get files to exclude
	excludeFiles := n.Child.Evaluate(filesByTag)

	// Build exclusion set
	excludeSet := make(map[string]bool)
	for _, file := range excludeFiles {
		excludeSet[file.Path] = true
	}

	// Return all files NOT in exclusion set
	result := []models.FileInfo{}
	for _, file := range allFiles {
		if !excludeSet[file.Path] {
			result = append(result, file)
		}
	}

	return result
}
