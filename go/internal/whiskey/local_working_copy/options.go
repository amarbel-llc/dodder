package local_working_copy

type (
	Options       int
	OptionsGetter interface {
		GetLocalRepoOptions() Options
	}
)

const (
	OptionsEmpty                = Options(iota)
	OptionsAllowConfigReadError = Options(1 << iota)
)

func (o Options) GetAllowConfigReadError() bool {
	return o&OptionsAllowConfigReadError != 0
}
