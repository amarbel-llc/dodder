package sku

import (
	"crypto/sha256"
	"io"
	"strings"
	"testing"

	mad_domain_interfaces "code.linenisgreat.com/madder/go/pkgs/domain_interfaces"
	mad_env_dir "code.linenisgreat.com/madder/go/pkgs/env_dir"

	"code.linenisgreat.com/dodder/go/internal/bravo/ids"
	"code.linenisgreat.com/dodder/go/internal/delta/objects"
	"code.linenisgreat.com/dodder/go/internal/echo/object_metadata_fmt_hyphence"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/env_repo"
	"code.linenisgreat.com/dodder/go/lib/alfa/ohio"
	"code.linenisgreat.com/dodder/go/lib/alfa/quiter"
	"code.linenisgreat.com/madder/go/pkgs/markl_io"
	"code.linenisgreat.com/piggy/go/pkgs/markl"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/collections_ptr"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/pool"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/ui"
)

type inlineTypChecker struct {
	answer bool
}

func (t inlineTypChecker) IsInlineTyp(k ids.Type) bool {
	return t.answer
}

func makeTagSet(t *ui.TestContext, vs ...string) (es ids.TagSet) {
	var err error

	es, err = collections_ptr.MakeValueSetString[ids.TagStruct](nil, vs...)
	t.AssertNoError(err)

	return es
}

func readFormat(
	t1 *ui.TestContext,
	format object_metadata_fmt_hyphence.Format,
	contents string,
) (metadata objects.MetadataMutable) {
	var object Transacted

	t := t1

	reader, repool := pool.GetStringReader(contents)
	defer repool()
	n, err := format.ParseMetadata(
		reader,
		&object,
	)
	t.AssertNoError(err)

	t.AssertEqual(int64(len(contents)), n)

	metadata = object.GetMetadataMutable()

	return metadata
}

func TestMakeTags(t1 *testing.T) {
	ui.RunTestContext(t1, testMakeTags)
}

func testMakeTags(t *ui.TestContext) {
	tagStrings := []string{
		"tag1",
		"tag2",
		"tag3",
	}

	var sutTagSet ids.TagSet
	var err error

	sutTagSet, err = ids.MakeTagSetStrings(tagStrings...)
	t.AssertNoError(err)

	t.AssertEqual(3, sutTagSet.Len())

	{
		actualLength := sutTagSet.Len()

		t.AssertEqual(3, actualLength)
	}

	sutTagSet2 := sutTagSet

	t.AssertEqual(3, sutTagSet2.Len())

	{
		actual := quiter.SortedStrings(sutTagSet)

		t.AssertEqual(tagStrings, actual)
	}

	{
		expected := "tag1, tag2, tag3"
		actual := quiter.StringCommaSeparated(sutTagSet)

		t.AssertEqualStrings(expected, actual)
	}

	{
		expected := "tag1, tag2, tag3"
		actual := quiter.StringCommaSeparated(sutTagSet)

		t.AssertEqualStrings(expected, actual)
	}
}

func TestEqualitySelf(t1 *testing.T) {
	ui.RunTestContext(t1, testEqualitySelf)
}

func testEqualitySelf(t *ui.TestContext) {
	text := objects.MakeBuilder().
		WithDescription("the title").
		WithType("text").
		WithTags(makeTagSet(
			t,
			"tag1",
			"tag2",
			"tag3",
		)).
		Build()

	t.AssertTrue(objects.Equaler.Equals(&text, &text), "expected text to equal itself")
}

func TestEqualityNotSelf(t1 *testing.T) {
	ui.RunTestContext(t1, testEqualityNotSelf)
}

func testEqualityNotSelf(t *ui.TestContext) {
	tags := makeTagSet(
		t,
		"tag1",
		"tag2",
		"tag3",
	)

	text := objects.MakeBuilder().
		WithDescription("the title").
		WithType("text").
		WithTags(tags).
		Build()

	text1 := objects.MakeBuilder().
		WithDescription("the title").
		WithType("text").
		WithTags(tags).
		Build()

	t.AssertTrue(objects.Equaler.Equals(&text, &text1), "expected text to equal text1")
}

