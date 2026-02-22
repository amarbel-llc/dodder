# Sandcastle --no-tempdir-cleanup Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add `--no-tempdir-cleanup` to sandcastle so bats temp dirs survive sandbox exit, fixing dodder fixture generation.

**Architecture:** Restore tmpdir lifecycle from sandcastle v0.0.37 into v0.0.38 source, add `--no-tempdir-cleanup` flag, thread it through batman's bats wrapper, and update dodder's fixture generator.

**Tech Stack:** TypeScript (sandcastle), Nix (batman wrapper), Bash (dodder fixtures)

---

### Task 1: Restore sandcastle CLI to v0.0.37 feature parity

**Files:**
- Modify: `~/eng/repos/purse-first/packages/sandcastle/src/cli.ts`

**Step 1: Replace cli.ts with v0.0.37 features restored**

The current v0.0.38 source is missing `--shell`, `--tmpdir`, `shellQuote()`,
and the tmpdir lifecycle. Restore them from the compiled v0.0.37 binary at
`/nix/store/031a2jkbdb0rhgznf1w9lhc4v9s87xhl-sandcastle-0.0.37/lib/sandcastle/sandcastle-cli.mjs`.

Key changes from v0.0.38 source to match v0.0.37:

1. Add `shellQuote()` helper function
2. Rename `--settings` to `--config` (v0.0.37 uses `--config`)
3. Add `--shell <shell>` option
4. Add `--tmpdir <path>` option
5. Add tmpdir lifecycle (create, setTmpdir, cleanup on exit)
6. Remove `-c <command>` mode (not in v0.0.37)
7. Use `shellQuote` for command building instead of simple `join(' ')`

Write the full updated `cli.ts`:

