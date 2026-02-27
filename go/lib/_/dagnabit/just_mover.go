package dagnabit

import (
	"fmt"
	"os"
	"os/exec"
)

// TODO: internalize the move logic (git mv + import rewriting + gofmt)
// instead of delegating to the justfile recipe.

// JustMover moves packages by shelling out to `just codemod-go-move_package`.
// Dir is the working directory containing the justfile.
type JustMover struct {
	Dir string
}

func (m JustMover) MovePackage(src, dst string) error {
	cmd := exec.Command("just", "codemod-go-move_package", src, dst)
	cmd.Dir = m.Dir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("just codemod-go-move_package %s %s: %w", src, dst, err)
	}

	return nil
}
