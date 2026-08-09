package commands_dodder

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"code.linenisgreat.com/dodder/go/internal/0/hyphence"
	"code.linenisgreat.com/dodder/go/internal/bravo/env_ui"
	"code.linenisgreat.com/dodder/go/internal/bravo/registry"
	"code.linenisgreat.com/dodder/go/internal/charlie/genesis_configs"
	"code.linenisgreat.com/dodder/go/internal/charlie/repo_config_cli"
	"code.linenisgreat.com/dodder/go/internal/delta/command"
	"code.linenisgreat.com/dodder/go/lib/0/primordial"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/errors"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/interfaces"
	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/tridex"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"golang.org/x/term"
)

func init() {
	utility.AddCmd("repos-list", &ReposList{})
}

// ReposList renders the host-wide registry view: every repo registered in
// the per-host index (RFC-0007 registry v1; the dodder twin of `madder
// list -all`) unioned with the repos discoverable from the current
// scope(s). Advisory only — the index is never consulted for repo
// resolution (FDR-0019); it just answers "what repos exist on this host".
type ReposList struct {
	Format string
}

var (
	_ interfaces.CommandComponentWriter = (*ReposList)(nil)
	_ command.CommandWithArgs           = (*ReposList)(nil)
)

func (cmd ReposList) GetDescription() command.Description {
	return command.Description{
		Short: "list dodder repos host-wide from the per-host registry",
		Long: "Union of the per-host registry index at " +
			"$XDG_STATE_HOME/dodder/index (every repo genesis has registered, " +
			"any scope, any cwd) and the repos addressable from the current " +
			"scope(s). A registered repo whose directory no longer exists is " +
			"marked (stale); registry-gc prunes such entries. The index is " +
			"advisory: it feeds this listing only, never repo resolution.",
	}
}

func (cmd *ReposList) GetArgs() []command.ArgGroup { return nil }

func (cmd *ReposList) SetFlagDefinitions(
	flagSet interfaces.CLIFlagDefinitions,
) {
	flagSet.StringVar(
		&cmd.Format,
		"format",
		"",
		"output format: text, ndjson, or json (default: styled table on a "+
			"TTY, ndjson otherwise)",
	)
}

func (cmd ReposList) Run(req command.Request) {
	req.AssertNoMoreArgs()

	config := repo_config_cli.FromAny(req.Utility.GetConfigAny())
	envUI := env_ui.Make(req, config, config.Debug, env_ui.Options{})

	live, err := listScopedRepos(req)
	if err != nil {
		envUI.Cancel(err)
		return
	}

	// Best-effort: a missing or unreadable index yields no registered
	// entries, leaving just the current-scope repos — never an error for a
	// listing.
	entries, _ := registry.Entries()

	rows := assembleRepoRows(live, entries)

	home, _ := os.UserHomeDir()

	switch strings.ToLower(cmd.Format) {
	case "", "auto":
		// auto + TTY renders the styled table; piped auto takes the
		// structured path — mirroring `madder list -all`'s dispatch.
		if primordial.IsTty(os.Stdout) {
			envUI.GetUI().Printf(
				"%s",
				renderReposTable(rows, home, reposTerminalWidth()),
			)
		} else if err := emitReposNDJSON(rows, home); err != nil {
			envUI.Cancel(err)
		}

	case "text":
		emitReposText(envUI, rows, home)

	case "ndjson":
		if err := emitReposNDJSON(rows, home); err != nil {
			envUI.Cancel(err)
		}

	case "json":
		if err := emitReposJSON(rows, home); err != nil {
			envUI.Cancel(err)
		}

	default:
		errors.ContextCancelWithBadRequestf(
			req,
			"unsupported -format %q (want text, ndjson, or json)",
			cmd.Format,
		)
	}
}

// repoRow is one fully-assembled row of `dodder repos-list`. pubkey and id
// hold the FULL markl-id strings (empty when the config-seed is missing,
// undecodable, or predates the field); the table renderer abbreviates them
// by shortest-distinct-prefix, while text/json emit them whole for
// scriptability.
type repoRow struct {
	name        string
	pubkey      string
	id          string
	locationDir string // the repo's directory (raw; renderers format it)
	stale       bool
	sortKey     string // registry key / dir hash, for a stable tiebreak
}

// assembleRepoRows unions the current scope's live repos with the
// registered index entries, deduping by registry key AND cleaned
// config-seed path (belt-and-suspenders against absolute/relative skew
// between the dir hashed at genesis and the one discovered at list time),
// so a repo present in both appears once. Registered-only entries are
// decoded from disk; dangling ones become stale rows.
func assembleRepoRows(
	live []scopedRepo,
	entries []registry.Entry,
) []repoRow {
	seenKey := make(map[string]bool)
	seenConfig := make(map[string]bool)
	var rows []repoRow

	for _, repo := range live {
		dir := repo.Dir
		if abs, err := filepath.Abs(dir); err == nil {
			dir = abs
		}

		configSeed := filepath.Join(dir, "config-seed")
		key := registry.Key(dir)
		seenKey[key] = true
		seenConfig[filepath.Clean(configSeed)] = true

		row := repoRow{
			name:        repo.Spelling(),
			locationDir: dir,
			sortKey:     key,
		}
		row.pubkey, row.id = decodeConfigSeedIdentity(configSeed)
		rows = append(rows, row)
	}

	for _, e := range entries {
		if seenKey[e.Key] || seenConfig[filepath.Clean(e.Target)] {
			continue
		}

		row := repoRow{
			name:        inferRepoSpelling(e.RepoDir()),
			locationDir: e.RepoDir(),
			sortKey:     e.Key,
		}

		if e.Dangling {
			row.stale = true
		} else {
			row.pubkey, row.id = decodeConfigSeedIdentity(e.Target)
		}

		rows = append(rows, row)
	}

	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].name != rows[j].name {
			return rows[i].name < rows[j].name
		}
		return rows[i].sortKey < rows[j].sortKey
	})

	return rows
}

