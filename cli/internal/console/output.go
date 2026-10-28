package console

import (
	"io"
	"os"
)

// Stdout returns a writer suitable for the current platform's standard output.
// Redirected output remains byte-for-byte UTF-8; Windows console output is
// adapted by the platform implementation without changing the console code page.
func Stdout() io.Writer {
	return platformWriter(os.Stdout)
}

// Stderr returns a writer suitable for the current platform's standard error.
func Stderr() io.Writer {
	return platformWriter(os.Stderr)
}
