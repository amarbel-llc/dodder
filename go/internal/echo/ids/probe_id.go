package ids

import "code.linenisgreat.com/dodder/go/internal/alfa/domain_interfaces"

type ProbeId struct {
	Key string
	Id  domain_interfaces.MarklId
}

type ProbeIdWithObjectId struct {
	ProbeId
	ObjectId *ObjectId
}