// decodeConfigSeedIdentity reads a repo's identity off its config-seed:
// the repo public key and (for V3+ configs, RFC-0007) the uuidv7 instance
// id. Best-effort: a missing, corrupt, or legacy-encoded config-seed (e.g.
// the combined-HRP key form) yields blanks rather than failing the
// listing — the row still shows NAME and LOCATION.
func decodeConfigSeedIdentity(
	configSeedPath string,
) (pubkey, instanceId string) {
	typed, err := hyphence.DecodeFromFile(
		genesis_configs.CoderPrivate,
		configSeedPath,
	)
	if err != nil || typed.Blob == nil {
		return pubkey, instanceId
	}

	if pk := typed.Blob.GetPublicKey(); pk != nil && len(pk.GetBytes()) > 0 {
		pubkey = pk.StringWithFormat()
	}

	if withId, ok := typed.Blob.(genesis_configs.ConfigInstanceId); ok &&
		!withId.GetInstanceId().IsNull() {
		instanceId = withId.GetInstanceId().StringWithFormat()
	}

	return pubkey, instanceId
}

// inferRepoSpelling reconstructs a repo's -repo_id spelling from its
// directory path. Best-effort: host-wide there is no marker that
// distinguishes every cwd root, so the `.` prefix is inferred from the
// `.dodder/` path segment and falls back to the bare leaf name. LOCATION
// always carries the exact path, so a missed prefix is cosmetic. Live
// current-scope repos carry their exact spelling (scopedRepo.Spelling)
// and do not go through here.
func inferRepoSpelling(repoDir string) string {
	name := filepath.Base(repoDir)
	sep := string(filepath.Separator)
	if strings.Contains(repoDir, sep+".dodder"+sep) {
		return "." + name
	}
	return name
}

// reposTildeOnly substitutes $HOME with "~" without the aggressive
// per-component shortening the table's path column applies — the plain
// form used by text/json output.
func reposTildeOnly(home, p string) string {
	if home == "" {
		return p
	}
	if p == home {
		return "~"
	}
	if rel, ok := strings.CutPrefix(p, home+string(filepath.Separator)); ok {
		return "~" + string(filepath.Separator) + rel
	}
	return p
}

func reposOrDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// emitReposText prints one tab-separated line per repo: NAME PUBKEY ID
// LOCATION, with a "(stale)" marker appended to the name of a dangling
// entry.
func emitReposText(envUI env_ui.Env, rows []repoRow, home string) {
	for _, r := range rows {
		name := r.name
		if r.stale {
			name += " (stale)"
		}
		envUI.GetUI().Printf(
			"%s\t%s\t%s\t%s",
			name,
			reposOrDash(r.pubkey),
			reposOrDash(r.id),
			reposTildeOnly(home, r.locationDir),
		)
	}
}

// reposListRecord is the structured (ndjson/json) shape. Emitted as a JSON
// array rather than a map keyed by name, because names collide host-wide
// (two cwd-scoped ".default" repos in different directories) — a keyed
// object would clobber one.
type reposListRecord struct {
	Name     string `json:"name"`
	Pubkey   string `json:"pubkey,omitempty"`
	Id       string `json:"id,omitempty"`
	Location string `json:"location"`
	Stale    bool   `json:"stale,omitempty"`
}

func makeReposRecord(r repoRow, home string) reposListRecord {
	return reposListRecord{
		Name:     r.name,
		Pubkey:   r.pubkey,
		Id:       r.id,
		Location: reposTildeOnly(home, r.locationDir),
		Stale:    r.stale,
	}
}

func emitReposNDJSON(rows []repoRow, home string) (err error) {
	buf := bufio.NewWriter(os.Stdout)
	defer errors.DeferredFlusher(&err, buf)

	enc := json.NewEncoder(buf)
	for _, r := range rows {
		_ = enc.Encode(makeReposRecord(r, home))
	}
	return nil
}

func emitReposJSON(rows []repoRow, home string) (err error) {
	out := make([]reposListRecord, 0, len(rows))
	for _, r := range rows {
		out = append(out, makeReposRecord(r, home))
	}

	buf := bufio.NewWriter(os.Stdout)
	defer errors.DeferredFlusher(&err, buf)

	enc := json.NewEncoder(buf)
	enc.SetIndent("", "  ")
	_ = enc.Encode(out)
	return nil
}

