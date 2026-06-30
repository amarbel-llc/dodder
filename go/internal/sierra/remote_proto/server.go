package remote_proto

import (
	"io"

	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/romeo/local_working_copy"
	mad_domain_interfaces "github.com/amarbel-llc/madder/go/pkgs/domain_interfaces"
	env_local "github.com/amarbel-llc/madder/go/pkgs/env_local"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
)

// KeySource provides the keypair the session handshake uses to attest the
// server (and verify the client). Lifting key access off the concrete repo
// lets tests inject a generated keypair, exactly as remote_http does.
type KeySource interface {
	GetPublicKey() mad_domain_interfaces.MarklId
	GetPrivateKey() mad_domain_interfaces.MarklId
}

// Server runs the receive/upload half of the drtp protocol over accepted
// connections. It is the hard-fork counterpart of remote_http.Server; the
// two coexist.
type Server struct {
	EnvLocal env_local.Env
	Repo     *local_working_copy.Repo

	// Public waives the client's attestation for fetches (read-only),
	// mirroring remote_http.Server.Public. A push always requires client
	// attestation regardless.
	Public bool

	// KeySource overrides the Repo-backed keys; nil in production.
	KeySource KeySource
}

func (server *Server) keys() keys {
	if server.KeySource != nil {
		return keys{
			private: server.KeySource.GetPrivateKey(),
			public:  server.KeySource.GetPublicKey(),
		}
	}

	return keys{
		private: server.Repo.GetImmutableConfigPrivate().Blob.GetPrivateKey(),
		public:  server.Repo.GetImmutableConfigPublic().GetPublicKey(),
	}
}

func (server *Server) selfCaps() selfCaps {
	config := server.Repo.GetImmutableConfigPublic()

	return selfCaps{
		listType:     config.GetInventoryListTypeId(),
		repoId:       config.GetRepoId().String(),
		storeVersion: config.GetStoreVersion().String(),
	}
}

// configDescriptor builds the RFC-0005 drtp-config-v1 descriptor naming
// the serving repo's current config-log head, for a fetch to offer config
// seeding. It returns nil when the config log has no head blob (an empty
// or genesis-default-only log offers nothing to seed), so the fetch
// behaves exactly as before — no frame, no folded blob.
func (server *Server) configDescriptor() *control {
	configSku := server.Repo.GetConfigStore().GetConfig().GetSku()

	blobDigest := configSku.GetBlobDigest()

	if blobDigest == nil || blobDigest.IsNull() {
		return nil
	}

	return &control{
		BlobId:     blobDigest.String(),
		ConfigType: configSku.GetType().String(),
		ConfigTai:  configSku.GetTai().String(),
	}
}

// ServeConn runs a single drtp session over conn. The caller owns conn's
// lifetime; ServeConn does not close it (the transport layer that produced
// conn does).
func (server *Server) ServeConn(conn io.ReadWriteCloser) (err error) {
	if err = server.serveSession(makeSession(conn)); err != nil {
		err = errors.Wrap(err)
		return err
	}

	return err
}

func (server *Server) serveSession(s *session) (err error) {
	env := server.Repo.GetEnv()

	var clientCaps control
	var want control

	if clientCaps, want, err = serverHandshake(
		s,
		server.keys(),
		server.selfCaps(),
		server.Public,
	); err != nil {
		err = errors.Wrap(err)
		return err
	}

	switch want.Direction {
	case DirectionFetch, "":
		// The client fetches: the server is the sender. It computes the
		// closure from its own store and streams it, compressing blobs when
		// the client advertised support.
		if err = sendClosure(
			env,
			s,
			server.Repo,
			server.Repo.GetEnvRepo().GetReadBlobStore(),
			server.Repo.GetEnvRepo(),
			want,
			negotiateCompression(clientCaps.Compression),
			server.configDescriptor(),
			true, // ship full object history for the receiver's merge negotiator (Option B, #299)
		); err != nil {
			s.writeError(err.Error(), 500)
			err = errors.Wrap(err)
			return err
		}

	case DirectionPush:
		// The client pushes: the server is the receiver. Config is never
		// pushed (RFC 0005), so no descriptor is captured.
		if err = receiveClosure(
			env,
			s,
			server.Repo,
			want,
			sku.GetStoreOptionsRemoteTransfer(),
			nil,
		); err != nil {
			s.writeError(err.Error(), 500)
			err = errors.Wrap(err)
			return err
		}

	default:
		err = errors.Errorf("unknown transfer direction %q", want.Direction)
		s.writeError(err.Error(), 400)
		return err
	}

	return err
}
