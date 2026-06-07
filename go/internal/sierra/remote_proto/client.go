package remote_proto

import (
	"io"

	"code.linenisgreat.com/dodder/go/internal/bravo/env_ui"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/papa/repo"
	"code.linenisgreat.com/dodder/go/internal/romeo/local_working_copy"
	mad_domain_interfaces "github.com/amarbel-llc/madder/go/pkgs/domain_interfaces"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
)

// Client drives the initiating half of the drtp protocol against a peer
// reached over conn. It is the hard-fork counterpart of remote_http's
// client; the two coexist.
type Client struct {
	env   env_ui.Env
	local *local_working_copy.Repo

	// PinnedPublicKey, when set, rejects a server whose advertised public
	// key does not match (vs. Trust-On-First-Use when nil).
	PinnedPublicKey mad_domain_interfaces.MarklId
}

func MakeClient(local *local_working_copy.Repo) *Client {
	return &Client{
		env:   local.GetEnv(),
		local: local,
	}
}

func (client *Client) keys() keys {
	return keys{
		private: client.local.GetImmutableConfigPrivate().Blob.GetPrivateKey(),
		public:  client.local.GetImmutableConfigPublic().GetPublicKey(),
	}
}

func (client *Client) listType() string {
	return client.local.GetImmutableConfigPublic().GetInventoryListTypeId()
}

// Fetch pulls every object matching query (and its expand-edges closure)
// from the peer on conn into the local repo. Closes conn when done.
func (client *Client) Fetch(
	conn io.ReadWriteCloser,
	query string,
	options repo.ImporterOptions,
) (err error) {
	s := makeSession(conn)
	defer errors.DeferredCloser(&err, s)

	var serverCaps control

	if serverCaps, _, err = clientHandshake(
		s,
		client.keys(),
		client.listType(),
		client.PinnedPublicKey,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	var want control

	if want, err = signWant(client.keys(), serverCaps.Nonce, control{
		Direction:           DirectionFetch,
		Query:               query,
		AllowMergeConflicts: options.AllowMergeConflicts,
		ExcludeBlobs:        options.ExcludeBlobs,
	}); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = s.writeControl(TypeWant, want); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = receiveClosure(
		client.env,
		s,
		client.local,
		want,
		sku.GetStoreOptionsImport(),
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

// Push sends every local object matching query (and its closure) to the
// peer on conn. Closes conn when done.
func (client *Client) Push(
	conn io.ReadWriteCloser,
	query string,
	options repo.ImporterOptions,
) (err error) {
	s := makeSession(conn)
	defer errors.DeferredCloser(&err, s)

	var serverCaps control

	if serverCaps, _, err = clientHandshake(
		s,
		client.keys(),
		client.listType(),
		client.PinnedPublicKey,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	var want control

	if want, err = signWant(client.keys(), serverCaps.Nonce, control{
		Direction:    DirectionPush,
		Query:        query,
		ExcludeBlobs: options.ExcludeBlobs,
	}); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = s.writeControl(TypeWant, want); err != nil {
		err = errors.Wrap(err)
		return err
	}

	if err = sendClosure(
		client.env,
		s,
		client.local,
		client.local.GetEnvRepo(),
		want,
		negotiateCompression(serverCaps.Compression),
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}