//  _____     _     _
// |_   _|_ _| |__ | | ___
//   | |/ _` | '_ \| |/ _ \
//   | | (_| | |_) | |  __/
//   |_|\__,_|_.__/|_|\___|
//
// Styled TTY rendering, mirroring `madder list -all`'s lipgloss table
// (madder go/internal/india/commands/list_table.go — the clown/posh/
// spinclass table idiom). Styles are local to this file since dodder has
// no other lipgloss table surface yet.

var reposSubtleColor = lipgloss.AdaptiveColor{Light: "240", Dark: "245"}

var (
	reposHeaderStyle = lipgloss.NewStyle().Bold(true).Padding(0, 1)
	reposCellStyle   = lipgloss.NewStyle().Padding(0, 1)
	reposBorderStyle = lipgloss.NewStyle().Foreground(reposSubtleColor)
	reposIdStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("4"))
	reposGreyStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("59"))
	reposStaleStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
)

// reposLocationColumn is the 0-based index of LOCATION in the header order
// (NAME · PUBKEY · ID · LOCATION). It is the only unpinned column, so it
// is the one lipgloss reflows to fill the table width.
const reposLocationColumn = 3

// reposTerminalWidth reports stdout's column count, or 0 when stdout is
// not a sized terminal (renderReposTable treats 0 as content-sized).
func reposTerminalWidth() int {
	w, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w <= 0 {
		return 0
	}
	return w
}

func reposFixedColumnWidths(rendered [][4]string) [4]int {
	w := [4]int{
		lipgloss.Width("NAME"),
		lipgloss.Width("PUBKEY"),
		lipgloss.Width("ID"),
		0,
	}
	for _, r := range rendered {
		for c := 0; c < reposLocationColumn; c++ {
			w[c] = max(w[c], lipgloss.Width(r[c]))
		}
	}
	return w
}

func reposTableStyleFunc(width int, fixed [4]int) table.StyleFunc {
	return func(row, col int) lipgloss.Style {
		if row == table.HeaderRow {
			return reposHeaderStyle
		}
		if width > 0 && col != reposLocationColumn {
			return reposCellStyle.Width(
				fixed[col] + reposCellStyle.GetHorizontalPadding(),
			)
		}
		return reposCellStyle
	}
}

// abbreviateRepoPath shortens p the way fish's prompt_pwd does: "~" for
// home, every intermediate component to its first character, the leaf kept
// whole. style may be nil (no styling applied to abbreviated components).
func abbreviateRepoPath(home, p string, style *lipgloss.Style) string {
	if home != "" {
		if p == home {
			p = "~"
		} else if rel, ok := strings.CutPrefix(p, home+string(filepath.Separator)); ok {
			p = "~" + string(filepath.Separator) + rel
		}
	}

	parts := strings.Split(p, string(filepath.Separator))
	for i := 0; i < len(parts)-1; i++ {
		part := parts[i]
		if part == "" || part == "~" {
			continue
		}
		abbreviated := string([]rune(part)[:1])
		if style != nil {
			abbreviated = style.Render(abbreviated)
		}
		parts[i] = abbreviated
	}

	return strings.Join(parts, string(filepath.Separator))
}

// renderReposTable renders the styled `dodder repos-list` table. pubkey/id
// are abbreviated by shortest-distinct-prefix (tridex) across the shown
// set, so each abbreviation stays unique against its siblings.
func renderReposTable(rows []repoRow, home string, width int) string {
	if len(rows) == 0 {
		return reposBorderStyle.Render("No repos registered.")
	}

	pubkeys := make([]string, 0, len(rows))
	ids := make([]string, 0, len(rows))
	for _, r := range rows {
		if r.pubkey != "" {
			pubkeys = append(pubkeys, r.pubkey)
		}
		if r.id != "" {
			ids = append(ids, r.id)
		}
	}
	pubkeyAbbr := tridex.Make(pubkeys...)
	idAbbr := tridex.Make(ids...)

	rendered := make([][4]string, 0, len(rows))
	for _, r := range rows {
		name := reposIdStyle.Render(r.name)
		if r.stale {
			name += " " + reposStaleStyle.Render("(stale)")
		}

		pubkey := "—"
		if r.pubkey != "" {
			pubkey = reposGreyStyle.Italic(true).Render(
				pubkeyAbbr.Abbreviate(r.pubkey),
			)
		}
		id := "—"
		if r.id != "" {
			id = reposGreyStyle.Italic(true).Render(idAbbr.Abbreviate(r.id))
		}
		loc := abbreviateRepoPath(home, r.locationDir, &reposGreyStyle)

		rendered = append(rendered, [4]string{name, pubkey, id, loc})
	}

	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(reposBorderStyle).
		Headers("NAME", "PUBKEY", "ID", "LOCATION").
		StyleFunc(reposTableStyleFunc(width, reposFixedColumnWidths(rendered)))
	if width > 0 {
		t = t.Width(width)
	}

	for _, r := range rendered {
		t.Row(r[0], r[1], r[2], r[3])
	}

	return t.Render()
}
