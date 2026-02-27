package objects

import (
	"strings"

	"code.linenisgreat.com/dodder/go/internal/_/flag_policy"
	"code.linenisgreat.com/dodder/go/internal/_/interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/bravo/flags"
)

// TODO replace with command_components.ObjectMetadata
func (metadata *metadata) SetFlagDefinitions(
	flagDefs interfaces.CLIFlagDefinitions,
) {
	metadata.SetFlagSetDescription(
		flagDefs,
		"the description to use for created or updated Zettels",
	)

	metadata.SetFlagSetTags(
		flagDefs,
		"the tags to use for created or updated object",
	)

	metadata.SetFlagSetType(
		flagDefs,
		"the type for the created or updated object",
	)
}

func (metadata *metadata) SetFlagSetDescription(
	flagDefs interfaces.CLIFlagDefinitions,
	usage string,
) {
	flagDefs.Var(
		&metadata.Description,
		"description",
		usage,
	)
}

func (metadata *metadata) SetFlagSetTags(
	flagDefs interfaces.CLIFlagDefinitions,
	usage string,
) {
	// TODO add support for tag_paths
	fes := flags.MakeWithPolicy(
		flag_policy.FlagPolicyAppend,
		func() string {
			return metadata.Index.TagPaths.String()
		},
		func(value string) (err error) {
			values := strings.SplitSeq(value, ",")

			for tagString := range values {
				if err = metadata.AddTagString(tagString); err != nil {
					err = errors.Wrap(err)
					return err
				}
			}

			return err
		},
		func() {
			metadata.ResetTags()
		},
	)

	flagDefs.Var(
		fes,
		"tags",
		usage,
	)
}

func (metadata *metadata) SetFlagSetType(
	flagDefs interfaces.CLIFlagDefinitions,
	usage string,
) {
	flagDefs.Func(
		"type",
		usage,
		func(value string) (err error) {
			return metadata.GetTypeMutable().SetType(value)
		},
	)
}
