# TODO Inventory

This document catalogs all TODO comments in the Dodder codebase, organized by theme with recommendations for prioritization.

## Summary Statistics

- **Total TODOs**: ~280+
- **Priority-tagged TODOs**: ~25 (P1-P4 system)
- **Most affected areas**: queries, remote_http, store operations, UI/formatting

---

## Categories

### 1. Architecture & Refactoring (High Impact)

These TODOs represent structural improvements that would improve code maintainability and reduce technical debt.

#### Store Configuration Consolidation
- `src/sierra/store_config/compiled.go:31` - Combine repo_configs.Config dependencies
- `src/sierra/store_config/compiled.go:37,40,43` - Move functionality to store
- `src/sierra/store_config/main.go:25` - Remove gob, separate store functionality
- `src/sierra/store_config/persist.go:4` - Remove gob encoding

#### Package Relocation
- `src/hotel/blob_store_configs/toml_uri_v0.go:8` - Move to config_common package
- `src/india/env_dir/blob_writer.go:13` - Move into own package
- `src/india/env_dir/blob_config.go:11` - Move into own package
- `src/india/env_dir/blob_reader.go:15` - Move into own package
- `src/india/env_dir/blob_mover.go:12` - Move into own package
- `src/golf/repo_config_cli/main.go:35` - Move to store_config
- `src/whiskey/user_ops/each_blob.go:13` - Move to store_fs
- `src/whiskey/user_ops/diff.go:23` - Move to store_fs
- `src/uniform/store/dormancy_and_tags.go:16` - Extract into store_tags
- `src/bravo/quiter/strings.go:12,29,61` - Move to collections_slice
- `src/charlie/collections_value/construction.go:11,33,96` - Move construction to derived package

#### Query Builder Refactoring ✅ COMPLETED

~~Multiple related TODOs in `src/oscar/queries/builder.go`:~~
~~- Lines 94, 100, 106, 112, 118, 126, 134, 142, 150 - Refactor into BuilderOption pattern~~

**Status**: Completed in commit b9546cb6c
- Removed 10 With* methods (WithPermittedSigil, WithDoNotMatchEmpty, WithRequireNonEmptyQuery, WithDebug, WithRepoId, WithFileExtensions, WithExpanders, WithDefaultGenres, WithDefaultSigil, WithHidden)
- Added 5 new BuilderOptions (DoNotMatchEmpty, Debug, RepoId, FileExtensions, Expanders)
- Updated all call sites to use BuilderOptions composition
- Kept WithOptions, WithExternalLike, WithTransacted as methods

---

### 2. Error Handling & Context (Medium-High Impact)

#### Error System Redesign
- `src/alfa/errors/sentinels.go:10` - Redesign all sentinel errors
- `src/alfa/errors/main.go:137` - Remove/rewrite error utilities
- `src/alfa/errors/is.go:99` - Remove deprecated function
- `src/alfa/errors/context.go:24,99,117` - Extricate from *context into generic function
- `src/alfa/errors/context.go:28` - Add target error for early termination
- `src/alfa/errors/context.go:272` - Add interface for stack frames in cancellation error
- `src/alfa/errors/group_builder.go:13` - Consider pool with repool function

#### Error Context in Operations
- `src/oscar/queries/errors.go:15` - Add recovery text
- `src/golf/triple_hyphen_io/decoder.go:76` - Add context to errors
- `src/lima/box_format/read.go:92` - Switch to returning ErrBoxParse
- `src/bravo/flags/main.go:1129` - Switch to errors.BadRequestf

**Recommendation**: Error handling improvements would enhance debugging and user experience across the entire system.

---

### 3. Performance Optimization (Medium Impact)

#### Memory Pooling
- `src/golf/fd/construction.go:98` - Use pool
- `src/golf/tag_paths/tags_with_types.go:24` - Pool *Path's
- `src/mike/stream_index/binary_decoder.go:66` - Pool decoder buffers
- `src/charlie/collections_ptr/flag.go:18` - Add Resetter2 and Pool

#### Algorithm/Structure Improvements
- `src/golf/tag_paths/tags_with_types.go:29` - Improve performance
- `src/india/env_dir/util.go:30` - More performant double operation
- `src/uniform/store/mutating.go:332` - More performant sshfs operations
- `src/whiskey/remote_http/server.go:656` - Cache this operation
- `src/whiskey/remote_http/server.go:727,758` - More performant reader returns
- `src/hotel/objects/contents_tag_set.go:38,49` - Switch to binary search
- `src/mike/stream_index/binary_decoder.go:206` - Replace with buffered seeker

#### Query Optimization
- `src/oscar/queries/executor.go:300` - Cache query with sigil and object id

**Recommendation**: Focus on pooling improvements first as they're lower risk and provide consistent benefits.

---

### 4. Remote/HTTP Operations (Medium Impact)

#### Client Improvements
- `src/whiskey/remote_http/client.go:240` - Local/remote version negotiation
- `src/whiskey/remote_http/client.go:258` - Reader version of inventory lists
- `src/whiskey/remote_http/client_blob_store.go:136` - Option to collect and present errors
- `src/whiskey/remote_http/client_inventory_list_store.go:24` - Add progress bar
- `src/whiskey/remote_http/client_inventory_list_store.go:119` - Ensure conflicts addressed before import

