package env_ui

import (
	"github.com/charmbracelet/huh"
)

func (env *env) Confirm(title, description string) (success bool) {
	if !env.GetIn().IsTty() {
		env.GetErr().Print(
			"stdin is not a tty, unable to get permission to continue",
		)

		return success
	}

	huh.NewConfirm().
		Title(title).
		Description(description).
		Affirmative("Yes").
		Negative("No").
		Value(&success).
		Run()

	return success
}
