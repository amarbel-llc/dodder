package objects

import (
	mad_domain_interfaces "code.linenisgreat.com/madder/go/pkgs/domain_interfaces"
)

func (metadata *metadata) GetBlobDigest() mad_domain_interfaces.MarklId {
	return &metadata.digBlob
}

func (metadata *metadata) GetBlobDigestMutable() mad_domain_interfaces.MarklIdMutable {
	return &metadata.digBlob
}

func (metadata *metadata) GetObjectDigest() mad_domain_interfaces.MarklId {
	return &metadata.digSelf
}

func (metadata *metadata) GetObjectDigestMutable() mad_domain_interfaces.MarklIdMutable {
	return &metadata.digSelf
}

func (metadata *metadata) GetMotherObjectSig() mad_domain_interfaces.MarklId {
	return &metadata.sigMother
}

func (metadata *metadata) GetMotherObjectSigMutable() mad_domain_interfaces.MarklIdMutable {
	return &metadata.sigMother
}

func (metadata *metadata) GetRepoPubKey() mad_domain_interfaces.MarklId {
	return metadata.pubRepo
}

func (metadata *metadata) GetRepoPubKeyMutable() mad_domain_interfaces.MarklIdMutable {
	return &metadata.pubRepo
}

func (metadata *metadata) GetObjectSig() mad_domain_interfaces.MarklId {
	return &metadata.sigRepo
}

func (metadata *metadata) GetObjectSigMutable() mad_domain_interfaces.MarklIdMutable {
	return &metadata.sigRepo
}
