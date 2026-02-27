package exec

import (
	"os/exec"
)

func ExecCommand(c string, args ...[]string) *exec.Cmd {
	actualArgs := make([]string, 0, len(args))

	for _, s := range args {
		actualArgs = append(actualArgs, s...)
	}

	return exec.Command(c, actualArgs...)
}
