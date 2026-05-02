package haustoria_orgmode

import (
	"bytes"
	"fmt"
	"io"
	"path"
	"strings"
	"time"

	"code.linenisgreat.com/dodder/go/internal/0/domain_interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/checkout_options"
	"code.linenisgreat.com/dodder/go/internal/alfa/genres"
	"code.linenisgreat.com/dodder/go/internal/bravo/checked_out_state"
	"code.linenisgreat.com/dodder/go/internal/charlie/haustoria"
	"code.linenisgreat.com/dodder/go/internal/golf/sku"
	"code.linenisgreat.com/dodder/go/internal/hotel/orgmode"
	"code.linenisgreat.com/dodder/go/internal/kilo/queries"
	"code.linenisgreat.com/dodder/go/internal/lima/store_workspace"
	"github.com/amarbel-llc/purse-first/libs/dewey/bravo/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/0/interfaces"
)

// FolderMapping associates a remote folder with a dodder type and optional
// tags. Each folder is listed independently, and discovered .org files are
// compiled into zettels with the configured type and tags.
type FolderMapping struct {
	Path   string // Remote folder path or URL
	TypeId string // Dodder type for objects in this folder
	Tags   []string
}

// Store implements both haustoria.Haustoria and store_workspace.StoreLike
// for orgmode files served over WebDAV or SFTP.
type Store struct {
	transport Transport
	folders   []FolderMapping
	supplies  store_workspace.Supplies
}

var (
	_ haustoria.Haustoria       = &Store{}
	_ store_workspace.StoreLike = &Store{}
)

// MakeStore creates an orgmode haustoria Store with the given transport and
// folder mappings.
func MakeStore(transport Transport, folders []FolderMapping) *Store {
	return &Store{
		transport: transport,
		folders:   folders,
	}
}

// --- StoreLike interface ---

func (store *Store) Initialize(supplies store_workspace.Supplies) error {
	store.supplies = supplies
	return nil
}

func (store *Store) QueryCheckedOut(
	queryGroup *queries.Query,
	output interfaces.FuncIter[sku.SkuType],
) (err error) {
	// Build a lookup of existing bindings: ExternalObjectId string -> Transacted.
	bindings := make(map[string]*sku.Transacted)

	if err = store.supplies.ReadPrimitiveQuery(
		nil,
		func(object *sku.Transacted) error {
			eoid := object.GetExternalObjectId()
			if eoid.IsEmpty() {
				return nil
			}

			cloned, _ := object.CloneTransacted() //repool:owned
			bindings[eoid.String()] = cloned
			return nil
		},
	); err != nil {
		return errors.Wrap(err)
	}

	for _, folder := range store.folders {
		if err = store.queryCheckedOutForFolder(folder, bindings, output); err != nil {
			return err
		}
	}

	return nil
}

func (store *Store) queryCheckedOutForFolder(
	folder FolderMapping,
	bindings map[string]*sku.Transacted,
	output interfaces.FuncIter[sku.SkuType],
) (err error) {
	files, err := store.transport.List(folder.Path)
	if err != nil {
		return errors.Wrapf(err, "list orgmode files in %s", folder.Path)
	}

	for _, file := range files {
		content, etag, readErr := store.transport.Read(file.Path)
		if readErr != nil {
			continue
		}

		headings, parseErr := parseFile(content)
		if parseErr != nil {
			continue
		}

		// Normalize: add :ID: UUID v7 to any heading missing one, then
		// write the file back if anything changed. After normalization,
		// every heading has a stable external identity.
		newContent, changed, normErr := normalizeIDs(content, headings)
		if normErr != nil {
			return errors.Wrap(normErr)
		}

		if changed {
			if writeErr := store.transport.Write(
				file.Path,
				newContent,
				etag,
			); writeErr != nil {
				return errors.Wrapf(
					writeErr,
					"write normalized orgmode file %s",
					file.Path,
				)
			}

			// Re-parse with the new content so byte spans are valid.
			headings, parseErr = parseFile(newContent)
			if parseErr != nil {
				continue
			}
			content = newContent
		}

		if len(headings) == 0 {
			synth, ok := synthesizeHeadingFromContent(content, file.Path)
			if !ok {
				continue
			}
			if err = store.emitHeading(
				folder,
				synth,
				content,
				bindings,
				output,
			); err != nil {
				return err
			}
			continue
		}

		for _, heading := range headings {
			if err = store.emitHeading(
				folder,
				heading,
				content,
				bindings,
				output,
			); err != nil {
				return err
			}
		}
	}

	return nil
}

