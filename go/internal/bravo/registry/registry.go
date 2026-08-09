// Package registry maintains a per-host, best-effort index of every dodder
// repo created on this machine, so `dodder repos-list` can enumerate repos
// host-wide regardless of which XDG scope or cwd created them.
//
// This is the dodder twin of madder's go/internal/bravo/registry (registry
// v1, RFC-0007 addendum): the same primitive, parameterized only by the
// Utility segment. Both copies are deliberately stdlib-only and
// utility-agnostic — they know only paths, symlinks, and a utility-name
// segment — so they stay cheap candidates for extraction into dewey; until
// that extraction, keep the two copies synchronized.
//
// The mechanism is lifted from spinclass's session index: one directory
// under $XDG_STATE_HOME, one entry per repo keyed by a stable hash of the
// repo's absolute directory, each entry a symlink pointing at that repo's
// config-seed. Writes are TOCTOU-safe (symlink-to-temp then rename). A
// dangling symlink — the repo was deleted or moved out from under the
// index — is classified as stale rather than treated as an error, and
// pruned by GC after a retention window.
//
// It is ADVISORY ONLY: nothing here participates in repo resolution (that
// correctness belongs to FDR-0019's scope rules). The index feeds listing
// surfaces exclusively.
//
// Sandbox safety: IndexDir addresses $XDG_STATE_HOME as an absolute path
// with no walk-up and no ceiling involvement, so a test (or bats lane) that
// redirects the XDG environment gets a fully isolated index for free — the
// composition risk RFC-0007 flagged dissolves by construction.
package registry

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Utility is the $XDG_STATE_HOME sub-directory the index lives under
// ($XDG_STATE_HOME/<Utility>/index). A package var, not a const, so a future
// dewey extraction can parameterize it per-consumer; dodder never rebinds it.
var Utility = "dodder"

// DefaultRetention mirrors spinclass's tombstone-retention default: a dangling
// index entry younger than this is kept, since a repo may be transiently
// unavailable (an unmounted filesystem) rather than truly gone.
func DefaultRetention() time.Duration {
	return 30 * 24 * time.Hour
}

// xdgStateBase returns $XDG_STATE_HOME or its XDG-default fallback. Read
// directly from the environment — never via env_dir's scope machinery — so
// the index never inherits the walk-up/ceiling ambiguity FDR-0019 exists to
// resolve, and so a sandbox that redirects XDG_STATE_HOME is isolated by
// construction.
func xdgStateBase() string {
	if v := os.Getenv("XDG_STATE_HOME"); v != "" {
		return v
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "state")
}

// IndexDir is the absolute directory holding index entries.
func IndexDir() string {
	return filepath.Join(xdgStateBase(), Utility, "index")
}

// Key derives an index entry's filename from a repo's absolute directory (the
// directory that contains its config-seed). It is sha256(Clean(baseAbsPath))
// truncated to 8 bytes, lowercase hex — 16 characters, no extension. Stable
// across invocations and collision-free in practice, so any code path can
// recompute the same key for the same repo (e.g. to dedup a live
// current-scope repo against its own index entry).
func Key(baseAbsPath string) string {
	h := sha256.Sum256([]byte(filepath.Clean(baseAbsPath)))
	return fmt.Sprintf("%x", h[:8])
}

// Register records a repo in the per-host index: a symlink at
// IndexDir()/Key(baseAbsPath) pointing at configAbsPath (the repo's
// config-seed). Re-registering an existing repo atomically replaces the
// entry. Best-effort by contract — callers warn on error and never fail the
// enclosing genesis.
//
// The write is TOCTOU-safe: os.Symlink to a unique pid+nano temp name in the
// index dir, then rename over the final key. pid+nano gives a
// unique-by-construction temp with no intermediary regular file, avoiding the
// create/remove/symlink window a CreateTemp pattern would expose.
func Register(baseAbsPath, configAbsPath string) error {
	dir := IndexDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	link := filepath.Join(dir, Key(baseAbsPath))
	tmp := filepath.Join(
		dir,
		fmt.Sprintf(".tmp-%d-%d", os.Getpid(), time.Now().UnixNano()),
	)

	if err := os.Symlink(configAbsPath, tmp); err != nil {
		return err
	}
	if err := os.Rename(tmp, link); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// Entry is one index entry.
type Entry struct {
	// Key is the entry's filename (the truncated-hash key).
	Key string
	// Link is the absolute path of the index symlink itself.
	Link string
	// Target is the symlink's target — the repo's config-seed path.
	// Recovered even for dangling entries (Readlink succeeds; the target
	// simply no longer exists on disk).
	Target string
	// Dangling is true when Target does not resolve (the repo was deleted or
	// moved), marking the entry stale.
	Dangling bool
}

// RepoDir returns the repo's directory — the parent of the config-seed
// target — recovered from the entry. Empty when Target is empty.
func (e Entry) RepoDir() string {
	if e.Target == "" {
		return ""
	}
	return filepath.Dir(e.Target)
}

// Entries returns every index entry, live and dangling, sorted by key. A
// missing index dir is not an error (nothing has registered yet); it returns
// nil. Non-symlink entries and the transient .tmp-* staging files are skipped
// defensively.
func Entries() ([]Entry, error) {
	dir := IndexDir()
	dirEntries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var out []Entry
	for _, de := range dirEntries {
		name := de.Name()
		if len(name) > 0 && name[0] == '.' {
			continue // .tmp-* staging files (and any dotfile) are not entries
		}

		link := filepath.Join(dir, name)
		info, lerr := os.Lstat(link)
		if lerr != nil || info.Mode()&os.ModeSymlink == 0 {
			continue // not a symlink — not one of ours
		}

		target, rerr := os.Readlink(link)
		if rerr != nil {
			continue
		}

		// os.Stat follows the symlink; ErrNotExist means the target is gone.
		_, statErr := os.Stat(link)
		out = append(out, Entry{
			Key:      name,
			Link:     link,
			Target:   target,
			Dangling: errors.Is(statErr, os.ErrNotExist),
		})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out, nil
}

// shouldPrune reports whether a dangling entry registered at mtime should be
// pruned given retention and the current time. Extracted from GC as a pure
// function so the retention boundary can be unit-tested without manipulating
// symlink timestamps (bats exercises the on-disk aging path via `touch -h`).
func shouldPrune(dangling bool, mtime, now time.Time, retention time.Duration) bool {
	if retention <= 0 || !dangling {
		return false
	}
	return mtime.Before(now.Add(-retention))
}

// GC removes dangling index entries whose symlink is older than retention
// (measured from the symlink's own mtime, i.e. registration time). Live
// entries are always kept. retention <= 0 is a no-op returning 0, matching
// spinclass's GCTombstones contract. Returns the number of entries removed.
//
// v1 keeps no tombstone: a deleted repo needs no post-mortem record (unlike a
// closed session), so a pruned entry leaves nothing behind. The retention
// window is a grace period against transient unavailability, not a history
// buffer.
func GC(retention time.Duration) (int, error) {
	if retention <= 0 {
		return 0, nil
	}

	entries, err := Entries()
	if err != nil {
		return 0, err
	}

	now := time.Now()
	removed := 0
	for _, e := range entries {
		if !e.Dangling {
			continue
		}
		info, lerr := os.Lstat(e.Link)
		if lerr != nil {
			continue
		}
		if !shouldPrune(e.Dangling, info.ModTime(), now, retention) {
			continue
		}
		if os.Remove(e.Link) == nil {
			removed++
		}
	}
	return removed, nil
}
