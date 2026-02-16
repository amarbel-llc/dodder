package markl

import (
	"crypto/sha256"
	"fmt"
	"hash"
	"io"

	"code.linenisgreat.com/dodder/go/src/_/interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/domain_interfaces"
	"code.linenisgreat.com/dodder/go/src/alfa/errors"
	"code.linenisgreat.com/dodder/go/src/alfa/pool"
	"golang.org/x/crypto/blake2b"
)

type FormatHash struct {
	pool interfaces.Pool[Hash]
	id   string
	null Id
}

var (
	_ domain_interfaces.MarklFormat = FormatHash{}
	_ domain_interfaces.FormatHash  = FormatHash{}

	formatHashes map[string]FormatHash = map[string]FormatHash{}

	// TODO remove unnecessary references
	FormatHashSha256     FormatHash
	FormatHashBlake2b256 FormatHash
)

func init() {
	FormatHashSha256 = makeFormatHash(
		sha256.New,
		FormatIdHashSha256,
		&FormatHashSha256,
	)

	FormatHashBlake2b256 = makeFormatHash(
		func() hash.Hash {
			hash, _ := blake2b.New256(nil)
			return hash
		},
		FormatIdHashBlake2b256,
		&FormatHashBlake2b256,
	)
}

func makeFormatHash(
	constructor func() hash.Hash,
	id string,
	self *FormatHash,
) FormatHash {
	_, alreadyExists := formats[id]

	if alreadyExists {
		panic(fmt.Sprintf("hash type already registered: %q", id))
	}

	formatHash := FormatHash{
		pool: pool.MakeValue(
			func() Hash {
				return Hash{
					hash:       constructor(),
					formatHash: self,
				}
			},
			func(hash Hash) {
				hash.Reset()
			},
		),
		id: id,
	}

	hash := constructor()
	formatHash.null.format = self
	formatHash.null.allocDataIfNecessary(hash.Size())
	formatHash.null.data = hash.Sum(formatHash.null.data)

	formats[id] = formatHash
	formatHashes[id] = formatHash

	return formatHash
}

func GetFormatHashOrError(
	formatHashId string,
) (formatHash FormatHash, err error) {
	var ok bool
	formatHash, ok = formatHashes[formatHashId]

	if !ok {
		err = errors.Errorf("unknown hash format: %q", formatHashId)
		return formatHash, err
	}

	return formatHash, err
}

func (formatHash FormatHash) GetHash() (domain_interfaces.Hash, interfaces.FuncRepool) {
	hash, repool := formatHash.Get()
	return hash, repool
}

func (formatHash *FormatHash) Get() (*Hash, interfaces.FuncRepool) {
	hash, repool := formatHash.pool.GetWithRepool()
	hash.formatHash = formatHash
	hash.written = 0
	hash.hash.Reset()
	return &hash, repool
}

func (formatHash FormatHash) GetMarklFormatId() string {
	return formatHash.id
}

func (formatHash FormatHash) GetSize() int {
	return formatHash.null.GetSize()
}

func (formatHash FormatHash) GetBlobId() (domain_interfaces.MarklIdMutable, interfaces.FuncRepool) {
	hash, repool := formatHash.Get()
	defer repool()

	return hash.GetMarklId()
}

func (formatHash FormatHash) GetMarklIdForString(
	input string,
) (domain_interfaces.MarklId, interfaces.FuncRepool) {
	hash, repool := formatHash.Get()
	defer repool()

	if _, err := io.WriteString(hash, input); err != nil {
		errors.PanicIfError(err)
	}

	return hash.GetMarklId()
}

func (formatHash FormatHash) GetMarklIdForMarklId(
	input domain_interfaces.MarklId,
) (domain_interfaces.MarklId, interfaces.FuncRepool) {
	hash, repool := formatHash.Get()
	defer repool()

	if _, err := hash.Write(input.GetBytes()); err != nil {
		errors.PanicIfError(err)
	}

	return hash.GetMarklId()
}

func (formatHash FormatHash) FromStringContent(
	input string,
) domain_interfaces.MarklId {
	id, _ := formatHash.GetMarklIdForString(input)
	return id
}

func (formatHash FormatHash) GetMarklIdFromStringFormat(
	format string,
	args ...any,
) (domain_interfaces.MarklId, interfaces.FuncRepool) {
	hash, repool := formatHash.Get()
	defer repool()

	if _, err := fmt.Fprintf(hash, format, args...); err != nil {
		errors.PanicIfError(err)
	}

	return hash.GetMarklId()
}

func (formatHash FormatHash) GetBlobIdForHexString(
	input string,
) (domain_interfaces.MarklId, interfaces.FuncRepool) {
	hash, hashRepool := formatHash.pool.GetWithRepool()
	defer hashRepool()

	id, repool := hash.GetMarklId()

	errors.PanicIfError(SetHexBytes(formatHash.id, id, []byte(input)))

	return id, repool
}
