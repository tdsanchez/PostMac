//go:build darwin

package scanner

import (
	"os"
	"syscall"
	"time"
)

// getBirthTime returns the file creation (birth) time on macOS via Birthtimespec.
func getBirthTime(info os.FileInfo) time.Time {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return info.ModTime()
	}
	return time.Unix(stat.Birthtimespec.Sec, stat.Birthtimespec.Nsec)
}
