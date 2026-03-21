package import_plan

import (
	"sort"

	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/bravo/markl"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/internal/hotel/store_abbr"
	"code.linenisgreat.com/dodder/go/lib/_/dagnabit"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
	chai "github.com/brandondube/tai"
)

// ObjectTransform mutates an object before plan classification.
// Return true to keep the object, false to drop it entirely.
type ObjectTransform func(*sku.Transacted) (keep bool, err error)

type Builder struct {
	index sku.Index

	sourcePaths   []string
	entries       []Entry
	objectByKey   map[string]int
	taiByObjectId map[string]ids.Tai
	edges         []dagnabit.Edge
	typeNameToKey map[string]string

	dedupFormatId string
	dedupLookup   map[string]struct{}

	abbrIndex  *store_abbr.InMemoryIndex
	transforms []ObjectTransform
}

func MakeImportBuilder(
	index sku.Index,
	dedupFormatId string,
) Builder {
	return Builder{
		index:         index,
		objectByKey:   make(map[string]int),
		taiByObjectId: make(map[string]ids.Tai),
		typeNameToKey: make(map[string]string),
		dedupFormatId: dedupFormatId,
		dedupLookup:   make(map[string]struct{}),
		abbrIndex:     store_abbr.NewInMemoryIndex(),
	}
}

func MakeLocalBuilder() Builder {
	return Builder{
		objectByKey:   make(map[string]int),
		taiByObjectId: make(map[string]ids.Tai),
		typeNameToKey: make(map[string]string),
		dedupLookup:   make(map[string]struct{}),
		abbrIndex:     store_abbr.NewInMemoryIndex(),
	}
}

func (b *Builder) PeekEntries() []Entry {
	return b.entries
}

func (b *Builder) AddTransform(t ObjectTransform) {
	b.transforms = append(b.transforms, t)
}

func (b *Builder) AddSourcePath(path string) int {
	idx := len(b.sourcePaths)
	b.sourcePaths = append(b.sourcePaths, path)
	return idx
}

func entryKey(objectId string, tai ids.Tai) string {
	return objectId + "\x00" + tai.String()
}

func (b *Builder) nextAvailableTai(
	objectId ids.Id,
	tai ids.Tai,
) ids.Tai {
	objectIdString := objectId.String()

	for {
		tai.Asec += int64(chai.Attosecond)
		key := entryKey(objectIdString, tai)

		if _, inBatch := b.objectByKey[key]; inBatch {
			continue
		}

		if b.index != nil {
			if _, err := b.index.ReadOneObjectIdTai(
				objectId,
				tai,
			); err == nil {
				continue
			}
		}

		break
	}

	return tai
}

func (b *Builder) AddObject(
	object *sku.Transacted,
	sourceIndex int,
) {
	genre := genres.Make(object.GetGenre())

	if b.index != nil && genre == genres.Config {
		return
	}

	for _, transform := range b.transforms {
		keep, err := transform(object)
		if err != nil {
			return
		}

		if !keep {
			return
		}
	}

	b.abbrIndex.AddObject(object)

	var entry Entry
	entry.SourceIndex = sourceIndex
	sku.TransactedResetter.ResetWith(&entry.object, object)

	objectIdString := object.GetObjectId().String()
	tai := object.GetTai()
	entry.OriginalTai.ResetWith(tai)

	if genre == genres.Type && object.GetBlobDigest().IsNull() {
		entry.Classification = ClassificationSkipBloblessType
		b.appendEntry(entry)
		return
	}

	if b.dedupFormatId != "" {
		id, repool := markl.GetId()

		if err := object.CalculateDigestForPurpose(
			b.dedupFormatId,
			id,
		); err == nil {
			digestKey := string(id.GetBytes())
			repool()

			if _, seen := b.dedupLookup[digestKey]; seen {
				entry.Classification = ClassificationSkipDedup
				b.appendEntry(entry)
				return
			}

			b.dedupLookup[digestKey] = struct{}{}
		} else {
			repool()
		}
	}

	key := entryKey(objectIdString, tai)

	if _, inBatch := b.objectByKey[key]; inBatch {
		newTai := b.nextAvailableTai(object.GetObjectId(), tai)
		entry.object.SetTai(newTai)
		entry.Classification = ClassificationResolveTaiReassign
		key = entryKey(objectIdString, newTai)
		b.appendEntryWithKey(entry, key)
		return
	}

	if b.index != nil {
		existing, err := b.index.ReadOneObjectIdTai(
			object.GetObjectId(),
			tai,
		)

		if err == nil {
			localBlobDigest := existing.GetBlobDigest().String()
			remoteBlobDigest := object.GetBlobDigest().String()

			if localBlobDigest == remoteBlobDigest {
				entry.Classification = ClassificationSkipExists
				b.appendEntry(entry)
				return
			}

			newTai := b.nextAvailableTai(object.GetObjectId(), tai)
			entry.object.SetTai(newTai)
			entry.Classification = ClassificationResolveTaiReassign
			key = entryKey(objectIdString, newTai)
			b.appendEntryWithKey(entry, key)
			return
		} else if !errors.IsErrNotFound(err) {
			entry.Classification = ClassificationErrorMissingBlob
			b.appendEntry(entry)
			return
		}
	}

	entry.Classification = ClassificationImport
	b.appendEntryWithKey(entry, key)
}

