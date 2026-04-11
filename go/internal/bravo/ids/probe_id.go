package ids

import "code.linenisgreat.com/dodder/go/internal/0/domain_interfaces"

type ProbeId struct {
	Key string
	Id  domain_interfaces.MarklId
}

type ProbeIdWithObjectId struct {
	ProbeId
	ObjectId *ObjectId
}
