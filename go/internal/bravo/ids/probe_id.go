package ids

import (
	mad_domain_interfaces "code.linenisgreat.com/madder/go/pkgs/domain_interfaces"
)

type ProbeId struct {
	Key string
	Id  mad_domain_interfaces.MarklId
}

type ProbeIdWithObjectId struct {
	ProbeId
	ObjectId *ObjectId
}
