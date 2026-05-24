---
author:
-
date: April 2026
title: WORKSPACE(7) Dodder \| Miscellaneous
---

# NAME

workspace - dodder working directory for editing objects

# DESCRIPTION

A workspace is a directory where dodder objects are checked out as editable
files. It bridges the canonical object store and the filesystem, providing a
checkout/edit/checkin workflow similar to Git's working tree.

Workspaces are configured by a **.dodder-workspace** file. When running dodder
commands, the workspace is discovered by walking up from the current directory.
If no workspace file is found, a temporary workspace in **/tmp** is used for
read-only operations.

# WORKSPACE TYPES

## Lightweight Workspace

The default workspace type. Created with **dodder init-workspace**. Objects are
checked out from and committed directly to the parent repository's store. No
independent history is maintained.

## Repo-Backed Workspace (experimental)

Created with **dodder init-workspace -experimental-repo**. A full dodder
repository (**.dodder/** directory) is created at the workspace location with
its own inventory list, signing key, and commit history. Changes accumulate in
the workspace's history and propagate to the parent only on explicit **push**.

This provides an isolation boundary: automated tools can mutate the workspace
freely, with a review gate before changes reach the parent.

**Blob storage** is shared with the parent rather than independent. The
workspace's **.madder/local/share/blob_stores/** holds a pointer config (a
**!toml-blob_store_config-pointer-v1**, see **blob-store**(7)) that resolves
by absolute path to the parent repo's default blob store. Reads through the
pointer fetch from the parent's store; writes from the workspace land directly
there. The workspace never maintains a separate copy of any blob.

The pointer is locked to an absolute path at init time. If the parent's blob
store moves, the pointer breaks --- recover by hand-editing the workspace's
**blob_store-config** to point at the new location. If the parent has no
default blob store at init time, **init-workspace** cancels with a clear
error rather than producing a dangling pointer.

# CONFIGURATION

The **.dodder-workspace** file is a hyphence-encoded document (see
**hyphence**(7)). Use **dodder info-workspace** to inspect it.

## Fields

**query**
:   Default doddish query filter (see **doddish**(7)). Used by **show** and as
    the pull filter for repo-backed workspaces.

**defaults.type**
:   Default type assigned to new objects created in this workspace.

**defaults.tags**
:   Default tags applied to new objects.

Repo-backed workspaces add:

**parent-path**
:   Absolute path to the parent repository.

**sync-tai**, **sync-digest**
:   Baseline timestamp and digest for divergence detection.

# OBJECT STATES

Objects in a workspace have one of these states, shown by **dodder status**:

**CheckedOut**
:   Synced between store and workspace. Ready for editing.

**Recognized**
:   Store and working copy differ. The object was modified externally.

**Untracked**
:   Exists only in the workspace. Not yet committed to the store.

**Conflicted**
:   Merge conflict detected during sync.

# WORKFLOW

## Basic Checkout/Edit/Checkin

Initialize a workspace, check out objects, edit them, and commit:

    dodder init-workspace
    dodder checkout todo
    # edit files in the workspace
    dodder checkin

## Inspect Changes

    dodder status          # list objects and their states
    dodder diff            # show store vs workspace differences

## Batch Organization

    dodder organize todo   # open tagged objects in text editor

The organize-text format groups objects under tag headings. Moving objects
between headings changes their tags. See **organize-text**(7).

## Remove Working Copies

    dodder clean           # remove unmodified checked-out objects
    dodder clean -force    # remove all, including modified

## Repo-Backed Sync

    dodder pull            # fetch parent changes (filtered by query)
    dodder push            # commit workspace changes to parent

Pull respects the workspace query filter with edge reachability (depth 5):
referenced objects are included even if they don't match the query, ensuring
referential integrity for tags and types.

Push is unfiltered --- all workspace objects propagate.

## Divergence Detection

    dodder check-workspace dirty

Exit codes: 0 = workspace has uncommitted changes, 1 = clean, 2 = not in a
workspace.

# SEE ALSO

**dodder**(1), **doddish**(7), **organize-text**(7), **hyphence**(7)
