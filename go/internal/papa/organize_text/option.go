package organize_text

import (
	"fmt"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	"code.linenisgreat.com/dodder/go/lib/charlie/comments"
	"code.linenisgreat.com/dodder/go/lib/charlie/values"
)

type OptionComment interface {
	CloneOptionComment() OptionComment
	interfaces.StringerSetter
}

type OptionCommentWithApply interface {
	OptionComment
	ApplyToText(Options, *Assignment) error
	ApplyToReader(Options, *reader) error
	ApplyToWriter(Options, *writer) error
}

// TODO add config to automatically add dry run if necessary
func MakeOptionCommentSet(
	elements map[string]OptionComment,
	options ...OptionComment,
) OptionCommentSet {
	ocs := OptionCommentSet{
		prototype:      make(PrototypeOptionComments),
		OptionComments: options,
	}

	if elements != nil {
		for k, el := range elements {
			ocs.AddPrototype(k, el)
		}
	}

	ocs.AddPrototype("hide", optionCommentHide(""))
	ocs.AddPrototype("", optionCommentHide(""))

	return ocs
}

type PrototypeOptionComments map[string]OptionComment

type OptionCommentSet struct {
	prototype      PrototypeOptionComments
	OptionComments []OptionComment
}

func (ocs *OptionCommentSet) GetPrototypeOptionComments() PrototypeOptionComments {
	return ocs.prototype
}

func (ocs *OptionCommentSet) AddPrototype(
	key string,
	o OptionComment,
) OptionComment {
	o = OptionCommentWithKey{
		Key:           key,
		OptionComment: o,
	}

	ocs.prototype[key] = o

	return o
}

func (ocs *OptionCommentSet) AddPrototypeAndOption(
	key string,
	o OptionComment,
) OptionComment {
	o = ocs.AddPrototype(key, o)
	ocs.OptionComments = append(ocs.OptionComments, o)
	return o
}

func (ocs *OptionCommentSet) Set(v string) (err error) {
	head, tail, _ := strings.Cut(v, ":")

	oc, ok := ocs.prototype[head]

	if ok {
		oc = oc.CloneOptionComment()
	} else {
		oc = &OptionCommentUnknown{}
	}

	oc = OptionCommentWithKey{
		Key:           head,
		OptionComment: oc,
	}

	if err = oc.Set(tail); err != nil {
		err = errors.Wrap(err)
		return err
	}

	ocs.OptionComments = append(
		ocs.OptionComments,
		oc,
	)

	return err
}

// TODO add support for ApplyTo*
type OptionCommentWithKey struct {
	Key string
	OptionComment
}

func (ocf OptionCommentWithKey) CloneOptionComment() OptionComment {
	return ocf
}

func (ocf OptionCommentWithKey) Set(v string) (err error) {
	if err = ocf.OptionComment.Set(v); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (ocf OptionCommentWithKey) String() string {
	return fmt.Sprintf("%s:%s", ocf.Key, ocf.OptionComment)
}

type optionCommentHide string

func (ocf optionCommentHide) CloneOptionComment() OptionComment {
	return ocf
}

func (ocf optionCommentHide) Set(v string) (err error) {
	return comments.Implement()
}

func (ocf optionCommentHide) String() string {
	return fmt.Sprintf("hide:%s", string(ocf))
}

func (ocf optionCommentHide) ApplyToText(Options, *Assignment) (err error) {
	return err
}

func (ocf optionCommentHide) ApplyToReader(
	Options,
	*reader,
) (err error) {
	return err
}

func (ocf optionCommentHide) ApplyToWriter(
	f Options,
	aw *writer,
) (err error) {
	return err
}

type OptionCommentDryRun struct {
	domain_interfaces.MutableConfigDryRun
}

func (ocf *OptionCommentDryRun) CloneOptionComment() OptionComment {
	return ocf
}

func (ocf *OptionCommentDryRun) Set(v string) (err error) {
	var boolValue values.Bool

	if err = boolValue.Set(v); err != nil {
		err = errors.Wrap(err)
		return err
	}

	ocf.SetDryRun(boolValue.Bool())

	return err
}

func (ocf *OptionCommentDryRun) String() string {
	return fmt.Sprintf("%t", ocf.IsDryRun())
}

type OptionCommentUnknown struct {
	Value string
}

func (ocf OptionCommentUnknown) CloneOptionComment() OptionComment {
	return &OptionCommentUnknown{Value: ocf.Value}
}

func (ocf *OptionCommentUnknown) Set(v string) (err error) {
	ocf.Value = v
	return err
}

func (ocf OptionCommentUnknown) String() string {
	return ocf.Value
}

type OptionCommentBooleanFlag struct {
	Value   *bool
	Comment string
}

func (ocf OptionCommentBooleanFlag) CloneOptionComment() OptionComment {
	return ocf
}

func (ocf OptionCommentBooleanFlag) Set(v string) (err error) {
	head, tail, _ := strings.Cut(v, " ")

	var b values.Bool

	if err = b.Set(head); err != nil {
		err = errors.Wrap(err)
		return err
	}

	*ocf.Value = b.Bool()

	ocf.Comment = tail

	return err
}

func (ocf OptionCommentBooleanFlag) String() string {
	if ocf.Comment != "" {
		return fmt.Sprintf("%t %s", *ocf.Value, ocf.Comment)
	} else {
		return fmt.Sprintf("%t", *ocf.Value)
	}
}
