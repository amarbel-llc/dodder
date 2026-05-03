# repo_actions

High-level actions performed on a local working copy repository.

## Construction

All action types embed a private `*repo` field (a type alias for
`local_working_copy.Repo`) and **must** be created via `Make*` constructors.
Struct literal construction from outside the package is a compile error.

Required fields are constructor parameters; optional fields (booleans, strings,
Proto) are set via field assignment after construction:

``` go
op := repo_actions.MakeCheckout(repo)
op.Edit = true
op.Options = myOptions
```

## Key Actions

- `Checkin`: Check in changes from working copy
- `Checkout`: Check out objects to working copy
- `CreateFromPaths`: Create objects from file paths
- `CreateFromShas`: Create objects from SHA references
- `Diff`: Compare internal vs external object versions
- `Organize` / `Organize2`: Organize zettels interactively via editor
- `UpdateObject`: Update metadata/tags/type/blob of existing objects
- `WriteNewZettels`: Batch create empty zettels
- `ExecLua`: Execute Lua scripts on objects
- `OpenEditor`: Open files in editor
- `EachBlob`: Run utility command on checked-out blob files
- `CreateOrganizeFile` / `ReadOrganizeFile`: Generate and parse organize text
- `CheckinHaustoria` / `NewHaustoria`: CalDAV/remote source operations

## Organize Options

Standalone functions for building organize text options, moved from
`local_working_copy.Repo`:

- `MakeOrganizeOptionsWithOrganizeMetadata`
- `MakeOrganizeOptionsWithQueryGroup`
- `ApplyToOrganizeOptions`
- `LockAndCommitOrganizeResults`
