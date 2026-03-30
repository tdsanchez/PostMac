package models

import "time"

// FileInfo represents metadata about a media file
type FileInfo struct {
	Name    string
	Path    string
	RelPath string
	Tags    []string
	Comment string // Finder comment
	Created time.Time
	Size    int64 // File size in bytes
}

// CategoryPreview represents a tag category with preview information
type CategoryPreview struct {
	Tag         string
	Count       int
	PreviewFile FileInfo
}

// TagOperation represents a request to add or remove a tag from a file
type TagOperation struct {
	FilePath string `json:"filePath"`
	Tag      string `json:"tag"`
}

// BatchTagOperation represents a request to add or remove a tag from multiple files
type BatchTagOperation struct {
	FilePaths []string `json:"filePaths"`
	Tag       string   `json:"tag"`
}

// RevealRequest represents a request to reveal a file in Finder
type RevealRequest struct {
	FilePath string `json:"filePath"`
}

// WriteQueueItem represents a pending write operation for tag persistence
type WriteQueueItem struct {
	FilePath  string
	Tags      []string
	Timestamp time.Time
}

// FileMetadata represents EXIF and file system metadata for a file
type FileMetadata struct {
	FileName     string    `json:"fileName"`
	FileSize     int64     `json:"fileSize"`
	Created      time.Time `json:"created"`
	Modified     time.Time `json:"modified"`
	Width        int       `json:"width,omitempty"`
	Height       int       `json:"height,omitempty"`
	Make         string    `json:"make,omitempty"`
	Model        string    `json:"model,omitempty"`
	DateTime     string    `json:"dateTime,omitempty"`
	Orientation  string    `json:"orientation,omitempty"`
	ISO          string    `json:"iso,omitempty"`
	FNumber      string    `json:"fNumber,omitempty"`
	ExposureTime string    `json:"exposureTime,omitempty"`
	FocalLength  string    `json:"focalLength,omitempty"`
	Flash        string    `json:"flash,omitempty"`
	WhiteBalance string    `json:"whiteBalance,omitempty"`
	Artist       string    `json:"artist,omitempty"`
	Copyright    string    `json:"copyright,omitempty"`
	Software     string    `json:"software,omitempty"`
}
