//go:build !darwin

package scanner

import (
	"os"
	"time"
)

// getBirthTime returns the best available approximation of file creation time on Linux.
// Linux does not expose birth time via syscall.Stat_t in a portable way;
// ModTime is used as a fallback.
func getBirthTime(info os.FileInfo) time.Time {
	return info.ModTime()
}
