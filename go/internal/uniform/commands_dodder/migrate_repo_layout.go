package commands_dodder

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/errors"
	"github.com/amarbel-llc/purse-first/libs/dewey/pkgs/interfaces"
)

func init() {
	// Implements the "explicit migration (`dodder migrate-repo-layout`,
	// exact name TBD)" described but never built in FDR-0019 (Scoped Repo
	// Resolution): a legacy flat repo tree (pre-repos/<name>/ nesting) is
	// unreadable by the current binary, with no fallback (confirmed by
	// direct testing -- see dodder#360-adjacent session notes and
	// docs/features/0019-scoped-repo-resolution.md's "Legacy
	// compatibility" section, which promised a read-in-place fallback
	// that was never implemented either).
	//
	// This command deliberately does ONLY the directory restructuring --
	// no dodder object-model parsing, no repo-opening, pure filesystem
	// copy. It NEVER writes to -source; it only reads from it and writes
	// a new tree at -dest. Verify the result with a real `dodder`
	// invocation against -dest afterward (this command does not do that
	// itself, by design, to keep its own blast radius minimal).
	utility.AddCmd("migrate-repo-layout", &MigrateRepoLayout{})
}

// repoLayoutCategories are the XDG-category roots that nest under
// repos/<name>/ per FDR-0019's on-disk layout table. The madder blob slot
// (.madder/) is deliberately excluded -- FDR-0019: "the madder blob env
// stays flat" -- and this command only ever touches -source's .dodder-shaped
// tree, never a sibling .madder/.
var repoLayoutCategories = []string{
	"cache",
	"config",
	filepath.Join("local", "runtime"),
	filepath.Join("local", "share"),
	filepath.Join("local", "state"),
}

type MigrateRepoLayout struct {
	Source string
	Dest   string
	Name   string
}

var (
	_ interfaces.CommandComponentWriter = (*MigrateRepoLayout)(nil)
	_ command.CommandWithArgs           = (*MigrateRepoLayout)(nil)
)

func (cmd MigrateRepoLayout) GetDescription() command.Description {
	return command.Description{
		Short: "copy a legacy flat .dodder tree into the repos/<name>/ nested layout (FDR-0019); never modifies -source",
	}
}

func (cmd *MigrateRepoLayout) GetArgs() []command.ArgGroup { return nil }

func (cmd *MigrateRepoLayout) SetFlagDefinitions(f interfaces.CLIFlagDefinitions) {
	f.StringVar(&cmd.Source, "source", "", "path to the legacy flat .dodder directory (read-only, never modified)")
	f.StringVar(&cmd.Dest, "dest", "", "path to write the new, nested .dodder directory (must not already exist)")
	f.StringVar(&cmd.Name, "name", "default", "repo name to nest under (matches FDR-0019: legacy trees become the repo named \"default\" in their scope)")
}

func (cmd MigrateRepoLayout) Run(req command.Request) {
	if cmd.Source == "" || cmd.Dest == "" {
		errors.ContextCancelWithErrorf(req, "-source and -dest are both required")
		return
	}

	if info, err := os.Stat(cmd.Source); err != nil {
		errors.ContextCancelWithErrorf(req, "-source %q: %s", cmd.Source, err)
		return
	} else if !info.IsDir() {
		errors.ContextCancelWithErrorf(req, "-source %q is not a directory", cmd.Source)
		return
	}

	if _, err := os.Stat(cmd.Dest); err == nil {
		errors.ContextCancelWithErrorf(req, "-dest %q already exists; refusing to write into an existing tree", cmd.Dest)
		return
	} else if !os.IsNotExist(err) {
		errors.ContextCancelWithErrorf(req, "-dest %q: %s", cmd.Dest, err)
		return
	}

	for _, category := range repoLayoutCategories {
		srcCategoryDir := filepath.Join(cmd.Source, category)

		info, err := os.Stat(srcCategoryDir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}

			errors.ContextCancelWithErrorf(req, "stat %q: %s", srcCategoryDir, err)
			return
		}

		if !info.IsDir() {
			errors.ContextCancelWithErrorf(req, "%q is not a directory", srcCategoryDir)
			return
		}

		destRepoDir := filepath.Join(cmd.Dest, category, "repos", cmd.Name)

		fileCount, byteCount, err := copyTree(srcCategoryDir, destRepoDir)
		if err != nil {
			errors.ContextCancelWithErrorf(req, "copying %q -> %q: %s", srcCategoryDir, destRepoDir, err)
			return
		}

		fmt.Fprintf(os.Stdout, "%s: copied %d file(s), %d byte(s) -> %s\n", category, fileCount, byteCount, destRepoDir)
	}

	fmt.Fprintf(
		os.Stdout,
		"done. -source untouched. verify -dest with a real dodder invocation (e.g. cd next to -dest and run `dodder show`) before treating -source as disposable.\n",
	)
}

// copyTree recursively copies srcDir's contents into destDir (creating
// destDir and any needed parents). Symlinks are followed (os.Stat, not
// os.Lstat) rather than preserved as links -- acceptable for this
// command's scope (dodder's own on-disk state trees are not expected to
// contain symlinks), but worth knowing if that assumption ever breaks.
func copyTree(srcDir, destDir string) (fileCount int, byteCount int64, err error) {
	err = filepath.Walk(srcDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		rel, relErr := filepath.Rel(srcDir, path)
		if relErr != nil {
			return relErr
		}

		destPath := filepath.Join(destDir, rel)

		if info.IsDir() {
			return os.MkdirAll(destPath, 0o755)
		}

		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			return err
		}

		n, copyErr := copyFile(path, destPath, info.Mode())
		if copyErr != nil {
			return copyErr
		}

		fileCount++
		byteCount += n

		return nil
	})

	return fileCount, byteCount, err
}

func copyFile(srcPath, destPath string, mode os.FileMode) (n int64, err error) {
	src, err := os.Open(srcPath)
	if err != nil {
		return 0, err
	}

	defer src.Close()

	dest, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return 0, err
	}

	defer dest.Close()

	return io.Copy(dest, src)
}
