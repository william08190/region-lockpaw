//go:build windows

package platform

import (
	"io"
	"log"
	"os"
	"path/filepath"
)

const maxLogSize = 1 << 20

func openAppLogger() (*log.Logger, func()) {
	cacheDirectory, err := os.UserCacheDir()
	if err != nil {
		return log.New(io.Discard, "", 0), func() {}
	}
	logDirectory := filepath.Join(cacheDirectory, "Region Lockpaw")
	if err := os.MkdirAll(logDirectory, 0o700); err != nil {
		return log.New(io.Discard, "", 0), func() {}
	}

	logPath := filepath.Join(logDirectory, "region-lockpaw.log")
	if info, statErr := os.Stat(logPath); statErr == nil && info.Size() > maxLogSize {
		_ = os.Remove(logPath + ".old")
		_ = os.Rename(logPath, logPath+".old")
	}

	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return log.New(io.Discard, "", 0), func() {}
	}
	return log.New(file, "", log.Ldate|log.Ltime|log.Lmicroseconds), func() { _ = file.Close() }
}