func (store *Store) emitHeading(
	folder FolderMapping,
	heading ParsedHeading,
	content []byte,
	bindings map[string]*sku.Transacted,
	output interfaces.FuncIter[sku.SkuType],
) (err error) {
	if heading.ID == "" {
		// Headings without an :ID: property are skipped. Normalization
		// should have filled this in for every heading, so this is a
		// defensive no-op.
		return nil
	}

	co, _ := sku.GetCheckedOutPool().GetWithRepool() //repool:owned

	external := co.GetSkuExternal()
	metadata := external.GetMetadataMutable()

	if err = metadata.GetDescriptionMutable().Set(heading.Title); err != nil {
		return errors.Wrap(err)
	}

	if folder.TypeId != "" {
		if err = metadata.GetTypeMutable().SetType(folder.TypeId); err != nil {
			return errors.Wrap(err)
		}
	}

	for _, tagStr := range folder.Tags {
		if addErr := metadata.AddTagString(tagStr); addErr != nil {
			continue
		}
	}

	for _, tagStr := range heading.Tags {
		if addErr := metadata.AddTagString(tagStr); addErr != nil {
			continue
		}
	}

	body := content[heading.BodyStart:heading.BodyEnd]
	if len(body) > 0 {
		if err = store.writeBlob(external, body); err != nil {
			return errors.Wrapf(err, "write blob for %s", heading.ID)
		}
	}

	if err = external.GetExternalObjectIdMutable().SetWithGenre(
		heading.ID,
		genres.Zettel,
	); err != nil {
		return errors.Wrap(err)
	}

	if bound, ok := bindings[heading.ID]; ok {
		sku.TransactedResetter.ResetWith(co.GetSku(), bound)
		co.SetState(checked_out_state.CheckedOut)
	} else {
		co.GetSku().GetObjectIdMutable().SetGenre(genres.Zettel)
		co.SetState(checked_out_state.Untracked)
	}

	if err = output(co); err != nil {
		return errors.Wrap(err)
	}

	return nil
}

func (store *Store) CheckoutOne(
	options checkout_options.Options,
	transactedGetter sku.TransactedGetter,
) (checkedOut sku.SkuType, err error) {
	object := transactedGetter.GetSku()

	description := object.GetMetadata().GetDescription().String()
	typeId := object.GetMetadata().GetType().String()

	var tags []string
	for tag := range object.GetMetadata().GetTags().All() {
		tags = append(tags, tag.String())
	}

	var blob []byte
	if !object.GetBlobDigest().IsNull() {
		if blob, err = store.readBlob(object); err != nil {
			return nil, errors.Wrapf(err,
				"read blob for %s", object.GetObjectId(),
			)
		}
	}

	result, err := store.Decompile(haustoria.DecompileRequest{
		ObjectId:    object.GetObjectId().String(),
		Description: description,
		Blob:        blob,
		Tags:        tags,
		TypeId:      typeId,
	})
	if err != nil {
		return nil, errors.Wrapf(err,
			"decompile %s to orgmode", object.GetObjectId(),
		)
	}

	co, _ := sku.GetCheckedOutPool().GetWithRepool() //repool:owned

	sku.TransactedResetter.ResetWith(co.GetSku(), object)

	external := co.GetSkuExternal()
	sku.TransactedResetter.ResetWith(external, object)

	if err = external.GetExternalObjectIdMutable().SetWithGenre(
		result.ExternalId,
		genres.Zettel,
	); err != nil {
		return nil, errors.Wrap(err)
	}

	co.SetState(checked_out_state.CheckedOut)

	return co, nil
}

