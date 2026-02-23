package xdg

import (
	"fmt"
	"os"
	"path/filepath"

	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/bravo/blob_store_id"
	"code.linenisgreat.com/dodder/go/src/bravo/env_vars"
	"code.linenisgreat.com/dodder/go/src/charlie/files"
	"code.linenisgreat.com/dodder/go/src/charlie/xdg_defaults"
)

type XDG struct {
	Home env_vars.DirectoryLayoutBaseEnvVar
	Cwd  env_vars.DirectoryLayoutBaseEnvVar

	overridePath string

	UtilityName string

	Data    env_vars.DirectoryLayoutBaseEnvVar
	Config  env_vars.DirectoryLayoutBaseEnvVar
	State   env_vars.DirectoryLayoutBaseEnvVar
	Cache   env_vars.DirectoryLayoutBaseEnvVar
	Runtime env_vars.DirectoryLayoutBaseEnvVar
}

func (xdg XDG) GetDirHome() interfaces.DirectoryLayoutBaseEnvVar { return xdg.Home }

func (xdg XDG) GetDirCwd() interfaces.DirectoryLayoutBaseEnvVar { return xdg.Cwd }

func (xdg XDG) GetDirData() interfaces.DirectoryLayoutBaseEnvVar { return xdg.Data }

func (xdg XDG) GetDirConfig() interfaces.DirectoryLayoutBaseEnvVar { return xdg.Config }

func (xdg XDG) GetDirState() interfaces.DirectoryLayoutBaseEnvVar { return xdg.State }

func (xdg XDG) GetDirCache() interfaces.DirectoryLayoutBaseEnvVar { return xdg.Cache }

func (xdg XDG) GetDirRuntime() interfaces.DirectoryLayoutBaseEnvVar { return xdg.Runtime }

func (xdg XDG) GetXDGEnvVars() []env_vars.DirectoryLayoutBaseEnvVar {
	return []env_vars.DirectoryLayoutBaseEnvVar{
		xdg.Data,
		xdg.Config,
		xdg.State,
		xdg.Cache,
		xdg.Runtime,
	}
}

func (xdg XDG) GetXDGPaths() []string {
	return []string{
		xdg.Data.String(),
		xdg.Config.String(),
		xdg.State.String(),
		xdg.Cache.String(),
		xdg.Runtime.String(),
	}
}

func (xdg XDG) AddToEnvVars(envVars interfaces.EnvVars) {
	initElements := xdg.getInitElements()

	for _, element := range initElements {
		envVars[element.defawlt.Name] = element.actual.ActualValue
	}
}

func (xdg XDG) CloneWithUtilityName(
	name string,
) XDG {
	initArgs := InitArgs{
		Home:        xdg.Home.ActualValue,
		Cwd:         xdg.Cwd.ActualValue,
		UtilityName: name,
	}

	if xdg.overridePath != "" {
		errors.PanicIfError(
			xdg.InitializeOverridden(initArgs, xdg.overridePath),
		)
	} else {
		errors.PanicIfError(xdg.InitializeStandardFromEnv(initArgs))
	}

	return xdg
}

func (xdg XDG) CloneWithOverridePath(overridePath string) XDG {
	initArgs := InitArgs{
		Home:        xdg.Home.ActualValue,
		Cwd:         xdg.Cwd.ActualValue,
		UtilityName: xdg.UtilityName,
	}

	errors.PanicIfError(xdg.InitializeOverridden(initArgs, overridePath))

	return xdg
}

func (xdg XDG) CloneWithoutOverride() XDG {
	xdg.overridePath = ""

	initArgs := InitArgs{
		Home:        xdg.Home.ActualValue,
		Cwd:         xdg.Cwd.ActualValue,
		UtilityName: xdg.UtilityName,
	}

	errors.PanicIfError(xdg.InitializeStandardFromEnv(initArgs))

	return xdg
}

func (xdg XDG) GetLocationType() blob_store_id.LocationType {
	if xdg.overridePath == "" {
		return blob_store_id.LocationTypeXDGUser
	} else {
		return blob_store_id.LocationTypeCwd
	}
}

func (xdg *XDG) setInitArgs(initArgs InitArgs) (err error) {
	xdg.Home = xdg_defaults.Home.MakeBaseEnvVar(initArgs.Home)
	xdg.Cwd = xdg_defaults.Cwd.MakeBaseEnvVar(initArgs.Cwd)
	xdg.UtilityName = initArgs.UtilityName

	return err
}

func (initArgs InitArgs) makeCwdXDGOverridePath(base string) string {
	return filepath.Join(
		base,
		fmt.Sprintf(".%s", initArgs.UtilityName),
	)
}

func (initArgs InitArgs) getCwdXDGOverridePath() (string, bool) {
	dir := initArgs.Cwd
	pathCwdXDGOverride := initArgs.makeCwdXDGOverridePath(dir)

	var exists bool

	var count int
	for {
		exists = files.Exists(pathCwdXDGOverride)

		if exists || dir == string(filepath.Separator) {
			break
		}

		count++
		dir = filepath.Dir(dir)
		pathCwdXDGOverride = initArgs.makeCwdXDGOverridePath(dir)

		if count > 100 {
			panic("too deep")
		}
	}

	return dir, exists
}

func (xdg *XDG) InitializeOverriddenIfNecessary(
	initArgs InitArgs,
) (err error) {
	dirCwdXDGOverride, exists := initArgs.getCwdXDGOverridePath()

	if exists {
		if err = xdg.InitializeOverridden(initArgs, dirCwdXDGOverride); err != nil {
			err = errors.Wrap(err)
			return err
		}
	} else {
		if err = xdg.InitializeStandardFromEnv(initArgs); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	return err
}

func (xdg *XDG) InitializeOverridden(
	initArgs InitArgs,
	overridePath string,
) (err error) {
	xdg.overridePath = overridePath

	if err = xdg.setInitArgs(initArgs); err != nil {
		err = errors.Wrap(err)
		return err
	}

	getenv := xdg_defaults.MakeGetenv(
		os.Getenv,
		overridePath,
		xdg.UtilityName,
	)

	for _, initElement := range xdg.getInitElements() {
		initElement.actual.Name = initElement.defawlt.Name
		initElement.actual.DefaultValueTemplate = initElement.defawlt.TemplateOverride

		if err = initElement.actual.InitializeXDGTemplate(getenv); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	return err
}

func (xdg *XDG) InitializeStandardFromEnv(
	initArgs InitArgs,
) (err error) {
	if err = xdg.setInitArgs(initArgs); err != nil {
		err = errors.Wrap(err)
		return err
	}

	getenv := xdg_defaults.MakeGetenv(
		os.Getenv,
		xdg.Cwd.ActualValue,
		xdg.UtilityName,
	)

	for _, initElement := range xdg.getInitElements() {
		initElement.actual.Name = initElement.defawlt.Name
		initElement.actual.DefaultValueTemplate = initElement.defawlt.TemplateDefault

		if err = initElement.actual.InitializeXDGEnvVarOrTemplate(
			xdg.UtilityName,
			getenv,
		); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	return err
}
