Dodder is distributed platform for creating, editing, and deploying blobs /
files / knowledges. Dodder takes from both Zettelkasten and Git as inspiration:

-   everything has an automatically-assigned identifier
-   flat hierarchy (no directories)
-   tags
-   everything is a note or blob
-   every change is tracked and saved

Zettelkasten is a knowledge-management method that is used by academics and
professionals to graph their thinking to make it more usable and navigable.
Examples of Zettelkasten-like systems are Roam Research and Obsidian.

And Git, well you know Git.

# Object ID's

Normally in Zettelkasten, timestamps are used as identifiers. Dodder deviates
from this as timestamps are not very ergonomic to type, autocomplete, or
disambiguate. Dodder instead uses an identifier system that combines two lists
of user-supplied identifiers to generate unique combinations. Here's an example:

-   user supplies two lists
    -   list a
        -   red
        -   green
        -   blue
    -   list b
        -   apple
        -   banana
        -   orange
    -   possible identifiers
        -   red/apple
        -   red/banana
        -   red/orange
        -   ...
        -   blue/orange
-   as new zettels are created, new and unused combinations are used to generate
    new identifiers that are then assigned to the zettel

# Contributions

At this time, contributions are welcome only after explicitly getting approval
from one of the authors.

# directory structure

## top-level

I originally wrote this project in Go. However, as the project has grown larger,
I've bumped into some of Go's tradeoffs that have led me to attempt rewriting
small parts of Dodder in Rust.

## per-edition

Besides the entrypoints (e.g., main.go, main.rs), each edition uses a strategy
for forcing modularity and unidirectional dependencies like so:

-   dodder/go/internal/
    -   alfa
        -   errors
    -   bravo
        -   files
    -   charlie
        -   file_lock

In the above selection, code in the `file_lock` module may depend on anything at
the `alfa` and `bravo` levels. Code in the `files` module may depend on anything
at the `alfa` level. And code in `alfa` may only depend on stdlib or external
modules. This encourages moduarity and also helps avoid go's lack of support for
circular dependencies. This also may have positive effects on build time as
modules are likely cached by the go build process.

For the code "levels", the NATO-phonetic alphabet is used. The benefits of this
approach are it encourages modularity and creating a pyramid-like structure of
dependency. For languages like Go, circular dependencies are not allowed and so
this approach prevents hard-to-untangle dependency refactors. For Rust, this
approach may help with build-times.