func (store *Store) readBlob(object *sku.Transacted) (content []byte, err error) {
	blobReader, err := store.supplies.Env.GetDefaultBlobStore().MakeBlobReader(
		object.GetBlobDigest(),
	)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	defer errors.DeferredCloser(&err, blobReader)

	if content, err = io.ReadAll(blobReader); err != nil {
		return nil, errors.Wrap(err)
	}

	return content, nil
}

func (store *Store) ReadAllExternalItems() error {
	return nil
}

func (store *Store) Flush() error {
	return store.transport.Close()
}

func (store *Store) GetObjectIdsForString(
	v string,
) ([]domain_interfaces.ExternalObjectId, error) {
	return nil, nil
}

func (store *Store) writeBlob(
	object *sku.Transacted,
	content []byte,
) (err error) {
	blobWriter, err := store.supplies.Env.GetDefaultBlobStore().MakeBlobWriter(nil)
	if err != nil {
		return errors.Wrap(err)
	}

	defer errors.DeferredCloser(&err, blobWriter)

	if _, err = bytes.NewReader(content).WriteTo(blobWriter); err != nil {
		return errors.Wrap(err)
	}

	if err = object.SetBlobDigest(blobWriter.GetMarklId()); err != nil {
		return errors.Wrap(err)
	}

	return nil
}

// --- Haustoria interface ---

// Compile reads an orgmode heading and returns dodder-compatible fields.
// external -> dodder. The req.ExternalId is the heading's :ID: UUID.
func (store *Store) Compile(req haustoria.CompileRequest) (haustoria.CompileResult, error) {
	for _, folder := range store.folders {
		files, err := store.transport.List(folder.Path)
		if err != nil {
			continue
		}

		for _, file := range files {
			content, etag, readErr := store.transport.Read(file.Path)
			if readErr != nil {
				continue
			}

			headings, parseErr := parseFile(content)
			if parseErr != nil {
				continue
			}

			if len(headings) == 0 {
				synth, ok := synthesizeHeadingFromContent(content, file.Path)
				if !ok || synth.ID != req.ExternalId {
					continue
				}
				return haustoria.CompileResult{
					ExternalId:  synth.ID,
					Description: synth.Title,
					Blob:        content[synth.BodyStart:synth.BodyEnd],
					Tags:        append([]string(nil), folder.Tags...),
					TypeId:      folder.TypeId,
					ETag:        etag,
				}, nil
			}

			for _, heading := range headings {
				if heading.ID != req.ExternalId {
					continue
				}

				allTags := make([]string, 0, len(folder.Tags)+len(heading.Tags))
				allTags = append(allTags, folder.Tags...)
				allTags = append(allTags, heading.Tags...)

				return haustoria.CompileResult{
					ExternalId:  heading.ID,
					Description: heading.Title,
					Blob:        content[heading.BodyStart:heading.BodyEnd],
					Tags:        allTags,
					TypeId:      folder.TypeId,
					ETag:        etag,
				}, nil
			}
		}
	}

	return haustoria.CompileResult{}, fmt.Errorf(
		"orgmode heading not found: %s", req.ExternalId,
	)
}

