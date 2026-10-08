package targets

import "github.com/Matkurban/fpack/go/internal/runner"

func runnerResult(out string) runner.Result    { return runner.Result{Output: out} }
func runnerResultOut(out string) runner.Result { return runner.Result{Output: out} }
