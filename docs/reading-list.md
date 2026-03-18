# Reading List: Dodder-Relevant Saved Stories

Extracted from NewsBlur starred stories via nebulous MCP (2026-03-15).

## Merkle Trees & Content-Addressable Structures

- **Building a Transparent Keyserver** — Full tlog stack (Tessera/Torchwood/witnesses) for auditable content-addressable operations
  - hash: `9490228:ce3c75` | https://words.filippo.io/keyserver-tlog/
- **Monoidal Hashing** — `H(a++b) = H(a)*H(b)` enables arbitrary chunk boundaries and parallel sync; alternative to Merkle trees
  - hash: `9490228:cf6d3c` | https://www.scannedinavian.com/monoidal-hashing.html
- **Merkle Tree Certificates** — IETF draft for tlog-native TLS certs replacing CT's SCT model
  - hash: `6327282:a5d222` | https://blog.cloudflare.com/bootstrap-mtc/
- **Setsum** — O(1) commutative checksum for replica divergence detection; tells you *that* states differ, not *where*
  - hash: `9490228:9e2521` | https://avi.im/blag/2025/setsum/
- **How CT Works** — Primer on Certificate Transparency's Merkle-based append-only logs
  - hash: `0:542650` | https://certificate.transparency.dev/howctworks/
- **Merkle Town** — Cloudflare's live CT log explorer
  - hash: `6327282:cbc3c0` | https://ct.cloudflare.com/
- **Scrapscript** — Content-addressable programming language; expressions identified by hash, not name
  - hash: `6327282:18f214` | https://github.com/tekknolagi/scrapscript

## CRDTs & Distributed Data Structures

- **CRDT Emulation, Simulation, and Representation Independence** (ICFP 2025) — State-based vs op-based CRDTs are interchangeable; sync protocol choice is an implementation detail
  - hash: `9490228:c3bcae` | https://decomposition.al/blog/2025/06/29/crdt-emulation-simulation-and-representation-independence-will-appear-at-icfp-2025/
- **A simple way to understand CRDTs** — Algebraic framing: ACI properties neutralize network ordering corruption
  - hash: `9490228:395df7` | https://interjectedfuture.com/a-simple-way-to-understand-crdts/
- **Building Document-Centric, CRDT-Native Editors** — Making CRDT the primary data model, not an add-on sync layer
  - hash: `6327282:b499c8` | https://blocksuite.io/blog/document-centric.html
- **CRDT Concepts: Causal Trees** — Operations as tree nodes with causal parent pointers; natural fit for blob mutation history
  - hash: `6327282:3514a6` | https://www.farley.ai/posts/causal
- **CRDT: Text Buffer** — Evan Wallace's interactive explainer on sequence CRDTs
  - hash: `6327282:ce1142` | https://madebyevan.com/algos/crdt-text-buffer/
- **You don't need a CRDT** — Counterweight: many collaborative use cases are simpler than full CRDTs
  - hash: `6327282:208183` | https://zknill.io/posts/collaboration-no-crdts/

## Local-First Architecture

- **Local-first software: You own your data** (Ink & Switch) — The canonical reference; seven ideals for local-first
  - hash: `0:540562` | https://www.inkandswitch.com/local-first/
- **Local-First Foundations** — Primer: no canonical server, CRDTs as the enabling technology
  - hash: `9490228:a05556` | https://bytes.zone/posts/local-first-foundations/
- **Local-First From Scratch, part 1** — Building local-first from first principles; logical clocks → state CRDTs → delta-state sync
  - hash: `9490228:7d31a6` | https://bytes.zone/micro/lffs-001/
- **LFFS: Simplicity vs Efficiency** — Content-addressed blobs are natural CRDT atoms; `split` into irreducible atoms for delta-sync
  - hash: `9490228:059858` | https://bytes.zone/micro/lffs-002/
- **Fireproof** — Local-first database with content-addressed encrypted blocks, Git-like sync, pluggable backends
  - hash: `6327282:fef7d9` | https://fireproof.storage/
- **A Local-First Case Study** — Trip planner built with Automerge; friction of peer discovery in practice
  - hash: `9372890:0527ab` | https://jakelazaroff.com/words/a-local-first-case-study/
