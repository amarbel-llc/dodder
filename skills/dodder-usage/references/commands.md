# Command Reference

Complete dodder/der command reference organized by workflow. Every command listed
here works identically with both `dodder` and `der`.

## Repository Management

### init

Initialize a new dodder repository.

**Positional arguments:** `repo-id` (required)

**Key flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `-yin` | `""` | File containing list of zettel ID left parts |
| `-yang` | `""` | File containing list of zettel ID right parts |
| `-override-xdg-with-cwd` | `false` | Create `.dodder/` in current directory instead of using XDG |
| `-inventory_list-type` | (current version) | Type for inventory lists |
| `-blob_store-id` | `""` | Name of an existing madder blob store to use |

Genesis configuration flags (set at creation time):

| Flag | Description |
|------|-------------|
| `-lock-internal-files` | Lock internal store files |
| `-compression` | Blob compression type |
| `-encryption` | Blob encryption type |

```bash
dodder init my-repo
dodder init -override-xdg-with-cwd my-repo
dodder init -yin left-words.txt -yang right-words.txt my-repo
```

### clone

Clone a repository from a remote source.

**Positional arguments:** `new-repo-id` (required), remote URI (required),
optional query arguments

**Key flags:** Same genesis flags as `init`, plus remote transfer flags.

```bash
dodder clone local-id remote:///path/to/source
dodder clone local-id remote:///path !md:z
dodder clone -override-xdg-with-cwd local-id remote:///path
```

### deinit

Remove a dodder repository. Deletes XDG data directories or the local
`.dodder/` directory and workspace configuration.

**Key flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `-force` | `false` | Skip confirmation prompt |

```bash
dodder deinit
dodder deinit -force
```

### info

Display repository information.

**Positional arguments:** One or more info keys (default: `store-version`)

**Supported keys:**

| Key | Output |
|-----|--------|
| `store-version` | Current store format version |
| `store-version-next` | Next store format version |
| `compression-type` | Blob compression algorithm |
| `age-encryption` | Blob encryption status |
| `env` | All environment variables in dotenv format |
| `xdg` | XDG directory paths in dotenv format |

```bash
dodder info
dodder info store-version
dodder info compression-type age-encryption
dodder info env
dodder info xdg
```

### info-repo

Display detailed repository metadata.

```bash
dodder info-repo
```

## Workspace Management

### init-workspace

Create a workspace in the current or specified directory. A workspace provides
default tags, type, and query for operations within that directory.

**Positional arguments:** Optional directory path (creates and enters directory)

**Key flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `-tags` | `""` | Default tags for `checkin`, `new`, `organize` |
| `-type` | `""` | Default type for `new` and `organize` |
| `-query` | `""` | Default query for `show` |

```bash
dodder init-workspace
dodder init-workspace project-dir
dodder init-workspace -tags work -type md
dodder init-workspace -query "project:z"
```

### info-workspace

Display workspace configuration.

**Positional arguments:** Optional info key

**Supported keys:**

| Key | Output |
|-----|--------|
| (empty) or `config` | Full workspace configuration |
| `query` | Default query string |
| `defaults.type` | Default type |
| `defaults.tags` | Default tags |

```bash
dodder info-workspace
dodder info-workspace query
dodder info-workspace defaults.tags
```

### status

Show the state of checked-out objects in the workspace. Displays the
relationship between store objects and their working copy representations.

**Positional arguments:** Optional query arguments

```bash
dodder status
dodder status one/uno
```

### clean

Remove checked-out files from the working directory.

**Key flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `-force` | `false` | Remove files even if they have changes |
| `-include-mutter` | `false` | Remove files matching their parent version |
| `-recognized-blobs` | `false` | Also remove recognized blobs |
| `-recognized-zettelen` | `false` | Also remove recognized zettels |
| `-organize` | `false` | Open organize UI to select removals |

```bash
dodder clean
dodder clean -force
dodder clean -organize
dodder clean -recognized-blobs
```

## Content Creation

### new

Create new zettels. By default, creates one empty zettel and opens it in the
editor.

**Positional arguments:** Optional file paths to import

