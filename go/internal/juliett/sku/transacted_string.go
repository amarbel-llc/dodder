package sku

import "fmt"

func (transacted *Transacted) String() string {
	return fmt.Sprintf(
		"%s %s %s",
		&transacted.ObjectId,
		transacted.GetObjectDigest(),
		transacted.GetBlobDigest(),
	)
}

func (transacted *Transacted) StringObjectIdDescription() string {
	return fmt.Sprintf(
		"[%s %q]",
		&transacted.ObjectId,
		transacted.GetMetadata().GetDescription(),
	)
}

func (transacted *Transacted) StringObjectIdTai() string {
	return fmt.Sprintf(
		"%s@%s",
		&transacted.ObjectId,
		transacted.GetTai().StringDefaultFormat(),
	)
}

func (transacted *Transacted) StringObjectIdTaiBlob() string {
	return fmt.Sprintf(
		"%s@%s@%s",
		&transacted.ObjectId,
		transacted.GetTai().StringDefaultFormat(),
		transacted.GetBlobDigest(),
	)
}

func (transacted *Transacted) StringObjectIdSha() string {
	return fmt.Sprintf(
		"%s@%s",
		&transacted.ObjectId,
		transacted.GetMetadataMutable().GetObjectDigest(),
	)
}

func (transacted *Transacted) StringObjectIdParent() string {
	return fmt.Sprintf(
		"%s^@%s",
		&transacted.ObjectId,
		transacted.GetMetadataMutable().GetMotherObjectSig(),
	)
}