```typescript
#!/usr/bin/env node
import { Command } from 'commander'
import { SandboxManager } from './index.js'
import type { SandboxRuntimeConfig } from './sandbox/sandbox-config.js'
import { spawn } from 'child_process'
import { logForDebugging } from './utils/debug.js'
import { loadConfig, loadConfigFromString } from './utils/config-loader.js'
import * as readline from 'readline'
import * as fs from 'fs'
import * as path from 'path'
import * as os from 'os'

/**
 * Get default config path
 */
function getDefaultConfigPath(): string {
  return path.join(os.homedir(), '.srt-settings.json')
}

/**
 * Create a minimal default config if no config file exists
 */
function getDefaultConfig(): SandboxRuntimeConfig {
  return {
    network: {
      allowedDomains: [],
      deniedDomains: [],
    },
    filesystem: {
      denyRead: [],
      allowWrite: [],
      denyWrite: [],
    },
  }
}

function shellQuote(s: string): string {
  return "'" + s.replace(/'/g, "'\\''") + "'"
}

async function main(): Promise<void> {
  const program = new Command()

  program
    .name('sandcastle')
    .description(
      'Run commands in a sandbox with network and filesystem restrictions',
    )
    .version(process.env.npm_package_version || '1.0.0')

  // Default command - run command in sandbox
  program
    .argument('[command...]', 'command to run in the sandbox')
    .option('-d, --debug', 'enable debug logging')
    .option(
      '--config <path>',
      'path to config file (default: ~/.srt-settings.json)',
    )
    .option('--shell <shell>', 'shell to execute the command with')
    .option(
      '--tmpdir <path>',
      'override the temporary directory used inside the sandbox',
    )
    .option(
      '--no-tempdir-cleanup',
      'do not remove the temporary directory on exit',
    )
    .option(
      '--control-fd <fd>',
      'read config updates from file descriptor (JSON lines protocol)',
      parseInt,
    )
    .allowUnknownOption()
    .action(
      async (
        commandArgs: string[],
        options: {
          debug?: boolean
          config?: string
          shell?: string
          tmpdir?: string
          tempdirCleanup?: boolean
          controlFd?: number
        },
      ) => {
        try {
          // Enable debug logging if requested
          if (options.debug) {
            process.env.DEBUG = 'true'
          }

          // Load config from file
          const configPath = options.config || getDefaultConfigPath()
          let runtimeConfig = loadConfig(configPath)

          if (!runtimeConfig) {
            logForDebugging(
              `No config found at ${configPath}, using default config`,
            )
            runtimeConfig = getDefaultConfig()
          }

          // Tmpdir lifecycle
          let sandboxTmpdir: string
          let cleanupTmpdir = false

          if (options.tmpdir) {
            sandboxTmpdir = options.tmpdir
            fs.mkdirSync(sandboxTmpdir, { recursive: true })
          } else {
            sandboxTmpdir = fs.mkdtempSync(
              path.join(os.tmpdir(), 'sandcastle-'),
            )
            cleanupTmpdir = true
          }

          // --no-tempdir-cleanup disables auto-cleanup
          if (options.tempdirCleanup === false) {
            cleanupTmpdir = false
          }

          SandboxManager.setTmpdir(sandboxTmpdir)

          process.on('exit', () => {
            if (cleanupTmpdir && sandboxTmpdir) {
              try {
                fs.rmSync(sandboxTmpdir, { recursive: true, force: true })
              } catch {
                // Best-effort cleanup
              }
            }
          })

          // Initialize sandbox with config
          logForDebugging('Initializing sandbox...')
          await SandboxManager.initialize(runtimeConfig)

          // Set up control fd for dynamic config updates if specified
          let controlReader: readline.Interface | null = null
          if (options.controlFd !== undefined) {
            try {
              const controlStream = fs.createReadStream('', {
                fd: options.controlFd,
              })
              controlReader = readline.createInterface({
                input: controlStream,
                crlfDelay: Infinity,
              })

              controlReader.on('line', line => {
                const newConfig = loadConfigFromString(line)
                if (newConfig) {
                  logForDebugging(
                    `Config updated from control fd: ${JSON.stringify(newConfig)}`,
                  )
                  SandboxManager.updateConfig(newConfig)
                } else if (line.trim()) {
                  // Only log non-empty lines that failed to parse
                  logForDebugging(
                    `Invalid config on control fd (ignored): ${line}`,
                  )
                }
              })

              controlReader.on('error', err => {
                logForDebugging(`Control fd error: ${err.message}`)
              })

              logForDebugging(
                `Listening for config updates on fd ${options.controlFd}`,
              )
            } catch (err) {
              logForDebugging(
                `Failed to open control fd ${options.controlFd}: ${err instanceof Error ? err.message : String(err)}`,
              )
            }
          }

          // Cleanup control reader on exit
          process.on('exit', () => {
            controlReader?.close()
          })

          // Determine command string based on mode
          let command: string
          if (commandArgs.length > 0) {
            if (options.shell) {
              const quoted = commandArgs.map(shellQuote).join(' ')
              command = `${options.shell} -c ${shellQuote(quoted)}`
            } else {
              command = commandArgs.map(shellQuote).join(' ')
            }
            logForDebugging(`Command: ${command}`)
          } else {
            console.error(
              'Error: No command specified. Provide command arguments.',
            )
            process.exit(1)
          }

          logForDebugging(
            JSON.stringify(
              SandboxManager.getNetworkRestrictionConfig(),
              null,
              2,
            ),
          )

          // Wrap the command with sandbox restrictions
          const sandboxedCommand =
            await SandboxManager.wrapWithSandbox(command)

          // Execute the sandboxed command
          const child = spawn(sandboxedCommand, {
            shell: true,
            stdio: 'inherit',
          })

          // Handle process exit
          child.on('exit', (code, signal) => {
            // Clean up bwrap mount point artifacts before exiting.
            SandboxManager.cleanupAfterCommand()

            if (signal) {
              if (signal === 'SIGINT' || signal === 'SIGTERM') {
                process.exit(0)
              } else {
                console.error(`Process killed by signal: ${signal}`)
                process.exit(1)
              }
            }
            process.exit(code ?? 0)
          })

          child.on('error', error => {
            console.error(`Failed to execute command: ${error.message}`)
            process.exit(1)
          })

          // Handle cleanup on interrupt
          process.on('SIGINT', () => {
            child.kill('SIGINT')
          })

          process.on('SIGTERM', () => {
            child.kill('SIGTERM')
          })
        } catch (error) {
          console.error(
            `Error: ${error instanceof Error ? error.message : String(error)}`,
          )
          process.exit(1)
        }
      },
    )

  program.parse()
}

main().catch(error => {
  console.error('Fatal error:', error)
  process.exit(1)
})
```

Note on Commander.js `--no-tempdir-cleanup`: Commander auto-parses `--no-X`
flags as the negation of `--X`. So `--no-tempdir-cleanup` sets
`options.tempdirCleanup = false`. This is standard Commander.js behavior.

**Step 2: Verify sandcastle builds**

Run from `~/eng/repos/purse-first`:
```bash
nix build .#sandcastle
```
Expected: successful build with no errors.

**Step 3: Commit**

```bash
git add packages/sandcastle/src/cli.ts
git commit -m "feat(sandcastle): restore tmpdir lifecycle and add --no-tempdir-cleanup"
```

---

### Task 2: Update batman bats wrapper to pass through --no-tempdir-cleanup

**Files:**
- Modify: `~/eng/repos/purse-first/lib/packages/batman.nix`

**Step 1: Add --no-tempdir-cleanup to batman's flag parsing**

In `batman.nix`, modify the bats wrapper's `text` attribute. Add
`--no-tempdir-cleanup` to the flag-parsing `while` loop and pass it to both
sandcastle and bats.

In the `while (( $# > 0 ))` loop, add a new case after `--no-sandbox`:

