package remote_proto

import (
	"io"

	"code.linenisgreat.com/dodder/go/internal/bravo/env_ui"
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/papa/repo"
	"code.linenisgreat.com/dodder/go/internal/romeo/local_working_copy"
	mad_domain_interfaces "code.linenisgreat.com/madder/go/pkgs/domain_interfaces"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
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

func (client *Client) selfCaps() selfCaps {
	config := client.local.GetImmutableConfigPublic()

	return selfCaps{
		listType:     config.GetInventoryListTypeId(),
		repoId:       config.GetRepoId().String(),
		storeVersion: config.GetStoreVersion().String(),
	}
}

// ConfigDescriptor is the RFC-0005 config-log head a fetch server may
// offer (its blob-id, the config blob's own type, and an informative
// source timestamp). Fetch surfaces it so a clone can seed its config log;
// it is the empty zero value (BlobId == "") when the server offered none.
type ConfigDescriptor struct {
	BlobId     string
	ConfigType string
	Tai        string
}

// Fetch pulls every object matching query (and its expand-edges closure)
// from the peer on conn into the local repo. Closes conn when done. It
// returns the source's config descriptor when the server offered one (RFC
// 0005); a clone uses it to seed config, a pull ignores it.
func (client *Client) Fetch(
	conn io.ReadWriteCloser,
	query string,
	options repo.ImporterOptions,
) (descriptor ConfigDescriptor, err error) {
	s := makeSession(conn)
	defer errors.DeferredCloser(&err, s)

	var serverCaps control

	if serverCaps, _, err = clientHandshake(
		s,
		client.keys(),
		client.selfCaps(),
		client.PinnedPublicKey,
	); err != nil {
		err = errors.Wrap(err)
		return descriptor, err
	}

	var want control

	if want, err = signWant(client.keys(), serverCaps.Nonce, control{
		Direction:           DirectionFetch,
		Query:               query,
		AllowMergeConflicts: options.AllowMergeConflicts,
		ExcludeBlobs:        options.ExcludeBlobs,
	}); err != nil {
		err = errors.Wrap(err)
		return descriptor, err
	}

	if err = s.writeControl(TypeWant, want); err != nil {
		err = errors.Wrap(err)
		return descriptor, err
	}

	var configFrame control

	if err = receiveClosure(
		client.env,
		s,
		client.local,
		want,
		sku.GetStoreOptionsImport(),
		&configFrame,
	); err != nil {
		err = errors.Wrap(err)
		return descriptor, err
	}

	descriptor = ConfigDescriptor{
		BlobId:     configFrame.BlobId,
		ConfigType: configFrame.ConfigType,
		Tai:        configFrame.ConfigTai,
	}

	return descriptor, err
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
		client.selfCaps(),
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
		client.local.GetEnvRepo().GetReadBlobStore(),
		client.local.GetEnvRepo(),
		want,
		negotiateCompression(serverCaps.Compression),
		nil, // config is never pushed (RFC 0005)
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}