- **CADmium** — Local-first CAD; immutable operation log as document model (same structure as dodder's append-only blobs)
  - hash: `6327282:ae4ee1` | https://mattferraro.dev/posts/cadmium
- **The Cloud Is a Prison** (Wired) — Mainstream framing of local-first as counter-narrative to SaaS
  - hash: `6327282:cef591` | https://www.wired.com/story/the-cloud-is-a-prison-can-the-local-first-software-movement-set-us-free/
- **Scoping a Local-First Image Archive** — Design exercise for local-first large-blob storage
  - hash: `6327282:f8781f` | https://www.scottishstoater.com/2025/03/scoping-a-local-first-image-archive/
- **Linear sent me down a local-first rabbit hole** — Hybrid approach: local-first UX, server-authoritative backend
  - hash: `6327282:5a4259` | https://bytemash.net/posts/i-went-down-the-linear-rabbit-hole/

## Zettelkasten & Knowledge Management

- **AgenticMemory: Zettelkasten inspired agentic memory system** — Zettelkasten principles for LLM agent memory
  - hash: `6327282:5a1ea0` | https://github.com/WujiangXu/AgenticMemory
- **A Zettelkasten Explodes Thought** — Rhizomatic, non-hierarchical associative networks; notes traversable from any entry point
  - hash: `6327282:7e8429` | https://writing.bobdoto.computer/inspired-destruction-how-a-zettelkasten-explodes-thoughts-so-you-can-have-newish-ones/
- **Carl Linnaeus's note-taking innovations** — Historical precursor: the index card as atomic, independently-sortable knowledge unit
  - hash: `6327282:a9bd9f` | https://jillianhess.substack.com/p/carl-linnaeuss-note-taking-innovations
- **Semantic note-taking** — Typed relationships and named entities over free prose; richer querying
  - hash: `9372890:c1dbaf` | https://cceckman.com/writing/notes/
- **Fern - CLI Knowledge Management** — Plain Markdown, no GUI dependency; grep-backed search with quiet/verbose modes
  - hash: `9490228:327592` | https://bugwhisperer.dev/blog/fern-cli-knowledge-management/

## Obsidian & PKM Landscape

- **Obsidian Sync headless client** — Sync as daemon, not UI-dependent
  - hash: `6327282:49d777` | https://help.obsidian.md/sync/headless
- **I Use Obsidian** (stephango) — Steph Ango on files-as-data-over-apps philosophy
  - hash: `6327282:4f161a` | https://stephango.com/vault
- **Ditching Obsidian and building my own** — Gap between off-the-shelf PKM and custom storage needs
  - hash: `6327282:7e522b` | https://amberwilliams.io/blogs/building-my-own-pkms
- **Obsidian Bases** — Structured query layer over flat files; signal that blob stores need query surfaces
  - hash: `6327282:a16011` | https://help.obsidian.md/bases
- **Be Careful with Obsidian** — Cautionary take on lock-in; prioritize open formats and portable blobs
  - hash: `6327282:2e689f` | https://phong.bearblog.dev/be-careful-with-obsidian/
- **My Obsidian Note-Taking Workflow** — Template-heavy workflow implying structured metadata schemas
  - hash: `6327282:e84953` | https://www.ssp.sh/blog/obsidian-note-taking-workflow/
- **Why note-taking apps don't make us smarter** — Storage is necessary but not sufficient; reduce friction for retrieval and synthesis
  - hash: `6327282:5a2bc4` | https://www.theverge.com/2023/8/25/23845590/note-taking-apps-ai-chat-distractions-notion-roam-mem-obsidian
- **Mirror Darkly** — Obsidian vault to git repo export; path mapping, idempotent overwrite, link normalization
  - hash: `9490228:8798d8` | https://rygoldstein.com/posts/introducing-mirror-darkly.html
- **Digital Gardening in Obsidian** — Notes as living, evergreen documents; revision over append-only
  - hash: `6327282:848fda` | https://bytes.zone/posts/digital-gardening-in-obsidian/
- **Love Letter to Obsidian** (Karpathy) — AI researcher second brain; signals LLM integration as expected feature
  - hash: `6327282:2b4f64` | https://twitter.com/karpathy/status/1761467904737067456
- **We Should Revisit Literate Programming in the Agent Era** — Literate programming as interface for agent-consumed code
  - hash: `9490228:399791` | https://silly.business/blog/we-should-revisit-literate-programming-in-the-agent-era/
