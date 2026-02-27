package inventory_list_coders

import (
	"bufio"
	"encoding/json"
	"io"

	"code.linenisgreat.com/dodder/go/lib/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/hotel/genesis_configs"
	"code.linenisgreat.com/dodder/go/internal/juliett/sku"
	"code.linenisgreat.com/dodder/go/internal/kilo/sku_json_fmt"
)

type jsonV0 struct {
	genesisConfig genesis_configs.ConfigPrivate
}

func (coder jsonV0) EncodeTo(
	object *sku.Transacted,
	bufferedWriter *bufio.Writer,
) (n int64, err error) {
	var objectJson sku_json_fmt.Transacted

	if err = objectJson.FromTransacted(object, nil); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	encoder := json.NewEncoder(bufferedWriter)

	if err = encoder.Encode(objectJson); err != nil {
		err = errors.Wrap(err)
		return n, err
	}

	return n, err
}

func (coder jsonV0) DecodeFrom(
	object *sku.Transacted,
	bufferedReader *bufio.Reader,
) (n int64, err error) {
	var objectJson sku_json_fmt.Transacted

	var bites []byte

	if bites, err = bufferedReader.ReadBytes('\n'); err != nil {
		err = errors.WrapExceptSentinel(err, io.EOF)
		return n, err
	}

	if err = json.Unmarshal(bites, &objectJson); err != nil {
		err = errors.Wrapf(err, "Line: %q", bites)
		return n, err
	}

	if err = objectJson.ToTransacted(object, nil); err != nil {
		err = errors.Wrapf(err, "Line: %q", bites)
		return n, err
	}

	return n, err
}