**Key flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `-edit` | `true` | Open editor after creation |
| `-count` | `1` | Number of empty zettels to create |
| `-tags` | `""` | Tags to apply to new zettels |
| `-type` | `""` | Type for new zettels |
| `-description` | `""` | Description for new zettels |
| `-shas` | `false` | Treat arguments as already-checked-in blob SHAs |
| `-filter` | `""` | Script to transform input files |
| `-delete` | `false` | Delete source files after import |
| `-organize` | `false` | Open organize UI after creation |
| `-kasten` | `""` | Target repo (none or Browser) |

```bash
dodder new
dodder new -edit=false
dodder new -edit=false document.txt
dodder new -tags project -type md -description "My note"
dodder new -count=5 -edit=false
dodder new -shas abc123 def456
dodder new -filter transform.lua file1.txt file2.txt
```

## Viewing

### show

Query and display objects. Default format is `log` (one-line summary).

**Positional arguments:** Query arguments (default genre: zettel)

**Key flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `-format` | `log` | Output format: `log`, `text`, or `json` |
| `-before` | (none) | Show objects before this timestamp (RFC3339) |
| `-after` | (none) | Show objects after this timestamp (RFC3339) |
| `-repo` | (none) | Query a remote repository by ID |

```bash
dodder show :z
dodder show -format text one/uno
dodder show -format json :z
dodder show -before 2024-06-01T00:00:00Z :z
dodder show -repo remote-id :z
dodder show tag-name:z
dodder show !md:z
```

### cat

Output raw content. Used with Alfred workflow integration (`cat-alfred`).

**Key flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `-genre` | (none) | Extract specific genre from matching objects |

```bash
dodder cat-alfred :z
dodder cat-alfred -genre tag :z
```

### last

Display the most recently changed objects from the latest inventory list.

**Key flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `-format` | `log` | Output format |
| `-edit` | `false` | Open results in editor |
| `-organize` | `false` | Open results in organize UI |
| `-kasten` | (none) | Target repo |

```bash
dodder last
dodder last -format text
dodder last -edit
dodder last -organize
```

## Editing

### edit

Open objects in the configured editor.

**Positional arguments:** Query arguments (default genres: zettel, tag, type,
repo)

**Key flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `-mode` | `default` | Edit mode: `metadata-and-blob`, `metadata-only`, `blob-only` |
| `-delete` | `false` | Delete working copy after editing |
| `-organize` | `false` | Open organize after editing |

```bash
dodder edit one/uno
dodder edit -mode metadata-only one/uno
dodder edit -mode blob-only :z
dodder edit -delete one/uno
```

### checkin

Commit changes from the working copy back to the store. Aliases: `add`, `save`.

**Positional arguments:** Query arguments (default sigil: external, default
genres: all)

**Key flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `-ignore-blob` | `false` | Update metadata only, do not change the blob |
| `-each-blob` | `""` | Checkout each blob and run a command on it |
| `-tags` | `""` | Tags to add during checkin |
| `-type` | `""` | Type to set during checkin |
| `-description` | `""` | Description to set during checkin |
| `-delete` | `false` | Delete working copy after checkin |
| `-organize` | `false` | Open organize after checkin |

```bash
dodder checkin one/uno
dodder checkin -ignore-blob one/uno
dodder checkin -each-blob "pandoc -t html" :z
dodder checkin -tags project -type md :z
dodder checkin -delete one/uno
dodder add one/uno        # alias for checkin
dodder save one/uno       # alias for checkin
```

## File Operations

### checkout

Sync objects from the store to the filesystem.

**Positional arguments:** Query arguments (default genre: zettel)

**Key flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `-organize` | `false` | Open organize after checkout |

Checkout mode and text formatter flags are inherited from the checkout options
component.

```bash
dodder checkout :z
dodder checkout one/uno
dodder checkout -organize :z
dodder checkout :?z
```

### organize

Bulk edit objects via a structured text file.

**Positional arguments:** Query arguments (default genre: zettel)

**Key flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `-mode` | `interactive` | Mode: `output-only`, `commit-directly`, `interactive` |
| `-filter` | `""` | Script to transform organize text |

Organize text flags (prefix-joints, refine) are set via the `organize_text.Flags`
component.

