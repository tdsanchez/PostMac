//go:build !darwin

package scanner

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/pkg/xattr"
)

// SetMacOSTags writes Finder-compatible tags to a file.
// Tries direct xattr write first (works for local files).
// If the filesystem doesn't support xattr (e.g. SSHFS), falls back to
// writing via SSH using the `tag` CLI on the remote host.
func SetMacOSTags(path string, tags []string) error {
	data, err := tagsToXattr(tags)
	if err != nil {
		return err
	}

	err = xattr.Set(path, "com.apple.metadata:_kMDItemUserTags", data)
	if err == nil {
		return nil
	}

	// Check for EOPNOTSUPP — filesystem doesn't support xattr (e.g. SSHFS)
	if isXattrUnsupported(err) {
		return sshTagWrite(path, tags)
	}

	return err
}

// isXattrUnsupported returns true if the error indicates xattr is not supported.
func isXattrUnsupported(err error) bool {
	if err == nil {
		return false
	}
	// Check for EOPNOTSUPP or ENOTSUP
	if errno, ok := err.(syscall.Errno); ok {
		return errno == syscall.EOPNOTSUPP || errno == 0x5f // ENOTSUP
	}
	s := err.Error()
	return strings.Contains(s, "operation not supported") ||
		strings.Contains(s, "not supported")
}

// sshTagWrite writes tags to a remote file via SSH using the `tag` CLI tool.
// It reads /proc/mounts to find the SSHFS mount for the given path, then
// SSHes to the remote host and runs ~/bin/tag --set.
func sshTagWrite(localPath string, tags []string) error {
	host, remotePath, err := resolveSSHFSPath(localPath)
	if err != nil {
		return fmt.Errorf("ssh tag fallback: %w", err)
	}

	tagArg := strings.Join(tags, ",")
	if tagArg == "" {
		// No tags — clear them
		tagArg = ""
	}

	var cmd *exec.Cmd
	if tagArg == "" {
		cmd = exec.Command("ssh", host, fmt.Sprintf("~/bin/tag --remove '*' %q", remotePath))
	} else {
		cmd = exec.Command("ssh", host,
			fmt.Sprintf("~/bin/tag --set %q %q", tagArg, remotePath))
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ssh tag write failed on %s: %v — %s", host, err, string(out))
	}

	return nil
}

// resolveSSHFSPath finds the SSHFS mount that contains localPath and returns
// the SSH host (user@host) and the corresponding remote absolute path.
func resolveSSHFSPath(localPath string) (host string, remotePath string, err error) {
	f, err := os.Open("/proc/mounts")
	if err != nil {
		return "", "", fmt.Errorf("cannot read /proc/mounts: %w", err)
	}
	defer f.Close()

	// Find the longest matching SSHFS mountpoint for localPath
	bestMount := ""
	bestSource := ""

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		fsType := fields[2]
		if fsType != "fuse.sshfs" {
			continue
		}

		source := fields[0] // e.g. user@host:/remote/path
		mountPoint := fields[1] // e.g. /home/user/mnt/mountpoint

		if strings.HasPrefix(localPath, mountPoint+"/") || localPath == mountPoint {
			if len(mountPoint) > len(bestMount) {
				bestMount = mountPoint
				bestSource = source
			}
		}
	}

	if bestMount == "" {
		return "", "", fmt.Errorf("no SSHFS mount found for path %s", localPath)
	}

	// Parse source: strip sshfs# prefix if present
	bestSource = strings.TrimPrefix(bestSource, "sshfs#")

	// Split user@host:remotedir
	colonIdx := strings.LastIndex(bestSource, ":")
	if colonIdx < 0 {
		return "", "", fmt.Errorf("unexpected SSHFS source format: %s", bestSource)
	}

	host = bestSource[:colonIdx]          // e.g. user@host
	remoteBase := bestSource[colonIdx+1:] // e.g. /remote/base/path

	// Map local path to remote path
	relPath := strings.TrimPrefix(localPath, bestMount)
	remotePath = remoteBase + relPath

	return host, remotePath, nil
}

// tagsToXattr encodes a list of tag strings as a binary plist for the
// com.apple.metadata:_kMDItemUserTags xattr. Used for local xattr writes.
func tagsToXattr(tags []string) ([]byte, error) {
	// Reuse the existing SetMacOSTags plist logic via the xattr package path.
	// We call the shared plist marshaling from tags.go (SetMacOSTags there does it).
	// Here we just need the encoded bytes — delegate to the shared encoder.
	return encodePlistTags(tags)
}

// SetMacOSComment stores a comment in a user-namespace xattr on Linux.
// Finder comments don't exist on Linux; we use user.comment as a fallback.
func SetMacOSComment(path string, comment string) error {
	if comment == "" {
		return xattr.Remove(path, "user.comment")
	}
	return xattr.Set(path, "user.comment", []byte(comment))
}

// GetMacOSComment reads a comment from user-namespace xattr on Linux.
func GetMacOSComment(path string) string {
	data, err := xattr.Get(path, "user.comment")
	if err != nil {
		return ""
	}
	return string(data)
}
