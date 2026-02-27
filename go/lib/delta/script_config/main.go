// TODO rename this to something clearer
package script_config

import (
	"fmt"
	"os/exec"

	"code.linenisgreat.com/dodder/go/lib/_/equality"
	"code.linenisgreat.com/dodder/go/lib/_/reset"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/ui"
)

type ScriptConfig struct {
	Description string            `toml:"description"`
	Shell       []string          `toml:"shell,omitempty"`
	Script      string            `toml:"script,omitempty,multiline"`
	Env         map[string]string `toml:"env,omitempty"`
}

func (a *ScriptConfig) Environ() map[string]string {
	ui.TodoP4("copy")
	return a.Env
}

func (a *ScriptConfig) Reset() {
	a.Description = ""
	a.Script = ""

	a.Shell = reset.Slice(a.Shell)
	a.Env = reset.Map(a.Env)
}

func (a ScriptConfig) Equals(b ScriptConfig) bool {
	if len(a.Shell) != len(b.Shell) {
		return false
	}

	if a.Description != b.Description {
		return false
	}

	for k, v := range a.Shell {
		v1 := b.Shell[k]

		if v != v1 {
			return false
		}
	}

	if a.Script != b.Script {
		return false
	}

	if !equality.MapsOrdered(a.Env, b.Env) {
		return false
	}

	return true
}

func (s *ScriptConfig) Merge(s2 ScriptConfig) {
	if s2.Description != "" {
		s.Description = s2.Description
	}

	if len(s2.Shell) > 0 {
		s.Shell = s2.Shell
	}

	if s2.Script != "" {
		s.Script = s2.Script
	}

	if len(s.Env) == 0 {
		s.Env = make(map[string]string)
	}

	for k, v := range s2.Env {
		s.Env[k] = v
	}
}

func (s ScriptConfig) Cmd(args ...string) (c *exec.Cmd, err error) {
	if len(args) > 0 {
		args = append([]string{"--"}, args...)
	}

	switch {
	case s.Script == "" && len(s.Shell) == 0:
		err = errors.ErrorWithStackf("no script or shell set")
		return c, err

	case s.Script != "" && len(s.Shell) > 0:
		all := append(s.Shell, s.Script)
		all = append(all, args...)
		c = exec.Command(s.Shell[0], all[1:]...)

	case s.Script != "":
		all := []string{
			"--noprofile",
			"--norc",
			"-c",
		}

		all = append(all, s.Script)
		all = append(all, args...)
		c = exec.Command("bash", all...)

	case len(s.Shell) > 0:
		all := s.Shell
		all = append(all, args...)

		if len(all) > 1 {
			c = exec.Command(all[0], all[1:]...)
		} else {
			c = exec.Command(all[0])
		}
	}

	envCollapsed := make([]string, 0, len(s.Env))

	for k, v := range s.Env {
		envCollapsed = append(envCollapsed, fmt.Sprintf("%s=%s", k, v))
	}

	c.Env = envCollapsed

	return c, err
}
