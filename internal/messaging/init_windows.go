//go:build windows

package messaging

import (
	"os"

	"golang.org/x/sys/windows"
)

func init() {
	// Switch stdin and stdout to binary mode on Windows.
	// Chrome/Edge spawn the host with pipe handles (not console handles).
	// O_TEXT mode (the Windows default) converts 0x0A bytes to 0x0D 0x0A,
	// which corrupts the 4-byte little-endian length prefix whenever a byte
	// happens to be 0x0A (newline).
	//
	// SetConsoleMode applies to console (TTY) handles only. For pipe handles
	// (which Chrome/Edge use), we call the CRT _setmode function via msvcrt.dll.
	// This is the documented Windows approach for setting O_BINARY mode on
	// file descriptors obtained from the C runtime.
	msvcrt := windows.NewLazySystemDLL("msvcrt.dll")
	setmode := msvcrt.NewProc("_setmode")
	// _setmode(fileno(stdin), O_BINARY=0x8000)
	setmode.Call(uintptr(os.Stdin.Fd()), 0x8000)
	// _setmode(fileno(stdout), O_BINARY=0x8000)
	setmode.Call(uintptr(os.Stdout.Fd()), 0x8000)
}
