package sku

import (
	"crypto/sha256"
	"io"
	"reflect"
	"strings"
	"testing"

	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/pool"
	"code.linenisgreat.com/dodder/go/src/bravo/markl_io"
	"code.linenisgreat.com/dodder/go/src/bravo/ohio"
	"code.linenisgreat.com/dodder/go/src/bravo/quiter"
	"code.linenisgreat.com/dodder/go/src/bravo/ui"
	"code.linenisgreat.com/dodder/go/src/charlie/collections_ptr"
	"code.linenisgreat.com/dodder/go/src/echo/ids"
	"code.linenisgreat.com/dodder/go/src/echo/markl"
	"code.linenisgreat.com/dodder/go/src/golf/objects"
	"code.linenisgreat.com/dodder/go/src/hotel/env_dir"
	"code.linenisgreat.com/dodder/go/src/india/object_metadata_fmt_triple_hyphen"
	"code.linenisgreat.com/dodder/go/src/juliett/env_repo"
)

type inlineTypChecker struct {
	answer bool
}

func (t inlineTypChecker) IsInlineTyp(k ids.Type) bool {
	return t.answer
}

func makeTagSet(t *ui.TestContext, vs ...string) (es ids.TagSet) {
	var err error

	if es, err = collections_ptr.MakeValueSetString[ids.TagStruct](nil, vs...); err != nil {
		t.Fatalf("%s", err)
	}

	return es
}

func readFormat(
	t1 *ui.TestContext,
	format object_metadata_fmt_triple_hyphen.Format,
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

	if n != int64(len(contents)) {
		t.Fatalf("expected to read %d but only read %d", len(contents), n)
	}

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

	if sutTagSet, err = ids.MakeTagSetStrings(tagStrings...); err != nil {
		t.Fatalf("%s", err)
	}

	if sutTagSet.Len() != 3 {
		t.Fatalf("expected len 3 but got %d", sutTagSet.Len())
	}

	{
		actualLength := sutTagSet.Len()

		if actualLength != 3 {
			t.Fatalf("expected len 3 but got %d", actualLength)
		}
	}

	sutTagSet2 := sutTagSet

	if sutTagSet2.Len() != 3 {
		t.Fatalf("expected len 3 but got %d", sutTagSet2.Len())
	}

	{
		actual := quiter.SortedStrings(sutTagSet)

		if !reflect.DeepEqual(actual, tagStrings) {
			t.Fatalf("expected %q but got %q", tagStrings, actual)
		}
	}

	{
		expected := "tag1, tag2, tag3"
		actual := quiter.StringCommaSeparated(sutTagSet)

		if actual != expected {
			t.Fatalf("expected %q but got %q", expected, actual)
		}
	}

	{
		expected := "tag1, tag2, tag3"
		actual := quiter.StringCommaSeparated(sutTagSet)

		if actual != expected {
			t.Fatalf("expected %q but got %q", expected, actual)
		}
	}
}

func TestEqualitySelf(t1 *testing.T) {
	ui.RunTestContext(t1, testEqualitySelf)
}

func testEqualitySelf(t *ui.TestContext) {
	text := objects.MakeBuilder().
		WithDescription("the title").
		WithType("text").
		WithTags(makeTagSet(t,
			"tag1",
			"tag2",
			"tag3",
		)).
		Build()

	if !objects.Equaler.Equals(&text, &text) {
		t.Fatalf("expected %v to equal itself", text)
	}
}

func TestEqualityNotSelf(t1 *testing.T) {
	ui.RunTestContext(t1, testEqualityNotSelf)
}

func testEqualityNotSelf(t *ui.TestContext) {
	tags := makeTagSet(t,
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

	if !objects.Equaler.Equals(&text, &text1) {
		t.Fatalf("expected %v to equal %v", text, text1)
	}
}

func makeTestTextFormatFactory(
	envDir env_dir.Env,
	blobStore domain_interfaces.BlobStore,
) object_metadata_fmt_triple_hyphen.Factory {
	return object_metadata_fmt_triple_hyphen.Factory{
		AllowMissingTypeSig: true,
		EnvDir:              envDir,
		BlobStore:           blobStore,
	}
}

func makeTestTextFormat(
	envDir env_dir.Env,
	blobStore domain_interfaces.BlobStore,
) object_metadata_fmt_triple_hyphen.Format {
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
		WithTags(makeTagSet(t,
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

	if !actual.GetBlobDigest().IsNull() {
		t.Fatalf("blob:\nexpected empty but got %q", actual.GetBlobDigest())
	}
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
		WithTags(makeTagSet(t,
			"tag1",
			"tag2",
			"tag3",
		)).
		Build()

	if !objects.Equaler.Equals(actual, &expected) {
		t.Fatalf("zettel:\nexpected: %#v\n  actual: %#v", &expected, actual)
	}

	if !actual.GetBlobDigest().IsNull() {
		t.Fatalf("blob:\nexpected empty but got %q", actual.GetBlobDigest())
	}
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
		WithTags(makeTagSet(t,
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
	digest domain_interfaces.MarklId,
) (readCloser domain_interfaces.BlobReader, err error) {
	var value string
	var ok bool

	if value, ok = blobStore.blobs[digest.String()]; !ok {
		blobStore.t.Fatalf("request for non-existent blob: %s", digest)
	}

	hashType, err := markl.GetFormatHashOrError(
		digest.GetMarklFormat().GetMarklFormatId(),
	)
	blobStore.t.AssertNoError(err)

	hash, _ := hashType.Get()
	readCloser = markl_io.MakeNopReadCloser(
		hash,
		ohio.NopCloser(strings.NewReader(value)),
	)

	return readCloser, err
}

func writeFormat(
	t *ui.TestContext,
	metadata objects.MetadataMutable,
	formatter object_metadata_fmt_triple_hyphen.Formatter,
	includeBlob bool,
	blobBody string,
	options object_metadata_fmt_triple_hyphen.FormatterOptions,
	hashType domain_interfaces.FormatHash,
) (out string) {
	hash := sha256.New()
	reader, repool := pool.GetStringReader(blobBody)
	defer repool()
	_, err := io.Copy(hash, reader)
	if err != nil {
		t.Fatalf("%s", err)
	}

	blobDigest, _ := hashType.GetMarklIdForString(blobBody)

	metadata.GetBlobDigestMutable().ResetWithMarklId(blobDigest)

	stringBuilder := &strings.Builder{}

	if _, err := formatter.FormatMetadata(
		stringBuilder,
		object_metadata_fmt_triple_hyphen.FormatterContext{
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
		WithTags(makeTagSet(t,
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
		object_metadata_fmt_triple_hyphen.FormatterOptions{},
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
		WithTags(makeTagSet(t,
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
		object_metadata_fmt_triple_hyphen.FormatterOptions{},
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
