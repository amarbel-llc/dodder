# checked_out_state

Checkout state enum for tracking object sync status.

## States

- `Internal`: Only in store
- `CheckedOut`: Synced to working copy
- `Untracked`: Only in working copy
- `Recognized`: Matched but not synced
- `Conflicted`: Sync conflict detected