#### Server Improvements
- `src/whiskey/remote_http/server.go:42` - Use context cancellation for HTTP errors
- `src/whiskey/remote_http/server.go:58` - Switch to not return error
- `src/whiskey/remote_http/server.go:161` - Add errors/context middleware
- `src/whiskey/remote_http/server.go:316,343` - Context vs error return cleanup
- `src/whiskey/remote_http/server.go:904` - Modify to not buffer
- `src/whiskey/remote_http/server_repo.go:187` - Make merge conflicts impossible

#### Transport Layer
- `src/whiskey/remote_http/round_tripper_wrapped_signer.go:24` - Extract signing into agnostic middleware
- `src/whiskey/remote_http/round_tripper_wrapped_signer.go:74` - Present TOFU prompt to user
- `src/whiskey/remote_http/round_tripper_stdio.go:33,37` - Better binary selection
- `src/whiskey/remote_http/round_tripper_unix_socket.go:18` - Add public key

**Recommendation**: The version negotiation and conflict handling are critical for remote reliability.

---

### 5. Security Improvements (High Priority)

- `src/juliett/blob_stores/util_ssh.go:24,97` - Make InsecureIgnoreHostKey configurable
- `src/lima/object_finalizer/lockfile.go:18` - Stop excluding builtin types from signing
- `src/whiskey/remote_http/round_tripper_wrapped_signer.go:74` - TOFU prompt for user

**Recommendation**: SSH host key verification should be addressed before production use with untrusted remotes.

---

### 6. Type System & ID Handling (Medium Impact)

#### Object ID Improvements
- `src/foxtrot/ids/object_id3.go:27,28` - Add binary marshaling, make fields private
- `src/foxtrot/ids/object_id2.go:199` - Perform validation
- `src/foxtrot/ids/object_id2.go:524` - Switch to SetWithSeq
- `src/foxtrot/ids/main.go:41` - Rename to BinaryTypeChecker
- `src/foxtrot/ids/main.go:99` - Rewrite to use ToSeq comparison

#### Markl/Hash System
- `src/foxtrot/markl/id_blech_coding.go:12,91` - Remove deprecated functions
- `src/foxtrot/markl/id_blech_coding.go:14` - Use registered format lengths
- `src/foxtrot/markl/id.go:233` - Enforce non-nil formats
- `src/foxtrot/markl/errors.go:142` - Add "wrong hasher" error type

---

### 7. Query System (Medium-High Impact)

- `src/oscar/queries/executor.go:34` - Use ExecutorPrimitive
- `src/oscar/queries/executor.go:57` - Refactor into internal methods
- `src/oscar/queries/executor.go:187` - Tease apart dotOperatorActive reliance
- `src/oscar/queries/build_state.go:78` - Switch to collections_slice
- `src/oscar/queries/build_state.go:223` - Convert to decision tree
- `src/oscar/queries/build_state.go:282` - Add support for digests and signatures
- `src/oscar/queries/build_state.go:483` - Use new generic and typed blobs
- `src/oscar/queries/primitive.go:17` - Migrate to query executor
- `src/oscar/queries/object_id.go:67` - Support exact matching

---

### 8. Organize Feature (Medium Impact)

- `src/papa/organize_text/constructor.go:39,74` - Use Type, fix tag issue
- `src/papa/organize_text/constructor.go:305` - Explore using shas as keys
- `src/papa/organize_text/metadata.go:42,45` - Replace with embedded *sku.Transacted, remove Matchers
- `src/papa/organize_text/option.go:25` - Add config for automatic dry run
- `src/papa/organize_text/option.go:110` - Add ApplyTo* support
- `src/papa/organize_text/assignment.go:17` - Move to object_factory
- `src/papa/organize_text/changeable.go:32` - Refactor to use single GetMetadataMutable() call
- `src/whiskey/user_ops/organize.go:22` - Migrate to Organize2
- `src/whiskey/user_ops/organize2.go:65` - Refactor into common vim processing loop

---

### 9. UI/Formatting (Low-Medium Impact)

- `src/bravo/ui/main.go:19` - Add compile-time verbosity
- `src/bravo/ui/main.go:85` - Add TAP printer
- `src/bravo/ui/todo.go:10` - Rename to comment printer
- `src/bravo/ui/cli_error_tree_state.go:45,117,160` - Error tree improvements
- `src/bravo/ui/cli_error_tree_state_stack.go:24` - Refactor parent/stack
- `src/hotel/env_ui/main.go:17,19` - Explore buffered writer, remove separate
- `src/victor/local_working_copy/printers.go:15,44` - Migrate to new format writers
- `src/lima/box_format/transacted.go:155` - Quote as necessary
- `src/foxtrot/descriptions/format_cli_generic.go:40` - Format ellipsis properly

---

### 10. Test Infrastructure (Low Impact)

