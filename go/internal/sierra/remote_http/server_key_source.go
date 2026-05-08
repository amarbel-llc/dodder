package remote_http

import (
	"code.linenisgreat.com/dodder/go/internal/romeo/local_working_copy"
	mad_domain_interfaces "github.com/amarbel-llc/madder/go/pkgs/domain_interfaces"
)

// KeySource provides the keypair the signing middleware uses to
// authenticate requests and sign response trailers. Lifting the key
// access off the concrete *local_working_copy.Repo lets tests
// supply a fake source without standing up a real repo on disk.
type KeySource interface {
	GetPublicKey() mad_domain_interfaces.MarklId
	GetPrivateKey() mad_domain_interfaces.MarklId
}

// repoKeySource adapts *local_working_copy.Repo to KeySource.
type repoKeySource struct {
	repo *local_working_copy.Repo
}

func (source repoKeySource) GetPublicKey() mad_domain_interfaces.MarklId {
	return source.repo.GetImmutableConfigPublic().GetPublicKey()
}

func (source repoKeySource) GetPrivateKey() mad_domain_interfaces.MarklId {
	return source.repo.GetImmutableConfigPrivate().Blob.GetPrivateKey()
}

// keySource returns the configured KeySource, falling back to one
// backed by server.Repo when the field is unset (the default in
// production).
func (server *Server) keySource() KeySource {
	if server.KeySource != nil {
		return server.KeySource
	}
	return repoKeySource{repo: server.Repo}
}