func makeTestTextFormatFactory(
	envDir mad_env_dir.Env,
	blobStore mad_domain_interfaces.BlobStore,
) object_metadata_fmt_hyphence.Factory {
	return object_metadata_fmt_hyphence.Factory{
		AllowMissingTypeSig: true,
		EnvDir:              envDir,
		BlobStore:           blobStore,
	}
}

func makeTestTextFormat(
	envDir mad_env_dir.Env,
	blobStore mad_domain_interfaces.BlobStore,
) object_metadata_fmt_hyphence.Format {
	return makeTestTextFormatFactory(envDir, blobStore).Make()
}

func TestReadWithoutBlob(t1 *testing.T) {
	ui.RunTestContext(t1, testReadWithoutBlob)
}

func testReadWithoutBlob(t *ui.TestContext) {
	envRepo := env_repo.MakeTesting(t, nil)

	actual := readFormat(
		t,
		makeTestTextFormat(envRepo, envRepo.GetDefaultBlobStore()),
		`---
# the title
- tag1
- tag2
- tag3
! md
---
`,
	)

	expected := objects.MakeBuilder().
		WithDescription("the title").
		WithType("md").
		WithTags(makeTagSet(
			t,
			"tag1",
			"tag2",
			"tag3",
		)).
		Build()

	if !objects.Equaler.Equals(actual, &expected) {
		t.Fatalf(
			"zettel:\nexpected: %s\n  actual: %s",
			StringMetadataSansTaiMerkle2(&expected),
			StringMetadataSansTaiMerkle2(actual),
		)
	}

	t.AssertTrue(actual.GetBlobDigest().IsNull(), "blob: expected empty")
}

func TestReadWithoutBlobWithMultilineDescription(t1 *testing.T) {
	ui.RunTestContext(t1, testReadWithoutBlobWithMultilineDescription)
}

func testReadWithoutBlobWithMultilineDescription(t *ui.TestContext) {
	envRepo := env_repo.MakeTesting(t, nil)

	actual := readFormat(
		t,
		makeTestTextFormat(envRepo, envRepo.GetDefaultBlobStore()),
		`---
# the title
# continues
- tag1
- tag2
- tag3
! md
---
`,
	)

	expected := objects.MakeBuilder().
		WithDescription("the title continues").
		WithType("md").
		WithTags(makeTagSet(
			t,
			"tag1",
			"tag2",
			"tag3",
		)).
		Build()

	if !objects.Equaler.Equals(actual, &expected) {
		t.Fatalf("zettel:\nexpected: %#v\n  actual: %#v", &expected, actual)
	}

	t.AssertTrue(actual.GetBlobDigest().IsNull(), "blob: expected empty")
}

func TestReadWithBlob(t1 *testing.T) {
	ui.RunTestContext(t1, testReadWithBlob)
}

func testReadWithBlob(t *ui.TestContext) {
	envRepo := env_repo.MakeTesting(
		t,
		nil,
	)

	actual := readFormat(
		t,
		makeTestTextFormat(envRepo, envRepo.GetDefaultBlobStore()),
		`---
# the title
- tag1
- tag2
- tag3
! md
---

the body`,
	)

	var expectedBlobDigest markl.Id
	t.AssertNoError(expectedBlobDigest.Set(
		"blake2b256-9j5cj9mjnk43k9rq4k2h3lezpl2sn3ura7cf8pa58cgfujw6nwgst7gtwz",
	))

	expected := objects.MakeBuilder().
		WithDescription("the title").
		WithType("md").
		WithBlobDigest(expectedBlobDigest).
		WithTags(makeTagSet(
			t,
			"tag1",
			"tag2",
			"tag3",
		)).
		Build()

	if !objects.Equaler.Equals(actual, &expected) {
		t.Fatalf("zettel:\nexpected: %#v\n  actual: %#v", &expected, actual)
	}
}

type blobReaderFactory struct {
	t     *ui.TestContext
	blobs map[string]string
}

