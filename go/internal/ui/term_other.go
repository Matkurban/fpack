//go:build !linux && !darwin && !windows

package ui

import (
	"io"
	"os"
)

func isTerminal(*os.File) bool { return false }
func termWidth(io.Writer) int  { return 100 }
func enableVT(*os.File)        {}
