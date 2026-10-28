//go:build !windows

package console

import (
	"io"
	"os"
)

func platformWriter(file *os.File) io.Writer {
	return file
}
