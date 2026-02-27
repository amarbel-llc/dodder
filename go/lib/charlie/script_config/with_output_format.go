package script_config

import "code.linenisgreat.com/dodder/go/lib/_/equality"

type WithOutputFormat struct {
	ScriptConfig
	UTI           string   `toml:"uti"` // deprecated
	UTIS          []string `toml:"utis"`
	FileExtension string   `toml:"file-extension"`
}

func (a WithOutputFormat) Equals(b WithOutputFormat) bool {
	if !equality.SliceOrdered(a.UTIS, b.UTIS) {
		return false
	}

	if a.FileExtension != b.FileExtension {
		return false
	}

	return a.ScriptConfig.Equals(b.ScriptConfig)
}

func (s *WithOutputFormat) Merge(s2 WithOutputFormat) {
	if s2.UTI != "" {
		s.UTI = s2.UTI
	}

	s.ScriptConfig.Merge(s2.ScriptConfig)
}
