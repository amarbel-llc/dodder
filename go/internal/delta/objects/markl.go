package objects

import "code.linenisgreat.com/dodder/go/internal/_/domain_interfaces"

func (metadata *metadata) GetBlobDigest() domain_interfaces.MarklId {
	return &metadata.DigBlob
}

func (metadata *metadata) GetBlobDigestMutable() domain_interfaces.MarklIdMutable {
	return &metadata.DigBlob
}

func (metadata *metadata) GetObjectDigest() domain_interfaces.MarklId {
	return &metadata.digSelf
}

func (metadata *metadata) GetObjectDigestMutable() domain_interfaces.MarklIdMutable {
	return &metadata.digSelf
}

func (metadata *metadata) GetMotherObjectSig() domain_interfaces.MarklId {
	return &metadata.sigMother
}

func (metadata *metadata) GetMotherObjectSigMutable() domain_interfaces.MarklIdMutable {
	return &metadata.sigMother
}

func (metadata *metadata) GetRepoPubKey() domain_interfaces.MarklId {
	return metadata.pubRepo
}

func (metadata *metadata) GetRepoPubKeyMutable() domain_interfaces.MarklIdMutable {
	return &metadata.pubRepo
}

func (metadata *metadata) GetObjectSig() domain_interfaces.MarklId {
	return &metadata.sigRepo
}

func (metadata *metadata) GetObjectSigMutable() domain_interfaces.MarklIdMutable {
	return &metadata.sigRepo
}
