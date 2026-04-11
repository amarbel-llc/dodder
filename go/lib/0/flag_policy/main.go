package flag_policy

type FlagPolicy int

const (
	FlagPolicyAppend = FlagPolicy(iota)
	FlagPolicyReset
)
