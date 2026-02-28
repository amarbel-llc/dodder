package env_ui

import "fmt"

func (env *env) Retry(header, retry string, err error) (tryAgain bool) {
	return env.Confirm(
		fmt.Sprintf("%s:\n%s\n%s", header, retry, err),
		"",
	)
}
