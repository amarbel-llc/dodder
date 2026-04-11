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
	"code.linenisgreat.com/dodder/go/lib/0/interfaces"
	"code.linenisgreat.com/dodder/go/lib/bravo/errors"
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
		content, _, readErr := store.transport.Read(file.Path)
		if readErr != nil {
			continue
		}

		doc, parseErr := orgmode.Parse(string(content))
		if parseErr != nil {
			continue
		}

		// Use the filename (without extension) as the external ID.
		externalId := fileExternalId(file.Path)

		// Extract the first heading as the zettel description.
		description, tags, body := extractFromDocument(doc)

		co, _ := sku.GetCheckedOutPool().GetWithRepool() //repool:owned

		external := co.GetSkuExternal()
		metadata := external.GetMetadataMutable()

		if err = metadata.GetDescriptionMutable().Set(description); err != nil {
			return errors.Wrap(err)
		}

		if folder.TypeId != "" {
			if err = metadata.GetTypeMutable().SetType(folder.TypeId); err != nil {
				return errors.Wrap(err)
			}
		}

		// Add per-folder tags from the mapping.
		for _, tagStr := range folder.Tags {
			if addErr := metadata.AddTagString(tagStr); addErr != nil {
				continue
			}
		}

		// Add tags from the orgmode heading.
		for _, tagStr := range tags {
			if addErr := metadata.AddTagString(tagStr); addErr != nil {
				continue
			}
		}

		if body != "" {
			if err = store.writeBlob(external, []byte(body)); err != nil {
				return errors.Wrapf(err, "write blob for %s", externalId)
			}
		}

		if err = external.GetExternalObjectIdMutable().SetWithGenre(
			externalId,
			genres.Zettel,
		); err != nil {
			return errors.Wrap(err)
		}

		// Check if this orgmode file has an existing dodder binding.
		if bound, ok := bindings[externalId]; ok {
			sku.TransactedResetter.ResetWith(co.GetSku(), bound)
			co.SetState(checked_out_state.CheckedOut)
		} else {
			co.GetSku().GetObjectIdMutable().SetGenre(genres.Zettel)
			co.SetState(checked_out_state.Untracked)
		}

		if err = output(co); err != nil {
			return errors.Wrap(err)
		}
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

// Compile reads an orgmode file and returns dodder-compatible fields.
// external -> dodder
func (store *Store) Compile(req haustoria.CompileRequest) (haustoria.CompileResult, error) {
	for _, folder := range store.folders {
		files, err := store.transport.List(folder.Path)
		if err != nil {
			continue
		}

		for _, file := range files {
			externalId := fileExternalId(file.Path)
			if externalId != req.ExternalId {
				continue
			}

			content, etag, readErr := store.transport.Read(file.Path)
			if readErr != nil {
				return haustoria.CompileResult{}, readErr
			}

			doc, parseErr := orgmode.Parse(string(content))
			if parseErr != nil {
				return haustoria.CompileResult{}, parseErr
			}

			description, tags, body := extractFromDocument(doc)

			// Merge folder tags with heading tags without mutating folder.Tags.
			allTags := make([]string, 0, len(folder.Tags)+len(tags))
			allTags = append(allTags, folder.Tags...)
			allTags = append(allTags, tags...)

			return haustoria.CompileResult{
				ExternalId:  externalId,
				Description: description,
				Blob:        []byte(body),
				Tags:        allTags,
				TypeId:      folder.TypeId,
				ETag:        etag,
			}, nil
		}
	}

	return haustoria.CompileResult{}, fmt.Errorf(
		"orgmode file not found: %s", req.ExternalId,
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
			resources = append(resources, haustoria.ExternalResource{
				ExternalId:  fileExternalId(file.Path),
				TypeId:      folder.TypeId,
				Description: strings.TrimSuffix(file.Name, ".org"),
			})
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
