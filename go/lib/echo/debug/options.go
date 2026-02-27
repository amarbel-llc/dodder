package debug

import (
	"strings"

	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/string_builder_joined"
)

type Options struct {
	ExitOnMemoryExhaustion bool
	Trace                  bool
	PProfCPU               bool
	PProfHeap              bool
	GCDisabled             bool
	NoTempDirCleanup       bool
	DryRun                 bool
}

func (options Options) GetCLICompletion() map[string]string {
	return map[string]string{
		"no-tempdir-cleanup":        "",
		"gc_disable":                "",
		"pprof_cpu":                 "",
		"pprof_heap":                "",
		"trace":                     "",
		"dry-run":                   "",
		"exit-on-memory-exhaustion": "",
	}
}

func (options Options) String() string {
	sb := string_builder_joined.Make(",")

	if options.NoTempDirCleanup {
		sb.WriteString("no-tempdir-cleanup")
	}

	if options.GCDisabled {
		sb.WriteString("gc_disabled")
	}

	if options.PProfCPU {
		sb.WriteString("pprof_cpu")
	}

	if options.PProfHeap {
		sb.WriteString("pprof_heap")
	}

	if options.Trace {
		sb.WriteString("trace")
	}

	if options.DryRun {
		sb.WriteString("dry-run")
	}

	if options.ExitOnMemoryExhaustion {
		sb.WriteString("exit-on-memory-exhaustion")
	}

	return sb.String()
}

func (options *Options) Set(v string) (err error) {
	parts := strings.Split(v, ",")

	if len(parts) == 0 {
		parts = []string{"all"}
	}

	for _, p := range parts {
		switch strings.ToLower(p) {
		case "false":

		case "gc_disabled":
			options.GCDisabled = true

		case "pprof_cpu":
			options.PProfCPU = true

		case "pprof_heap":
			options.PProfHeap = true

		case "trace":
			options.Trace = true

		case "no-tempdir-cleanup":
			options.NoTempDirCleanup = true

		case "dry-run":
			options.DryRun = true

		case "exit-on-memory-exhaustion":
			options.ExitOnMemoryExhaustion = true

		case "true":
			fallthrough

		case "all":
			options.GCDisabled = true
			options.PProfCPU = true
			options.PProfHeap = true
			options.Trace = true

		default:
			err = errors.ErrorWithStackf("unsupported debug option: %s", p)
			return err
		}
	}

	return err
}
