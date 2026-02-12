//go:build unix && !windows

package platform

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

const instanceLockFilename = "instance.lock"

type unixInstanceLock struct {
	file *os.File
}

func acquireInstanceLock(appID string) (InstanceLock, error) {
	lockPath, err := unixInstanceLockPath(appID)
	if err != nil {
		return nil, err
	}

	// #nosec G304 -- lockPath is built from process-owned runtime/temp directories.
	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open instance lock file: %w", err)
	}

	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = file.Close()
		if isUnixLockContention(err) {
			return nil, ErrInstanceAlreadyRunning
		}

		return nil, fmt.Errorf("acquire instance file lock: %w", err)
	}

	return &unixInstanceLock{file: file}, nil
}

func (l *unixInstanceLock) Release() error {
	if l == nil || l.file == nil {
		return nil
	}

	fd := int(l.file.Fd())
	unlockErr := syscall.Flock(fd, syscall.LOCK_UN)
	closeErr := l.file.Close()
	l.file = nil

	if unlockErr != nil && !errors.Is(unlockErr, syscall.EBADF) {
		return fmt.Errorf("unlock instance file lock: %w", unlockErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close instance lock file: %w", closeErr)
	}

	return nil
}

func unixInstanceLockPath(appID string) (string, error) {
	lockDir := strings.TrimSpace(os.Getenv("XDG_RUNTIME_DIR"))
	if lockDir != "" {
		lockDir = filepath.Join(lockDir, normalizeInstanceLockComponent(appID, "app"))
	} else {
		lockDir = filepath.Join(
			os.TempDir(),
			normalizeInstanceLockComponent(appID, "app")+"-"+strconv.Itoa(os.Getuid()),
		)
	}

	if err := os.MkdirAll(lockDir, 0o700); err != nil {
		return "", fmt.Errorf("create instance lock dir: %w", err)
	}

	return filepath.Join(lockDir, instanceLockFilename), nil
}

func isUnixLockContention(err error) bool {
	return errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN)
}