```bash
      --no-tempdir-cleanup)
        no_tempdir_cleanup=true
        shift
        ;;
```

Initialize the variable at the top alongside `sandbox=true`:

```bash
      no_tempdir_cleanup=false
```

In the `if $sandbox` branch, change the `exec sandcastle` line to conditionally
include `--no-tempdir-cleanup`:

```bash
      sandcastle_args=()
      if $no_tempdir_cleanup; then
        sandcastle_args+=(--no-tempdir-cleanup)
        set -- --no-tempdir-cleanup "$@"
      fi

      exec sandcastle "''${sandcastle_args[@]}" --shell bash --config "$config" bats "$@"
```

Replace the `else` (no-sandbox) branch to also forward to bats:

```bash
      if $no_tempdir_cleanup; then
        set -- --no-tempdir-cleanup "$@"
      fi
      exec bats "$@"
```

**Step 2: Verify batman builds**

Run from `~/eng/repos/purse-first`:
```bash
nix build .#batman
```
Expected: successful build.

**Step 3: Verify the wrapper script has the new flag**

```bash
cat result/bin/bats | grep -A2 'no-tempdir-cleanup'
```
Expected: shows the new case in the wrapper.

**Step 4: Commit**

```bash
git add lib/packages/batman.nix
git commit -m "feat(batman): pass --no-tempdir-cleanup through to sandcastle and bats"
```

---

### Task 3: Add batman wrapper test for --no-tempdir-cleanup

**Files:**
- Modify: `~/eng/repos/purse-first/packages/batman/zz-tests_bats/bats_wrapper.bats`

**Step 1: Write the test**

Add a new test function at the end of `bats_wrapper.bats`:

```bash
function bats_wrapper_no_tempdir_cleanup_preserves_tmpdir { # @test
  cat >"${TEST_TMPDIR}/preserve.bats" <<'EOF'
#! /usr/bin/env bats
function creates_file_in_tmpdir { # @test
  echo "marker" > "${BATS_TEST_TMPDIR}/marker.txt"
}
EOF
  run "$BATS_WRAPPER" --no-tempdir-cleanup "${TEST_TMPDIR}/preserve.bats"
  assert_success
  assert_output --partial "ok 1"
  # Extract BATS_RUN_TMPDIR from output (printed by --no-tempdir-cleanup)
  bats_run_dir="$(echo "$output" | grep "BATS_RUN_TMPDIR" | cut -d' ' -f2)"
  [[ -n "$bats_run_dir" ]]
  # Verify the temp dir survived sandcastle cleanup
  [[ -d "$bats_run_dir" ]]
  [[ -f "$bats_run_dir/test/1/marker.txt" ]]
  # Clean up manually
  rm -rf "$bats_run_dir"
}
```

**Step 2: Run the test to verify it passes**

```bash
cd ~/eng/repos/purse-first/packages/batman
bats --no-sandbox zz-tests_bats/bats_wrapper.bats
```
Expected: all tests pass, including the new one.

**Step 3: Commit**

```bash
git add packages/batman/zz-tests_bats/bats_wrapper.bats
git commit -m "test(batman): add test for --no-tempdir-cleanup preservation"
```

---

### Task 4: Update dodder fixture generation

**Files:**
- Modify: `~/eng/repos/dodder/zz-tests_bats/migration/generate_fixture.bash`

**Step 1: Replace --no-sandbox with --no-tempdir-cleanup**

In `generate_fixture.bash`, change the `cmd_bats` array:

```bash
cmd_bats=(
  bats
  --no-tempdir-cleanup
  --tap
  --no-tempdir-cleanup
  migration/generate_fixture.bats
)
```

Wait -- bats' `--no-tempdir-cleanup` is already there. The batman wrapper's
`--no-tempdir-cleanup` is a *batman-level* flag that gets consumed before
bats args. Since the wrapper strips its own flags before passing to bats,
we only need one `--no-tempdir-cleanup` and it will be handled by batman
(which passes it to both sandcastle and bats):

```bash
cmd_bats=(
  bats
  --no-tempdir-cleanup
  --tap
  migration/generate_fixture.bats
)
```

This replaces the current `--no-sandbox` (our temporary workaround from this
session). The flag is consumed by batman's flag loop, then forwarded to both
sandcastle (`--no-tempdir-cleanup` disables tmpdir cleanup) and bats
(`--no-tempdir-cleanup` preserves bats' own temp dirs).

**Step 2: Verify fixture generation works**

```bash
cd ~/eng/repos/dodder
just test-bats-update-fixtures
```
Expected: fixtures generated successfully, `git diff --stat` shows fixture
changes.

**Step 3: Commit**

```bash
git add zz-tests_bats/migration/generate_fixture.bash
git commit -m "fix: use --no-tempdir-cleanup instead of --no-sandbox for fixture generation"
```
