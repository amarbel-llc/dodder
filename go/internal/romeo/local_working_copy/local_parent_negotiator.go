package local_working_copy

import (
	"code.linenisgreat.com/dodder/go/internal/foxtrot/sku"
	"code.linenisgreat.com/dodder/go/internal/papa/repo"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
)

// mostRecentCommonAncestor returns the highest-TAI version present in both
// histories — the merge base. Versions are matched across repos by TAI, which
// is preserved on transfer and (unlike the content locks EqualerSansTai
// compares) is independent of the repo pubkey, so the same logical version
// still matches after a repo re-signs it under its own key (#298). Returns nil
// when the histories share no version (genuine divergence), which the caller
// treats as "no base" — a real conflict.
func mostRecentCommonAncestor(
	ancestorsLocal, ancestorsRemote []*sku.Transacted,
) (ancestor *sku.Transacted) {
	for _, candidate := range ancestorsLocal {
		isCommon := false

		for _, remote := range ancestorsRemote {
			if candidate.GetTai().Equals(remote.GetTai()) {
				isCommon = true
				break
			}
		}

		if !isCommon {
			continue
		}

		if ancestor == nil || ancestor.GetTai().Less(candidate.GetTai()) {
			ancestor = candidate
		}
	}

	return ancestor
}

// ParentNegotiatorFirstAncestor finds the merge base by reading both repos'
// full object histories directly. Used by the direct/local-override transport
// (both repos local) and the HTTP/stdio transport, where the remote's
// ReadObjectHistory fetches over the wire (remote_http /object-history, #299).
type ParentNegotiatorFirstAncestor struct {
	Local, Remote repo.Repo
}

func (parentNegotiator ParentNegotiatorFirstAncestor) GetParentNegotiator() sku.ParentNegotiator {
	return parentNegotiator
}

func (parentNegotiator ParentNegotiatorFirstAncestor) FindBestCommonAncestor(
	conflicted sku.Conflicted,
) (ancestor *sku.Transacted, err error) {
	var ancestorsLocal, ancestorsRemote []*sku.Transacted

	wg := errors.MakeWaitGroupParallel()

	wg.Do(
		func() (err error) {
			if ancestorsLocal, err = parentNegotiator.Local.ReadObjectHistory(
				conflicted.Local.GetObjectId(),
			); err != nil {
				err = errors.Wrap(err)
				return err
			}

			return err
		},
	)

	wg.Do(
		func() (err error) {
			if ancestorsRemote, err = parentNegotiator.Remote.ReadObjectHistory(
				conflicted.Local.GetObjectId(),
			); err != nil {
				err = errors.Wrap(err)
				return err
			}

			return err
		},
	)

	if err = wg.GetError(); err != nil {
		err = errors.Wrap(err)
		return ancestor, err
	}

	ancestor = mostRecentCommonAncestor(ancestorsLocal, ancestorsRemote)

	return ancestor, err
}

// ParentNegotiatorInBand finds the merge base using the sender's object history
// delivered IN-BAND in the transferred batch, rather than querying the remote.
// The drtp/websocket transport uses this: its session is a lock-step stream
// with no out-of-band history query, so the fetch sender ships each transferred
// object's full history and the receiver builds this negotiator from it (#299).
// The local side is the receiving repo's own history.
type ParentNegotiatorInBand struct {
	local         repo.Repo
	remoteHistory map[string][]*sku.Transacted
}

func MakeParentNegotiatorInBand(
	local repo.Repo,
) *ParentNegotiatorInBand {
	return &ParentNegotiatorInBand{
		local:         local,
		remoteHistory: make(map[string][]*sku.Transacted),
	}
}

// AddRemoteObject records one version from the transferred batch as part of the
// sender's history. The object is cloned because transfer iterators reuse
// pooled values.
func (negotiator *ParentNegotiatorInBand) AddRemoteObject(
	object *sku.Transacted,
) {
	key := object.GetObjectId().String()
	clone, _ := object.CloneTransacted() //repool:owned
	negotiator.remoteHistory[key] = append(negotiator.remoteHistory[key], clone)
}

func (negotiator *ParentNegotiatorInBand) FindBestCommonAncestor(
	conflicted sku.Conflicted,
) (ancestor *sku.Transacted, err error) {
	objectId := conflicted.Local.GetObjectId()

	var ancestorsLocal []*sku.Transacted

	if ancestorsLocal, err = negotiator.local.ReadObjectHistory(
		objectId,
	); err != nil {
		err = errors.Wrap(err)
		return ancestor, err
	}

	ancestor = mostRecentCommonAncestor(
		ancestorsLocal,
		negotiator.remoteHistory[objectId.String()],
	)

	return ancestor, err
}

// ExpandListToObjectHistory returns a new list holding the full version history
// of every distinct object id in the input, read from src. A transfer sender
// uses it so the receiver's in-band negotiator (ParentNegotiatorInBand) gets
// the sender's complete history per object: the transfer is otherwise
// effectively incremental (the query may resolve to latest-only), which would
// leave the receiver unable to tell a fast-forward from a real divergence
// (#299). Shared by both the drtp and HTTP transports.
func ExpandListToObjectHistory(
	src repo.Repo,
	list *sku.HeapTransacted,
) (expanded *sku.HeapTransacted, err error) {
	expanded = sku.MakeListTransacted()
	seen := make(map[string]struct{})

	for object := range list.All() {
		key := object.GetObjectId().String()

		if _, ok := seen[key]; ok {
			continue
		}

		seen[key] = struct{}{}

		var history []*sku.Transacted

		if history, err = src.ReadObjectHistory(
			object.GetObjectId(),
		); err != nil {
			err = errors.Wrap(err)
			return expanded, err
		}

		for _, version := range history {
			expanded.Add(version)
		}
	}

	return expanded, err
}
