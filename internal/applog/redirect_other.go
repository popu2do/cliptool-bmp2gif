//go:build !windows

package applog

import "os"

func redirectStandardStreams(target *os.File) error {
	os.Stdout = target
	os.Stderr = target
	return nil
}
