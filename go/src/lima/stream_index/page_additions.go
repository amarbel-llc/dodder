package stream_index

import (
	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/juliett/sku"
)

type additions interface {
	add(object *sku.Transacted)
	hasChanges() bool
	Len() int
	Reset()
	All() interfaces.Seq[*sku.Transacted]
	containsObjectId(string) bool
}

// pageAdditions is the in-memory additions implementation used for normal
// (non-reindex) commit paths.
type pageAdditions struct {
	defaultObjectDigestMarklFormatId string
	index                            *Index
	objectIdLookup                   map[string]struct{}
	objects                          *sku.HeapTransacted
}

func (pa *pageAdditions) initialize(index *Index) {
	index.defaultObjectDigestMarklFormatId = index.envRepo.GetObjectDigestType()

	pa.index = index
	pa.objects = sku.MakeListTransacted()
	pa.objectIdLookup = make(map[string]struct{})
}

func (pa *pageAdditions) add(object *sku.Transacted) {
	objectClone, _ := object.CloneTransacted()

	pa.objects.Add(objectClone)
	pa.objectIdLookup[object.ObjectId.String()] = struct{}{}

	seqProbeIds := object.AllProbeIds(
		pa.index.index.GetHashType(),
		pa.defaultObjectDigestMarklFormatId,
	)

	additionProbes := pa.index.probeIndex.additionProbes

	for probeId := range seqProbeIds {
		idBytes := probeId.Id.GetBytes()
		additionProbes.Set(string(idBytes), objectClone)
	}
}

func (pa *pageAdditions) hasChanges() bool {
	return pa.Len() > 0
}

func (pa *pageAdditions) Len() int {
	return pa.objects.Len()
}

func (pa *pageAdditions) Reset() {
	pa.objects.Reset()
}

func (pa *pageAdditions) All() interfaces.Seq[*sku.Transacted] {
	return pa.objects.All()
}

func (pa *pageAdditions) containsObjectId(objectIdString string) bool {
	_, ok := pa.objectIdLookup[objectIdString]
	return ok
}
