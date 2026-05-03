package exec

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
)

func ExecCommand(c string, args ...[]string) *exec.Cmd {
	actualArgs := make([]string, 0, len(args))

	for _, s := range args {
		actualArgs = append(actualArgs, s...)
	}

	return exec.Command(c, actualArgs...)
}

// MergeOSEnvWithAdder builds an env slice suitable for *exec.Cmd.Env by
// starting from os.Environ() and appending entries contributed by adder. A nil
// adder yields os.Environ() unchanged.
func MergeOSEnvWithAdder(adder interfaces.EnvVarsAdder) []string {
	out := os.Environ()

	if adder == nil {
		return out
	}

	envVars := make(interfaces.EnvVars)
	adder.AddToEnvVars(envVars)

	for k, v := range envVars {
		out = append(out, fmt.Sprintf("%s=%s", k, v))
	}

	return out
}
