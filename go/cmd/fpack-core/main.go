// Command fpack-core is the native core of fpack. Users normally run it
// through the `fpack` Dart wrapper (dart pub global activate fpack), which
// picks the right binary for the host and forwards arguments, stdio, signals
// and the exit code unchanged.
package main

import (
	"os"

	"github.com/Matkurban/fpack/go/internal/cli"
)

func main() {
	os.Exit(cli.Main(os.Args[1:], nil))
}
