package command

type (
	Cmd interface {
		Run(Request)
	}

	Description struct {
		Short, Long string
	}

	CommandWithDescription interface {
		GetDescription() Description
	}
)
