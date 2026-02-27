//go:build chrest

package local_working_copy

import (
	"encoding/json"

	"code.linenisgreat.com/chrest/go/src/bravo/client"
	"code.linenisgreat.com/dodder/go/internal/_/interfaces"
	"code.linenisgreat.com/dodder/go/internal/alfa/errors"
	"code.linenisgreat.com/dodder/go/internal/juliett/sku"
	"code.linenisgreat.com/dodder/go/internal/kilo/sku_json_fmt"
)

func init() {
	formatters["json-toml-bookmark"] = FormatFuncConstructorEntry{
		FormatFuncConstructor: func(
			repo *Repo,
			writer interfaces.WriterAndStringWriter,
		) interfaces.FuncIter[*sku.Transacted] {
			enc := json.NewEncoder(writer)

			var resp client.ResponseWithParsedJSONBody
			var b client.BrowserProxy

			req := client.BrowserRequest{
				Method: "GET",
				Path:   "/tabs",
			}

			if err := b.Read(); err != nil {
				repo.Cancel(err)
				return nil
			}

			var err error
			if resp, err = b.Request(req); err != nil {
				repo.Cancel(err)
				return nil
			}

			tabs := resp.ParsedJSONBody.([]interface{})

			return func(object *sku.Transacted) (err error) {
				var objectJSON sku_json_fmt.JsonWithUrl

				if objectJSON, err = sku_json_fmt.MakeJsonTomlBookmark(
					object,
					repo.GetStore().GetEnvRepo(),
					tabs,
				); err != nil {
					err = errors.Wrap(err)
					return err
				}

				if err = enc.Encode(objectJSON); err != nil {
					err = errors.Wrap(err)
					return err
				}

				return err
			}
		},
	}
}
