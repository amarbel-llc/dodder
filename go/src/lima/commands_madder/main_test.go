package commands_madder

import (
	"testing"
)

func TestUtilityHasCommands(t *testing.T) {
	if utility.LenCmds() == 0 {
		t.Fatal("expected commands to be registered")
	}
}
