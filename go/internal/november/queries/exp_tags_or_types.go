package queries

import (
	"fmt"
	"strings"

	"code.linenisgreat.com/dodder/go/lib/_/interfaces"
	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/charlie/doddish"
	"code.linenisgreat.com/dodder/go/internal/charlie/genres"
	"code.linenisgreat.com/dodder/go/internal/echo/ids"
	"code.linenisgreat.com/dodder/go/internal/juliett/sku"
)

type expTagsOrTypes struct {
	Or       bool
	Negated  bool
	Exact    bool
	Hidden   bool
	Debug    bool
	Children []sku.Query
}

func (expression *expTagsOrTypes) Clone() (b *expTagsOrTypes) {
	b = &expTagsOrTypes{
		Or:      expression.Or,
		Negated: expression.Negated,
		Exact:   expression.Exact,
		Hidden:  expression.Hidden,
		Debug:   expression.Debug,
	}

	b.Children = make([]sku.Query, len(expression.Children))

	for i, c := range expression.Children {
		switch ct := c.(type) {
		case *expTagsOrTypes:
			b.Children[i] = ct.Clone()

		default:
			b.Children[i] = ct
		}
	}

	return b
}

func (expression *expTagsOrTypes) CollectTags(tags ids.TagSetMutable) {
	if expression.Or || expression.Negated {
		return
	}

	for _, childExpression := range expression.Children {
		switch childExpression := childExpression.(type) {
		case *expTagsOrTypes:
			childExpression.CollectTags(tags)

		case *ObjectId:
			if childExpression.GetGenre() != genres.Tag {
				continue
			}

			tag := ids.MustTag(childExpression.GetObjectId().String())
			tags.Add(tag)
		}
	}
}

func (expression *expTagsOrTypes) reduce(buildState *buildState) (err error) {
	if expression.Exact {
		for _, child := range expression.Children {
			switch childExpression := child.(type) {
			case *ObjectId:
				childExpression.Exact = true

			case *expTagsOrTypes:
				childExpression.Exact = true

			default:
				continue
			}
		}
	}

	chillen := make([]sku.Query, 0, len(expression.Children))

	for _, m := range expression.Children {
		switch mt := m.(type) {
		case *expTagsOrTypes:
			if err = mt.reduce(buildState); err != nil {
				err = errors.Wrap(err)
				return err
			}

			if len(mt.Children) == 0 {
				continue
			}

			if mt.Or == expression.Or && mt.Negated == expression.Negated && mt.Exact == expression.Exact {
				chillen = append(chillen, mt.Children...)
				continue
			}

		case reducer:
			if err = mt.reduce(buildState); err != nil {
				err = errors.Wrap(err)
				return err
			}
		}

		chillen = append(chillen, m)
	}

	expression.Children = chillen

	return err
}

func (expression *expTagsOrTypes) Add(query sku.Query) (err error) {
	switch query := query.(type) {
	case *expTagsOrTypes:

	case *ObjectId:
		query.Exact = expression.Exact
	}

	expression.Children = append(expression.Children, query)

	return err
}

func (expression *expTagsOrTypes) Operator() rune {
	if expression.Or {
		return doddish.OpOr.ToRune()
	} else {
		return doddish.OpAnd.ToRune()
	}
}

func (expression *expTagsOrTypes) StringDebug() string {
	var sb strings.Builder

	op := expression.Operator()

	if expression.Negated {
		sb.WriteRune('^')
	}

	sb.WriteRune(doddish.OpGroupOpen.ToRune())
	fmt.Fprintf(&sb, "(%d)", len(expression.Children))

	for i, m := range expression.Children {
		if i > 0 {
			sb.WriteRune(op)
		}

		sb.WriteString(m.String())
	}

	sb.WriteRune(doddish.OpGroupClose.ToRune())

	return sb.String()
}

func (expression *expTagsOrTypes) String() string {
	if expression.Hidden {
		return ""
	}

	l := len(expression.Children)

	if l == 0 {
		return ""
	}

	var sb strings.Builder

	op := expression.Operator()

	if expression.Negated {
		sb.WriteRune('^')
	}

	switch l {
	case 1:
		sb.WriteString(expression.Children[0].String())

	default:
		sb.WriteRune(doddish.OpGroupOpen.ToRune())

		for i, m := range expression.Children {
			if i > 0 {
				sb.WriteRune(op)
			}

			sb.WriteString(m.String())
		}

		sb.WriteRune(doddish.OpGroupClose.ToRune())
	}

	return sb.String()
}

func (expression *expTagsOrTypes) negateIfNecessary(value bool) bool {
	if expression.Negated {
		return !value
	} else {
		return value
	}
}

func (expression *expTagsOrTypes) ContainsSku(
	objectGetter sku.TransactedGetter,
) (ok bool) {
	if len(expression.Children) == 0 {
		ok = expression.negateIfNecessary(true)
		return ok
	}

	if expression.Or {
		ok = expression.containsMatchableOr(objectGetter)
	} else {
		ok = expression.containsMatchableAnd(objectGetter)
	}

	return ok
}

func (expression *expTagsOrTypes) containsMatchableAnd(
	tg sku.TransactedGetter,
) bool {
	for _, m := range expression.Children {
		if !m.ContainsSku(tg) {
			return expression.negateIfNecessary(false)
		}
	}

	return expression.negateIfNecessary(true)
}

func (expression *expTagsOrTypes) containsMatchableOr(
	tg sku.TransactedGetter,
) bool {
	for _, m := range expression.Children {
		if m.ContainsSku(tg) {
			return expression.negateIfNecessary(true)
		}
	}

	return expression.negateIfNecessary(false)
}

func (expression *expTagsOrTypes) Each(
	f interfaces.FuncIter[sku.Query],
) (err error) {
	for _, m := range expression.Children {
		if err = f(m); err != nil {
			err = errors.Wrap(err)
			return err
		}
	}

	return err
}