func (b *Builder) appendEntry(entry Entry) {
	b.entries = append(b.entries, entry)
}

func (b *Builder) appendEntryWithKey(entry Entry, key string) {
	idx := len(b.entries)
	b.entries = append(b.entries, entry)
	b.objectByKey[key] = idx

	objectIdString := entry.object.GetObjectId().String()
	tai := entry.object.GetTai()

	if existing, ok := b.taiByObjectId[objectIdString]; !ok || tai.After(existing) {
		b.taiByObjectId[objectIdString] = tai
	}

	genre := genres.Make(entry.object.GetGenre())

	if genre == genres.Type {
		typeString := entry.object.GetObjectId().String()
		b.typeNameToKey[typeString] = key
	}

	if typeId := entry.object.GetType(); !typeId.IsEmpty() {
		typeString := typeId.String()

		if !ids.IsBuiltin(typeId) {
			b.edges = append(b.edges, dagnabit.Edge{
				Source: key,
				Target: "type:" + typeString,
			})
		}
	}
}

func (b *Builder) Build() (*Plan, error) {
	resolvedEdges := make([]dagnabit.Edge, 0, len(b.edges))

	for _, edge := range b.edges {
		targetTypeString := edge.Target[len("type:"):]

		if typeKey, ok := b.typeNameToKey[targetTypeString]; ok {
			resolvedEdges = append(resolvedEdges, dagnabit.Edge{
				Source: edge.Source,
				Target: typeKey,
			})
		}
	}

	heights, err := dagnabit.TopologicalSort(resolvedEdges)
	if err != nil {
		return nil, errors.Wrapf(err, "cycle in type dependencies")
	}

	for i := range b.entries {
		entry := &b.entries[i]
		key := entryKey(
			entry.object.GetObjectId().String(),
			entry.object.GetTai(),
		)

		if h, ok := heights[key]; ok {
			entry.Height = h
		}
	}

	erroredTypes := make(map[string]bool)

	for i := range b.entries {
		entry := &b.entries[i]
		genre := genres.Make(entry.object.GetGenre())

		if genre != genres.Type {
			continue
		}

		if entry.Classification == ClassificationSkipBloblessType ||
			entry.Classification == ClassificationErrorMissingBlob {
			typeString := entry.object.GetObjectId().String()
			erroredTypes[typeString] = true
		}
	}

	for i := range b.entries {
		entry := &b.entries[i]

		if !entry.Classification.IsCommittable() {
			continue
		}

		typeId := entry.object.GetType()
		if typeId.IsEmpty() || ids.IsBuiltin(typeId) {
			continue
		}

		typeString := typeId.String()
		if erroredTypes[typeString] {
			entry.Classification = ClassificationErrorMissingBlob
			entry.ErrorCause = typeString
		}
	}

	sort.SliceStable(b.entries, func(i, j int) bool {
		if b.entries[i].Height != b.entries[j].Height {
			return b.entries[i].Height < b.entries[j].Height
		}

		return b.entries[i].object.GetTai().Before(b.entries[j].object.GetTai())
	})

	plan := &Plan{
		Entries:     b.entries,
		SourcePaths: b.sourcePaths,
		Abbr:        b.abbrIndex.GetAbbr(),
	}

	for i := range plan.Entries {
		if plan.Entries[i].Classification.IsError() {
			plan.HasErrors = true
			break
		}
	}

	return plan, nil
}
