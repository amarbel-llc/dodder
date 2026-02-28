package env_dir

import (
	"fmt"
	"os"
	"path/filepath"

	"code.linenisgreat.com/dodder/go/internal/_/blob_store_id"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/echo/debug"
	"code.linenisgreat.com/dodder/go/lib/echo/xdg"
)

const (
	EnvDir               = "DIR_DODDER" // TODO chang to dodder-prefixed
	EnvBin               = "BIN_DODDER" // TODO change to dodder-prefixed
	XDGUtilityNameDodder = "dodder"
)

type Env interface {
	interfaces.ActiveContextGetter
	interfaces.EnvVarsAdder

	IsDryRun() bool
	GetCwd() string

	GetXDG() xdg.XDG
	GetXDGForBlobStores() xdg.XDG
	GetXDGForBlobStoreId(blob_store_id.Id) xdg.XDG

	GetExecPath() string
	GetTempLocal() TemporaryFS
	MakeDirs(dirs ...string) (err error)
	MakeDirsPerms(perms os.FileMode, dirs ...string) (err error)
	Rel(p string) (out string)
	RelToCwdOrSame(p string) (p1 string)
	MakeCommonEnv() map[string]string
	MakeRelativePathStringFormatWriter() interfaces.StringEncoderTo[string]
	AbsFromCwdOrSame(p string) (p1 string)

	Delete(paths ...string) (err error)
}

type env struct {
	errors.Context
	beforeXDG

	TempLocal, TempOS TemporaryFS

	xdg.XDG
}

var _ Env = &env{}

// sets XDG and creates tmp local
func (env *env) initializeXDG() (err error) {
	env.TempLocal.BasePath = env.Cache.MakePath(
		fmt.Sprintf("tmp-%d", env.GetPid()),
	).String()

	if err = env.MakeDirs(env.GetTempLocal().BasePath); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (env env) GetActiveContext() interfaces.ActiveContext {
	return env.Context
}

func (env env) GetDebug() debug.Options {
	return env.debugOptions
}

func (env env) IsDryRun() bool {
	return env.dryRun
}

func (env env) GetPid() int {
	return env.xdgInitArgs.Pid
}

func (env env) AddToEnvVars(envVars interfaces.EnvVars) {
	envVars[EnvBin] = env.GetExecPath()
}

func (env env) GetExecPath() string {
	return env.xdgInitArgs.ExecPath
}

func (env env) GetCwd() string {
	return env.XDG.Cwd.ActualValue
}

func (env env) GetXDG() xdg.XDG {
	return env.XDG
}

func (env env) GetXDGForBlobStores() xdg.XDG {
	xdg := env.XDG.CloneWithUtilityName("madder")
	return xdg
}

func (env env) GetXDGForBlobStoreId(id blob_store_id.Id) xdg.XDG {
	madderXDG := env.GetXDGForBlobStores()

	switch id.GetLocationType() {
	default:
		return madderXDG

	case blob_store_id.LocationTypeXDGUser:
		return madderXDG.CloneWithoutOverride()

	case blob_store_id.LocationTypeCwd:
		return madderXDG
	}
}

func (env *env) SetXDG(x xdg.XDG) {
	env.XDG = x
}

func (env env) GetTempLocal() TemporaryFS {
	return env.TempLocal
}

func (env env) AbsFromCwdOrSame(p string) (p1 string) {
	var err error
	p1, err = filepath.Abs(p)
	if err != nil {
		p1 = p
	}

	return p1
}

func (env env) RelToCwdOrSame(p string) (p1 string) {
	var err error

	if p1, err = filepath.Rel(env.GetCwd(), p); err != nil {
		p1 = p
	}

	return p1
}

func (env env) Rel(
	p string,
) (out string) {
	out = p

	p1, _ := filepath.Rel(env.GetCwd(), p)

	if p1 != "" {
		out = p1
	}

	return out
}

func (env env) MakeCommonEnv() map[string]string {
	return map[string]string{
		EnvBin: env.GetExecPath(),
		// TODO determine if EnvDir is kept
		// EnvDir: h.Dir(),
	}
}

func (env env) MakeDirs(ds ...string) (err error) {
	return env.MakeDirsPerms(0o755, ds...)
}

func (env env) MakeDirsPerms(perms os.FileMode, ds ...string) (err error) {
	for _, d := range ds {
		if err = os.MkdirAll(d, os.ModeDir|perms); err != nil {
			err = errors.Wrapf(err, "Dir: %q", d)
			return err
		}
	}

	return err
}
