package sku

import (
	_ "encoding/gob"
	"strings"

	"code.linenisgreat.com/dodder/go/src/_/external_state"
	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/charlie/genres"
	"code.linenisgreat.com/dodder/go/src/echo/ids"
	"code.linenisgreat.com/dodder/go/src/echo/markl"
	"code.linenisgreat.com/dodder/go/src/golf/objects"
)

type Transacted struct {
	ObjectId ids.ObjectId
	Metadata objects.MetadataStruct

	ExternalType ids.TypeStruct

	// TODO add support for querying the below
	RepoId           ids.RepoId
	State            external_state.State
	ExternalObjectId ids.ExternalObjectId
}

var (
	_ objects.Getter        = &Transacted{}
	_ objects.GetterMutable = &Transacted{}
	_ TransactedGetter      = &Transacted{}
	_ ExternalLike          = &Transacted{}
	_ ExternalLikeGetter    = &Transacted{}
)

func (transacted *Transacted) GetSkuExternal() *Transacted {
	return transacted
}

func (transacted *Transacted) GetRepoId() ids.RepoId {
	return transacted.RepoId
}

func (transacted *Transacted) GetExternalObjectId() domain_interfaces.ExternalObjectId {
	return &transacted.ExternalObjectId
}

func (transacted *Transacted) GetExternalState() external_state.State {
	return transacted.State
}

func (transacted *Transacted) CloneTransacted() (cloned *Transacted, repool interfaces.FuncRepool) {
	cloned, repool = GetTransactedPool().GetWithRepool()
	TransactedResetter.ResetWith(cloned, transacted)
	return cloned, repool
}

func (transacted *Transacted) GetSku() *Transacted {
	return transacted
}

func (transacted *Transacted) GetTags() ids.TagSet {
	return transacted.GetMetadata().GetTags()
}

func (transacted *Transacted) AllTags() interfaces.Seq[ids.Tag] {
	return transacted.GetMetadata().AllTags()
}

func (transacted *Transacted) AddTag(tag ids.Tag) (err error) {
	if transacted.ObjectId.GetGenre() == genres.Tag &&
		strings.HasPrefix(transacted.ObjectId.String(), tag.String()) {
		return err
	}

	tagKey := tag.String()

	if transacted.GetMetadata().GetIndex().GetImplicitTags().ContainsKey(tagKey) {
		return err
	}

	if err = transacted.GetMetadataMutable().AddTagPtr(tag); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (transacted *Transacted) AddTagPtrFast(tag ids.TagStruct) (err error) {
	if err = transacted.GetMetadataMutable().AddTagPtrFast(tag); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (transacted *Transacted) GetType() ids.Type {
	return transacted.GetMetadata().GetType()
}

func (transacted *Transacted) GetTypeLock() objects.TypeLock {
	return transacted.GetMetadata().GetTypeLock()
}

func (transacted *Transacted) GetMetadata() objects.Metadata {
	return &transacted.Metadata
}

func (transacted *Transacted) GetMetadataMutable() objects.MetadataMutable {
	return &transacted.Metadata
}

func (transacted *Transacted) GetTai() ids.Tai {
	return transacted.GetMetadata().GetTai()
}

func (transacted *Transacted) SetTai(tai ids.Tai) {
	transacted.GetMetadataMutable().GetTaiMutable().ResetWith(tai)
}

func (transacted *Transacted) GetObjectId() *ids.ObjectId {
	return &transacted.ObjectId
}

func (transacted *Transacted) SetObjectIdLike(
	objectIdLike ids.Id,
) (err error) {
	if err = transacted.ObjectId.SetWithId(objectIdLike); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (transacted *Transacted) Equals(other *Transacted) (ok bool) {
	if transacted.GetObjectId().String() != other.GetObjectId().String() {
		return ok
	}

	// TODO-P2 determine why object shas in import test differed
	// if !a.Metadata.Sha().Equals(b.Metadata.Sha()) {
	// 	return
	// }

	if !objects.Equaler.Equals(
		transacted.GetMetadata(),
		other.GetMetadata(),
	) {
		return ok
	}

	return true
}

func (transacted *Transacted) GetGenre() domain_interfaces.Genre {
	return transacted.ObjectId.GetGenre()
}

func (transacted *Transacted) IsNew() bool {
	return transacted.GetMetadata().GetMotherObjectSig().IsNull()
}

func (transacted *Transacted) SetDormant(v bool) {
	transacted.GetMetadataMutable().GetIndexMutable().GetDormantMutable().SetBool(v)
}

func (transacted *Transacted) GetObjectDigest() domain_interfaces.MarklId {
	return transacted.GetMetadataMutable().GetObjectDigest()
}

func (transacted *Transacted) GetBlobDigest() domain_interfaces.MarklId {
	return transacted.GetMetadata().GetBlobDigest()
}

func (transacted *Transacted) SetBlobDigest(
	merkleId domain_interfaces.MarklId,
) (err error) {
	if err = transacted.GetMetadataMutable().GetBlobDigestMutable().SetMarklId(
		merkleId.GetMarklFormat().GetMarklFormatId(),
		merkleId.GetBytes(),
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (transacted *Transacted) GetKey() string {
	return ids.FormattedString(transacted.GetObjectId())
}

func (transacted *Transacted) GetStringProbeKeys() map[string]string {
	return map[string]string{
		"objectId":     transacted.GetObjectId().String(),
		"objectId+tai": transacted.GetObjectId().String() + transacted.GetTai().String(),
	}
}

func (transacted *Transacted) AllProbeIds(
	hashType markl.FormatHash,
	defaultObjectDigestMarklFormatId string,
) interfaces.Seq[ids.ProbeId] {
	return func(yield func(ids.ProbeId) bool) {
		for key, value := range transacted.GetStringProbeKeys() {
			id, repool := hashType.GetMarklIdForString(value)

			probeId := ids.ProbeId{
				Key: key,
				Id:  id,
			}

			if !yield(probeId) {
				repool()
				return
			}

			repool()
		}

		{
			probeId := ids.ProbeId{
				Key: transacted.GetMetadata().GetObjectDigest().GetPurposeId(),
				Id:  transacted.GetMetadata().GetObjectDigest(),
			}

			if !yield(probeId) {
				return
			}
		}

		{
			probeId := ids.ProbeId{
				Key: markl.PurposeV5MetadataDigestWithoutTai,
				Id:  transacted.GetMetadata().GetIndex().GetSelfWithoutTai(),
			}

			if !yield(probeId) {
				return
			}
		}

		{
			probeId := ids.ProbeId{
				Key: markl.PurposeObjectSigV2,
				Id:  transacted.GetMetadata().GetObjectSig(),
			}

			if !yield(probeId) {
				return
			}
		}
	}
}
