# env_dir

Construction-arg shim around `madder/go/pkgs/env_dir`.

The package is a single file (`construction.go`) wrapping
`mad_env_dir.MakeDefault`, `MakeWithHomeAndInitialize`, etc. Its only
job is injecting `dodder_env.OwnConfig(debugOptions)` so dodder
processes keep honoring `DODDER_XDG_UTILITY_OVERRIDE` regardless of
the utility scope being constructed (own / madder).

For everything else — types (`Env`, `RelativePath`, `TemporaryFS`,
`Config`), helpers (`NewReader`, `NewWriter`, `MakeHashBucketPath`),
errors (`ErrBlobMissing`, `IsErrBlobAlreadyExists`) — import the
madder packages directly:

- `mad_env_dir "code.linenisgreat.com/madder/go/pkgs/env_dir"`
- `mad_blob_io "code.linenisgreat.com/madder/go/pkgs/blob_io"`

The previous alias-forwarding `main.go` was deleted; no code outside
this package should reference an `env_dir.X` symbol other than the
constructor wrappers.