func (blobStore blobReaderFactory) BlobReader(
	digest mad_domain_interfaces.MarklId,
) (readCloser mad_domain_interfaces.BlobReader, err error) {
	var value string
	var ok bool

	if value, ok = blobStore.blobs[digest.String()]; !ok {
		blobStore.t.Fatalf("request for non-existent blob: %s", digest)
	}

	hashType, err := markl.GetFormatHashOrError(
		digest.GetMarklFormat().GetMarklFormatId(),
	)
	blobStore.t.AssertNoError(err)

	hash, _ := hashType.Get() //repool:owned
	readCloser = markl_io.MakeNopReadCloser(
		hash,
		ohio.NopCloser(strings.NewReader(value)),
	)

	return readCloser, err
}

func writeFormat(
	t *ui.TestContext,
	metadata objects.MetadataMutable,
	formatter object_metadata_fmt_hyphence.Formatter,
	includeBlob bool,
	blobBody string,
	options object_metadata_fmt_hyphence.FormatterOptions,
	hashType mad_domain_interfaces.FormatHash,
) (out string) {
	hash := sha256.New()
	reader, repool := pool.GetStringReader(blobBody)
	defer repool()
	_, err := io.Copy(hash, reader)
	t.AssertNoError(err)

	blobDigest, _ := hashType.GetMarklIdForString(blobBody) //repool:owned

	metadata.GetBlobDigestMutable().ResetWithMarklId(blobDigest)

	stringBuilder := &strings.Builder{}

	if _, err := formatter.FormatMetadata(
		stringBuilder,
		object_metadata_fmt_hyphence.FormatterContext{
			EncoderContext:   metadata,
			FormatterOptions: options,
		},
	); err != nil {
		t.Errorf("%s", err)
	}

	out = stringBuilder.String()

	return out
}

func TestWriteWithoutBlob(t1 *testing.T) {
	ui.RunTestContext(t1, testWriteWithoutBlob)
}

func testWriteWithoutBlob(t *ui.TestContext) {
	object := objects.MakeBuilder().
		WithDescription("the title").
		WithType("md").
		WithTags(makeTagSet(
			t,
			"tag1",
			"tag2",
			"tag3",
		)).
		Build()

	envRepo := env_repo.MakeTesting(
		t,
		map[string]string{
			"blake2b256-9j5cj9mjnk43k9rq4k2h3lezpl2sn3ura7cf8pa58cgfujw6nwgst7gtwz": "the body",
		},
	)

	formatFamily := makeTestTextFormatFactory(
		envRepo,
		envRepo.GetDefaultBlobStore(),
	).MakeFormatterFamily()

	actual := writeFormat(
		t,
		&object,
		formatFamily.MetadataOnly,
		false,
		"the body",
		object_metadata_fmt_hyphence.FormatterOptions{},
		envRepo.GetDefaultBlobStore().GetDefaultHashType(),
	)

	expected := `---
# the title
- tag1
- tag2
- tag3
@ blake2b256-9j5cj9mjnk43k9rq4k2h3lezpl2sn3ura7cf8pa58cgfujw6nwgst7gtwz
! md
---
`

	t.AssertEqualStrings(expected, actual)
}

func TestWriteWithInlineBlob(t1 *testing.T) {
	ui.RunTestContext(t1, testWriteWithInlineBlob)
}

func testWriteWithInlineBlob(t *ui.TestContext) {
	object := objects.MakeBuilder().
		WithDescription("the title").
		WithType("md").
		WithTags(makeTagSet(
			t,
			"tag1",
			"tag2",
			"tag3",
		)).
		Build()

	envRepo := env_repo.MakeTesting(
		t,
		map[string]string{
			"blake2b256-9j5cj9mjnk43k9rq4k2h3lezpl2sn3ura7cf8pa58cgfujw6nwgst7gtwz": "the body",
		},
	)

	formatFamily := makeTestTextFormatFactory(
		envRepo,
		envRepo.GetDefaultBlobStore(),
	).MakeFormatterFamily()

	actual := writeFormat(
		t,
		&object,
		formatFamily.InlineBlob,
		true,
		"the body",
		object_metadata_fmt_hyphence.FormatterOptions{},
		envRepo.GetDefaultBlobStore().GetDefaultHashType(),
	)

	expected := `---
# the title
- tag1
- tag2
- tag3
! md
---

the body`

	t.AssertEqual(expected, actual)
}
