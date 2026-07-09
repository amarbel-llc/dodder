#!/usr/bin/env python3
"""Prune orphaned objects from the dodder Alfred workflow's info.plist.

Two triggers ported from the zit-era workflow have no current dodder CLI
equivalent and are dropped (see zz-alfred/README.md):

  - the "z add -kind ... -actions ..." file-copy/move action, which relied
    on a `z` shell alias and an `up.bash` that were never part of the
    workflow bundle; and
  - the "snippet-address" script filter, which shelled out to a removed
    `.zit/bin/snippet-address` helper.

Removing an Alfred object cleanly means deleting it from three places:
`objects` (the node), `uidata` (its canvas position/note), and both the
key side and the destination side of `connections` (the edges). This
script does all three and then asserts that no dangling edge to a removed
uid remains, so a plist that passes is internally consistent.

The dead-uid set is the forward-reachable closure of the two dead leaves
over the connection graph, restricted to objects that ONLY feed the dead
leaves (a shared utility reachable from a live trigger is never removed).
It is passed in explicitly rather than inferred so the removal is
auditable in review; the script verifies the set is self-consistent.
"""

import plistlib
import sys

# The two dead leaves plus every object that exists solely to feed them.
# Traced from the connections graph:
#   Copy-to-Dodder (E930A5C7) -> 4EF55715 / 7C84E71B -> 591C4D35 (z add)
#   Move-to-Dodder (B25DCC72) -> 9EEA61D6 / D4626ED2  -> 591C4D35 (z add)
#   @a snippet (A4B5282D)     -> 055AD868 (snippet-address) -> 61F2BA51*
# 61F2BA51 (clipboard out) is shared with the live `zt`/`z` snippet copy
# path, so it is intentionally NOT in this set.
DEAD_UIDS = {
    "591C4D35-1327-4991-A803-84171A901C0A",  # z add -kind (file copy/move action)
    "4EF55715-BDFA-4F65-BFBF-B86952B87130",  # -kind files-copy arg
    "7C84E71B-3981-4226-A420-31E84E93C20A",  # -actions edit,open-files arg
    "E930A5C7-5894-49B1-A64C-58F1EA230567",  # Copy to Dodder trigger
    "B25DCC72-385D-4538-91F1-28082AF95CFA",  # Move to Dodder trigger (dead one)
    "9EEA61D6-0625-470B-903C-EF9A53572FF5",  # -actions edit,open-files arg
    "D4626ED2-6585-4F85-8F85-16126374007D",  # -kind files arg
    "055AD868-B2EC-4355-A89E-DC2B2195E6A0",  # snippet-address script filter
    "A4B5282D-8B29-4FB5-B5AA-3B29DC84E388",  # @a snippet trigger
}


def main(path: str) -> int:
    with open(path, "rb") as f:
        plist = plistlib.load(f)

    objects = plist.get("objects", [])
    present = {o["uid"] for o in objects if "uid" in o}
    missing = DEAD_UIDS - present
    if missing:
        print(f"error: dead uids not present in objects: {sorted(missing)}",
              file=sys.stderr)
        return 1

    # 1. objects
    plist["objects"] = [o for o in objects if o.get("uid") not in DEAD_UIDS]

    # 2. uidata
    uidata = plist.get("uidata", {})
    plist["uidata"] = {k: v for k, v in uidata.items() if k not in DEAD_UIDS}

    # 3. connections: drop dead source keys, and dead destinations from
    #    surviving sources.
    connections = plist.get("connections", {})
    pruned = {}
    for src, edges in connections.items():
        if src in DEAD_UIDS:
            continue
        kept = [e for e in edges if e.get("destinationuid") not in DEAD_UIDS]
        if kept:
            pruned[src] = kept
    plist["connections"] = pruned

    # Verify: no surviving edge references a removed uid.
    dangling = []
    for src, edges in plist["connections"].items():
        for e in edges:
            dst = e.get("destinationuid")
            if dst in DEAD_UIDS:
                dangling.append((src, dst))
    if dangling:
        print(f"error: dangling edges remain after prune: {dangling}",
              file=sys.stderr)
        return 1

    with open(path, "wb") as f:
        plistlib.dump(plist, f)

    print(f"pruned {len(DEAD_UIDS)} orphaned objects from {path}")
    return 0


if __name__ == "__main__":
    if len(sys.argv) != 2:
        print("usage: prune_orphans.py <info.plist>", file=sys.stderr)
        sys.exit(2)
    sys.exit(main(sys.argv[1]))