- `src/sierra/store_config/main_test.go:12` - Remove test
- `src/papa/organize_text/reader_test.go:20,68` - Transition to TestContext, add pubkeys
- `src/bravo/doddish/scanner_test.go:15` - Transition to TestCase framework
- `src/bravo/ui/t.go:15` - Make private, switch to MakeTestContext
- `src/bravo/ui/t.go:120` - Move to AssertNotEqual

---

### 11. Blob Store Operations (Medium Impact)

- `src/juliett/blob_stores/verification.go:11,19` - Offer verification options, call VerifyBlob
- `src/juliett/blob_stores/store_remote_sftp.go:40,47` - Populate blobIOWrapper, extract struct
- `src/juliett/blob_stores/store_remote_sftp.go:124` - Read remote blob store config
- `src/juliett/blob_stores/store_remote_sftp.go:288` - Use hash type
- `src/juliett/blob_stores/store_remote_sftp.go:356` - Explore using env_dir.Mover
- `src/juliett/blob_stores/store_remote_sftp.go:575` - Combine with sftpReader
- `src/juliett/blob_stores/main.go:71,72` - Pass custom UI, consolidate envDir/ctx

---

### 12. Index/Stream Operations (Medium Impact)

- `src/mike/stream_index/binary_decoder.go:45,47` - Unembed fields
- `src/mike/stream_index/binary_decoder.go:55` - Transition to panic semantics
- `src/mike/stream_index/page.go:17` - Replace pageAdditions with sku.WorkingList
- `src/mike/stream_index/page.go:59` - Write to file-backed buffered writer
- `src/mike/stream_index/main.go:310` - Add errors.Context closure support
- `src/mike/stream_index/main.go:331` - Switch to errors.MakeWaitGroupParallel()
- `src/lima/object_probe_index/row.go:12,34` - Change to encoder

---

### 13. Lua Integration (Low Impact)

- `src/lima/sku_lua/lua_transacted_v2.go:14` - Transition to single Tags table
- `src/lima/sku_lua/lua_transacted_v2.go:98-102` - Add Description, Type, Tai, Blob, Cache
- `src/lima/sku_lua/lua_transacted_v1.go:97-101` - Add similar fields
- `src/mike/type_blobs/main.go:39` - Make typed hooks
- `src/mike/type_blobs/toml_v1.go:17` - Migrate to properly-typed hooks
- `src/bravo/lua/vm_pool_builder.go:21` - Support cloning of compiled

---

### 14. Working Copy & Checkout (Medium Impact)

- `src/victor/local_working_copy/main.go:48` - Switch key to workspace type
- `src/victor/local_working_copy/main.go:102` - Investigate removing unnecessary resets
- `src/victor/local_working_copy/op_get_blob_formatter.go:16` - Add checked out types support
- `src/victor/local_working_copy/op_get_blob_formatter.go:109` - Allow option to error on missing format
- `src/victor/local_working_copy/format_type.go:78,211` - Zettel default type, switch to typed variant
- `src/victor/local_working_copy/format.go:34,155,970` - Format and key restructuring
- `src/victor/local_working_copy/lock.go:17` - Print organize files on dry run
- `src/victor/local_working_copy/local_parent_negotiator.go:60` - Repool all skus except ancestor

---

### 15. Miscellaneous / Quick Wins

#### Environment Variables
- `src/india/env_dir/main.go:15,16` - Change to dodder-prefixed env vars

#### Naming/Renaming
- `src/golf/page_id/main.go:14` - Rename to Id
- `src/sierra/repo/remote_connection.go:48,67` - Rename fields
- `src/hotel/objects/contained_object.go:13` - Rename to lock entry
- `src/whiskey/remote_http/server.go:507` - Rename to blob id

#### Deprecated Code Removal
- `src/echo/directory_layout/v3.go:133,138` - Deprecate and remove
- `src/echo/age/write_closer.go:5` - Remove (P3)

---

## Priority Recommendations

### Immediate (High Value, Low Risk)

1. **SSH Host Key Verification** - Security critical
2. ~~**Query Builder Refactoring**~~ - ✅ COMPLETED (commit b9546cb6c)
3. **Error System Improvements** - Debugging benefits across codebase

### Short-Term (High Value, Medium Effort)

4. **Store Configuration Consolidation** - Reduce gob dependency, clean architecture
5. **Memory Pooling** - Consistent performance improvements
6. **Remote Version Negotiation** - Important for distributed use

### Medium-Term (Medium Value, Higher Effort)

7. **Package Relocations** - Improve module organization
8. **Query Executor Improvements** - Better query performance and maintainability
9. **Organize Feature Migration** - Complete Organize2 transition

### Long-Term / As-Needed

10. **Test Infrastructure Updates** - Migrate as tests are modified
11. **UI/Formatting Improvements** - Address during feature work
12. **Lua Integration Enhancements** - When extending Lua capabilities

---

## Notes

- Priority tags (P1-P4) exist in some TODOs but aren't consistently used
- Many TODOs are in pairs (duplicate code in constructor.go/constructor2.go)
- Several "remove" TODOs suggest deprecated code that could be cleaned up
- Some TODOs reference external issues (e.g., the organize tag issue)
