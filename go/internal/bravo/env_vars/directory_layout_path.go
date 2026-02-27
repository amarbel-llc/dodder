package env_vars

import (
	"fmt"
	"path/filepath"

	"code.linenisgreat.com/dodder/go/internal/_/interfaces"
)

type DirectoryLayoutPath struct {
	baseEnvVar interfaces.DirectoryLayoutBaseEnvVar
	target     string
	fullPath   string
}

var _ interfaces.DirectoryLayoutPath = DirectoryLayoutPath{}

func MakeAbsolutePath(path string) DirectoryLayoutPath {
	if !filepath.IsAbs(path) {
		panic(fmt.Sprintf("path is not absolute: %q, path", path))
	}

	return DirectoryLayoutPath{
		target:   path,
		fullPath: path,
	}
}

func (path DirectoryLayoutPath) GetBaseEnvVar() interfaces.DirectoryLayoutBaseEnvVar {
	return path.baseEnvVar
}

func (path DirectoryLayoutPath) GetTarget() string {
	return path.target
}

func (path DirectoryLayoutPath) String() string {
	return path.fullPath
}

func (path DirectoryLayoutPath) GetTemplate() string {
	if path.baseEnvVar == nil {
		return ""
	} else {
		return fmt.Sprintf(
			"$%s/%s",
			filepath.Clean(path.baseEnvVar.GetBaseEnvVarName()),
			filepath.Clean(path.target),
		)
	}
}