```bash
dodder organize :z
dodder organize -mode output-only :z
dodder organize -mode commit-directly :z
dodder organize tag-name:z
dodder organize !md:z
```

### diff

Show differences between store and working copy.

**Positional arguments:** Query arguments (default genres: all)

```bash
dodder diff
dodder diff one/uno
```

### export

Export objects as an inventory list to stdout.

**Positional arguments:** Query arguments (default genre: inventory list)

**Key flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `-age-identity` | (none) | Age identity for encryption |
| `-compression` | (none) | Compression type override |

```bash
dodder export :b
dodder export :t,konfig
dodder export -age-identity identity.txt :b
```

## Remote Sync

### remote-add

Register a remote repository.

**Positional arguments:** Remote URI, repo-id

**Key flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `-tags` | `""` | Tags for the remote object |
| `-description` | `""` | Description for the remote object |

```bash
dodder remote-add remote:///path/to/repo my-remote
dodder remote-add -description "Backup" remote:///backup backup-id
```

### push

Push local changes to a remote repository.

**Positional arguments:** `repo-id` (required), optional query arguments

```bash
dodder push remote-id
dodder push remote-id !md:z
```

### pull

Pull remote changes to the local repository.

**Positional arguments:** `repo-id` (required), optional query arguments

```bash
dodder pull remote-id
dodder pull remote-id !md:z
```

### pull-blob-store

Pull blob data from an external blob store.

**Positional arguments:** `blob_store-base-path`, `blob_store-config-path`,
optional query arguments

```bash
dodder pull-blob-store /path/to/store /path/to/config
```

## Maintenance

### reindex

Rebuild all indices from inventory lists. Takes no arguments.

```bash
dodder reindex
```

### fsck

Verify integrity of all objects. Reports errors without modifying data.

**Positional arguments:** Optional query arguments

**Key flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `-inventory_list-path` | `""` | Verify from a specific inventory list file |
| `-object-sig-required` | `true` | Require object signatures |
| `-skip-probes` | `false` | Skip probe index verification |
| `-skip-blobs` | `false` | Skip blob content verification |

```bash
dodder fsck
dodder fsck -skip-blobs
dodder fsck -skip-probes
dodder fsck -inventory_list-path /path/to/list
```

### repo-fsck

Verify repository-level integrity.

```bash
dodder repo-fsck
```

### find-missing

Check whether blob SHAs exist in the store.

**Positional arguments:** One or more blob digest strings

```bash
dodder find-missing abc123def456 789012abc345
```

### revert

Revert objects to a previous version.

**Positional arguments:** Query arguments (default genres: zettel, tag, type,
repo)

**Key flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `-last` | `false` | Revert all changes from the last inventory list |

```bash
dodder revert one/uno
dodder revert -last
```

### edit-config

Edit the repository configuration in the editor.

```bash
dodder edit-config
```

## Dormant Objects

### dormant-add

Mark objects matching specified tags as dormant (hidden from default queries).

**Positional arguments:** One or more tag names

```bash
dodder dormant-add archive
dodder dormant-add old-project deprecated
```

### dormant-edit

Open the dormant configuration blob in the editor for direct editing.

```bash
dodder dormant-edit
```

### dormant-remove

Remove tags from the dormant list, making matching objects visible again.

**Positional arguments:** One or more tag names

```bash
dodder dormant-remove archive
```

## Global Flags

These flags are available on all commands via the CLI configuration:

| Flag | Default | Description |
|------|---------|-------------|
| `-dir-dodder` | `""` | Override the dodder base directory path |
| `-ignore-workspace` | `false` | Ignore workspace defaults, use temporary workspace |
| `-ignore-hook-errors` | `false` | Continue despite hook errors |
| `-hooks` | `""` | Path to hooks configuration |
| `-checkout-cache-enabled` | `false` | Enable checkout caching |
| `-predictable-zettel-ids` | `false` | Generate IDs in sequential order (testing) |
| `-comment` | `""` | Comment attached to the inventory list |
| `-debug` | `false` | Enable debug output |
| `-dry-run` | `false` | Preview changes without committing |
| `-verbose` | `false` | Increase output verbosity |
