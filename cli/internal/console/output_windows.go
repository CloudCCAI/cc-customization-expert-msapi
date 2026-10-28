//go:build windows

package console

import (
	"fmt"
	"io"
	"os"
	"syscall"
	"unicode/utf16"
	"unicode/utf8"
	"unsafe"
)

var (
	kernel32           = syscall.NewLazyDLL("kernel32.dll")
	getConsoleModeProc = kernel32.NewProc("GetConsoleMode")
	writeConsoleWProc  = kernel32.NewProc("WriteConsoleW")
)

type windowsConsoleWriter struct {
	handle syscall.Handle
}

func platformWriter(file *os.File) io.Writer {
	handle := syscall.Handle(file.Fd())
	var mode uint32
	ok, _, _ := getConsoleModeProc.Call(
		uintptr(handle),
		uintptr(unsafe.Pointer(&mode)),
	)
	if ok == 0 {
		// Files and pipes keep the CLI's UTF-8 bytes for redirects and pipelines.
		return file
	}
	return &windowsConsoleWriter{handle: handle}
}

func (writer *windowsConsoleWriter) Write(data []byte) (int, error) {
	if len(data) == 0 {
		return 0, nil
	}
	if !utf8.Valid(data) {
		return 0, fmt.Errorf("windows console output is not valid UTF-8")
	}

	const maxChunkBytes = 16 * 1024
	writtenBytes := 0
	for writtenBytes < len(data) {
		end := min(writtenBytes+maxChunkBytes, len(data))
		for end < len(data) && !utf8.RuneStart(data[end]) {
			end--
		}
		if end == writtenBytes {
			end = len(data)
		}

		units := utf16.Encode([]rune(string(data[writtenBytes:end])))
		for offset := 0; offset < len(units); {
			var writtenUnits uint32
			ok, _, callErr := writeConsoleWProc.Call(
				uintptr(writer.handle),
				uintptr(unsafe.Pointer(&units[offset])),
				uintptr(len(units)-offset),
				uintptr(unsafe.Pointer(&writtenUnits)),
				0,
			)
			if ok == 0 {
				if callErr == syscall.Errno(0) {
					callErr = syscall.EINVAL
				}
				return writtenBytes, callErr
			}
			if writtenUnits == 0 {
				return writtenBytes, io.ErrShortWrite
			}
			offset += int(writtenUnits)
		}
		writtenBytes = end
	}
	return writtenBytes, nil
}
