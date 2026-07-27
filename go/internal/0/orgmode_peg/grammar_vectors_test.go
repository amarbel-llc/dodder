//go:build test

package orgmode_peg

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"code.linenisgreat.com/purse-first/libs/dewey/pkgs/ui"
)

// dodder#378: langlang grammar-validation gate. Mirrors piggy's
// grammar_vectors_test.go pattern (piggy/go/internal/charlie/
// markl_registrations/grammar_vectors_test.go) -- the reference
// implementation for this pattern across amarbel-llc repos.
//
// Deliberately avoids declaring *ui.T as an explicit parameter type on
// any standalone helper: ui.MakeT's actual return type is an internal
// package type from a different module this package cannot name
// directly, so a helper with an explicit *ui.T parameter fails to
// compile even though calling methods on the value works fine inside
// the function that created it (dodder#374(b)'s three_way_test.go hit
// the identical issue -- see this repo's task list, "Fix ui.T so it
// works as a cross-function test assertion anchor"). Helpers below take
// and return plain values instead; ui.T-specific calls (AssertNoError,
// Skip, Run) stay inline in TestGrammarVectors and its subtest
// closures, where ui.T's real type is already in scope.

// langlangStartRule is orgmode.peg's own top-level rule (orgmode.peg:8,
// `OrgFile <- Preamble Heading* !.`) -- the production name langlang's
// generated main.go names its entry point after, and the string this
// harness matches langlang's stdout against to confirm a vector parsed
// under the CORRECT production, not a permissive catch-all.
const langlangStartRule = "OrgFile"

// langlangFailurePattern matches langlang's own "<file>:<line>:<col>: "
// error-line prefix. Required because langlang#26: `-input` mode always
// exits 0, even on a parse failure -- the exit code cannot be trusted,
// so pass/fail is decided by scraping stdout instead.
var langlangFailurePattern = regexp.MustCompile(`^\S+:\d+:\d+: `)

// ansiSGRPattern strips ANSI color escapes langlang may emit to a
// non-tty stdout under some terminal-detection edge cases.
var ansiSGRPattern = regexp.MustCompile("\x1b\\[[0-9;]*m")

func resolveLanglangBin() (bin string, err error) {
	if bin = os.Getenv("LANGLANG_BIN"); bin != "" {
		return bin, nil
	}

	return exec.LookPath("langlang")
}

func parsedSuccessfully(trimmed string, cmdErr error) bool {
	return cmdErr == nil &&
		strings.HasPrefix(trimmed, langlangStartRule) &&
		!langlangFailurePattern.MatchString(trimmed)
}

func runLanglangInput(
	bin, grammarPeg, tmpDir, content string,
) (trimmed string, cmdErr error) {
	tmp, err := os.CreateTemp(tmpDir, "grammar-vector-*.org")
	if err != nil {
		return "", err
	}

	defer os.Remove(tmp.Name())

	if _, err = tmp.WriteString(content); err != nil {
		_ = tmp.Close()
		return "", err
	}

	if err = tmp.Close(); err != nil {
		return "", err
	}

	cmd := exec.Command(
		bin,
		"-grammar", grammarPeg,
		"-input", tmp.Name(),
		"-disable-builtins",
		"-disable-spaces",
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmdErr = cmd.Run()

	trimmed = strings.TrimSpace(ansiSGRPattern.ReplaceAllString(stdout.String(), ""))

	return trimmed, cmdErr
}

// sampleOrgVector reuses haustoria_orgmode's own real fixture
// (lima/haustoria_orgmode/headings_test.go's sampleOrg) rather than
// inventing a new one -- it already exercises headings, a TODO
// keyword, a tag suffix, a PROPERTIES drawer, and a SCHEDULED planning
// line, i.e. every non-trivial production orgmode.peg defines.
const sampleOrgVector = `* Nuisance distractions
:PROPERTIES:
:ID:       e794126b-b510-45af-a233-ee4c9e4879f1
:END:

One of my biggest focus inhibitors is frustration.

* Spinclass :work:health:
- explore session state
- explore session ids

* TODO Today important
SCHEDULED: <2025-08-25 Mon>
:PROPERTIES:
:ID:       05a231ff-d627-486f-9373-3b98b9f83878
:END:

body content here
`

func TestGrammarVectors(t1 *testing.T) {
	t := ui.MakeT(t1)

	bin, err := resolveLanglangBin()
	if err != nil {
		t1.Skip("langlang binary not available (set LANGLANG_BIN or run via `just test-grammar-vectors`)")
		return
	}

	grammarPeg, err := filepath.Abs("orgmode.peg")
	t.AssertNoError(err)

	tmpDir := t1.TempDir()

	// Both vectors are positive (must parse). orgmode.peg is
	// deliberately permissive by design (its own header comment:
	// "Subheadings...and inline markup are preserved as opaque body
	// text") -- Drawer is optional (Heading's `Drawer? Body`) and Body
	// swallows any line not matching HeadingStart, so a well-formed
	// UTF-8 negative vector proved surprisingly hard to construct
	// (confirmed empirically: an unterminated :PROPERTIES: drawer
	// still parses fine, just reinterpreted as opaque body text, not a
	// failure). That this harness actually detects a genuine parse
	// failure is verified separately, against the grammar file itself
	// (dodder#378 verification step), not baked into this vector set.
	vectors := []struct {
		name        string
		content     string
		wantSuccess bool
	}{
		{"single bare heading", "* just a title\n", true},
		{"real fixture (haustoria_orgmode sampleOrg)", sampleOrgVector, true},
	}

	for _, vector := range vectors {
		t.Run(ui.MakeTestCaseInfo(vector.name), func(t *ui.T) {
			trimmed, cmdErr := runLanglangInput(bin, grammarPeg, tmpDir, vector.content)
			got := parsedSuccessfully(trimmed, cmdErr)

			if got != vector.wantSuccess {
				t.Errorf(
					"%s: want success=%t, got success=%t\nlanglang output:\n%s",
					vector.name, vector.wantSuccess, got, trimmed,
				)
			}
		})
	}
}
