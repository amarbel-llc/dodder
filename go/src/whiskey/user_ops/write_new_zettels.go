package user_ops

import (
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/juliett/sku"
	"code.linenisgreat.com/dodder/go/src/victor/local_working_copy"
)

type WriteNewZettels struct {
	*local_working_copy.Repo
}

func (op WriteNewZettels) RunMany(
	proto sku.Proto,
	count int,
) (results sku.TransactedMutableSet, err error) {
	if err = op.Lock(); err != nil {
		err = errors.Wrap(err)
		return results, err
	}

	results = sku.MakeTransactedMutableSet()

	// TODO-P4 modify this to be run once
	for range count {
		var zt *sku.Transacted

		if zt, err = op.runOneAlreadyLocked(proto); err != nil {
			err = errors.Wrap(err)
			return results, err
		}

		if err = results.Add(zt); err != nil {
			err = errors.Wrap(err)
			return results, err
		}
	}

	if err = op.Unlock(); err != nil {
		err = errors.Wrap(err)
		return results, err
	}

	return results, err
}

func (c WriteNewZettels) RunOne(
	z sku.Proto,
) (result *sku.Transacted, err error) {
	if err = c.Lock(); err != nil {
		err = errors.Wrap(err)
		return result, err
	}

	if result, err = c.runOneAlreadyLocked(z); err != nil {
		err = errors.Wrap(err)
		return result, err
	}

	if err = c.Unlock(); err != nil {
		err = errors.Wrap(err)
		return result, err
	}

	return result, err
}

func (c WriteNewZettels) runOneAlreadyLocked(
	proto sku.Proto,
) (object *sku.Transacted, err error) {
	object, _ = proto.Make()

	if err = c.GetStore().CreateOrUpdateDefaultProto(
		object,
		sku.StoreOptions{
			ApplyProto: true,
		},
	); err != nil {
		err = errors.Wrap(err)
		return object, err
	}

	return object, err
}