// Decompile writes a dodder object to an orgmode file.
// dodder -> external
func (store *Store) Decompile(req haustoria.DecompileRequest) (haustoria.DecompileResult, error) {
	folder := store.folderForType(req.TypeId)
	if folder == nil {
		return haustoria.DecompileResult{}, fmt.Errorf(
			"no folder mapping for type %s", req.TypeId,
		)
	}

	// Build orgmode document from dodder fields.
	props := orgmode.Properties{{Key: "DODDER_ID", Value: req.ObjectId}}

	heading := orgmode.MakeHeading(
		req.Description,
		req.Tags,
		string(req.Blob),
		props,
	)

	doc := orgmode.Document{
		Headings: []orgmode.Heading{heading},
	}

	content := orgmode.Serialize(doc)

	// Determine filename from external ID or object ID.
	externalId := req.ExternalId
	if externalId == "" {
		externalId = sanitizeFilename(req.ObjectId)
		if externalId == "" {
			externalId = fmt.Sprintf("dodder-%d", time.Now().UnixNano())
		}
	}

	filePath := path.Join(folder.Path, externalId+".org")

	if err := store.transport.Write(filePath, []byte(content), req.ETag); err != nil {
		return haustoria.DecompileResult{}, fmt.Errorf("decompile to orgmode: %w", err)
	}

	return haustoria.DecompileResult{
		ExternalId: externalId,
	}, nil
}

func (store *Store) Discover() ([]haustoria.ExternalResource, error) {
	var resources []haustoria.ExternalResource

	for _, folder := range store.folders {
		files, err := store.transport.List(folder.Path)
		if err != nil {
			return nil, fmt.Errorf("discover orgmode files in %s: %w", folder.Path, err)
		}

		for _, file := range files {
			content, _, readErr := store.transport.Read(file.Path)
			if readErr != nil {
				continue
			}

			headings, parseErr := parseFile(content)
			if parseErr != nil {
				continue
			}

			if len(headings) == 0 {
				synth, ok := synthesizeHeadingFromContent(content, file.Path)
				if !ok {
					continue
				}
				resources = append(resources, haustoria.ExternalResource{
					ExternalId:  synth.ID,
					TypeId:      folder.TypeId,
					Description: synth.Title,
				})
				continue
			}

			for _, heading := range headings {
				if heading.ID == "" {
					continue
				}
				resources = append(resources, haustoria.ExternalResource{
					ExternalId:  heading.ID,
					TypeId:      folder.TypeId,
					Description: heading.Title,
				})
			}
		}
	}

	return resources, nil
}

func (store *Store) Delete(externalId string) error {
	for _, folder := range store.folders {
		filePath := path.Join(folder.Path, externalId+".org")
		if err := store.transport.Delete(filePath); err == nil {
			return nil
		}
	}

	return fmt.Errorf("orgmode file not found for delete: %s", externalId)
}

func (store *Store) folderForType(typeId string) *FolderMapping {
	for i := range store.folders {
		if store.folders[i].TypeId == typeId {
			return &store.folders[i]
		}
	}

	// Fall back to first folder.
	if len(store.folders) > 0 {
		return &store.folders[0]
	}

	return nil
}

// extractFromDocument pulls the description, tags, and body from an orgmode
// Document. Uses the first heading if present; falls back to preamble.
func extractFromDocument(doc orgmode.Document) (description string, tags []string, body string) {
	if len(doc.Headings) > 0 {
		heading := doc.Headings[0]
		description = heading.Title
		tags = heading.Tags
		body = heading.Body

		if heading.Keyword != "" {
			// Preserve TODO keyword as a tag (e.g., "TODO" -> "todo").
			tags = append(tags, strings.ToLower(heading.Keyword))
		}

		return description, tags, body
	}

	// No headings: use preamble as body, first line as description.
	lines := strings.SplitN(doc.Preamble, "\n", 2)
	if len(lines) > 0 {
		description = strings.TrimSpace(lines[0])
	}
	if len(lines) > 1 {
		body = strings.TrimSpace(lines[1])
	}

	return description, nil, body
}

// fileExternalId derives the external identifier from a file path by
// stripping the directory and .org extension.
func fileExternalId(filePath string) string {
	base := path.Base(filePath)
	return strings.TrimSuffix(base, ".org")
}

// sanitizeFilename makes an object ID safe for use as a filename by replacing
// slashes with hyphens.
func sanitizeFilename(objectId string) string {
	return strings.ReplaceAll(objectId, "/", "-")
}
