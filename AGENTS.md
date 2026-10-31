# AGENTS.md

Handoff for a coding agent with no prior context. Read this before changing anything.

## What this is and the claim

SkillParity is a shared corpus of SKILL.md files plus a Go runner. It feeds every file to several real Agent Skills loaders and compares what each loader read (loaded or skipped, `name`, `description`, body) with what the spec says the file means.

The claim: the same SKILL.md is read differently by real loaders today, and a loader can return a wrong description with no error. A shared corpus with spec-derived expectations finds these splits, and adding one more loader costs one registry entry and one small adapter. Every loader that joins gets compared against the same expectations as every other loader. That is the network effect: each new loader or case makes every report more useful to skill authors and loader authors alike.

Agent Skills joined the Agentic AI Foundation (AAIF) on 2026-10-22. The AAIF path for this repo is conformance vectors for the Agent Skills project, next to `skills-ref`.

## Status

Real:
- Five loaders run their published code, pinned:
  - `skills-ref`. Agent Skills reference library (Python), `agentskills/agentskills@69ef37e`, run through `uv`.
  - `tanstack`. `@tanstack/ai-skills@1.0.0` (TypeScript, hand-rolled lenient parser), `parseSkill`.
  - `skill-tools`. `@skill-tools/core@0.2.2` (TypeScript, gray-matter), `parseSkillContent`.
  - `flexigpt`. `github.com/flexigpt/agentskills-go@v0.22.0` (Go, goccy/go-yaml), `document.ParseSkillDocument`.
  - `hivemind`. Hivemind OS `crates/hive-skills/src/parser.rs` at `hivemind-os/hivemind@4b99f02`, vendored verbatim (sha256 checked by `setup.sh`). Only `SkillManifest` is shimmed in `adapters/hivemind/shim`, because the real `hive-contracts` crate pulls the whole workspace.
- 14 corpus cases. Scored expectations are re-derived by `go test` with a YAML 1.2 parse (gopkg.in/yaml.v3) of the block between delimiter lines.

Not real or not covered:
- No agent product is run end to end. Adapters call each library's parse function. A product might pre-process files before calling it.
- No model is called. Nothing is stubbed because nothing needs a model.
- Only frontmatter and body are compared. `metadata`, `allowed-tools`, `license` and `compatibility` are not compared yet.

## Data shape

Name these first when you change anything.

- `Case` (`corpus/<id>/case.json`): `{id, skill_dir, expect: {load: "yes"|"no"|"either", name, description, body}, note}`. The skill lives at `corpus/<id>/<skill_dir>/SKILL.md`. `either` means the spec does not decide. Those cases are reported, never scored.
- `Loader` (`loaders.json`): `{id, what, version, cmd}`. `cmd` is argv. `{corpus}` is replaced with the absolute corpus path.
- `Reading` (one JSON line per case from each adapter on stdout): `{loader, case, loaded, name?, description?, body?, error?}`.
- `Verdict` per case and loader: `MATCH`, `SKIPPED`, `MISREAD` (loaded, a value differs), `REFUSED` (expected load, skipped), `ACCEPTED` (expected skip, loaded), `CONTESTED`.
- `Report` (`report.json`): `{corpus_digest, mode, loaders, cases, cells: [{case, loader, verdict, fields, reading}]}`. `corpus_digest` hashes every corpus path and byte, so two reports only compare when they ran the same files.

Comparison forgives CRLF vs LF and surrounding whitespace. Nothing else.

## Repo map

- `cmd/skillparity/main.go`. Runner. Reads cases and the registry, runs each adapter, judges, prints the matrix, writes `report.json`.
- `cmd/skillparity/corpus_test.go`. Re-derives expectations with YAML 1.2. Unit tests for `judge`.
- `corpus/`. 14 cases. `c08-crlf` and `c09-bom` contain raw CRLF and a BOM on purpose. Do not let an editor normalize them.
- `loaders.json`. Loader registry.
- `adapters/skills-ref/read.py`, `adapters/js/read.mjs` (both TS loaders), `adapters/flexigpt/main.go`, `adapters/hivemind/` (Rust crate, vendored parser, shim).
- `setup.sh`. Installs pinned loaders. `demo.sh`. The 9 checks.

## Setup, run, verify (fresh clone)

Needs Go 1.26+, Node 20+ with npm, uv, a Rust toolchain with cargo, jq, network access for the first setup.

```bash
git clone https://github.com/ideator-labs/skillparity && cd skillparity
bash setup.sh            # ends with "setup ok"
go test ./...            # ok
bash demo.sh             # ends with "9/9 passed", exit 0
bash demo.sh --break     # ends with "6/9 passed", exit 1 (checks 5, 7, 9 fail)
```

`--break` runs the same loaders but judges them on load or skip only, the way a "does it parse" validator does. The silent misreads then look like passes, so the checks that depend on comparing values fail. That is the point of the repo.

To see one loader's raw readings: `go run ./adapters/flexigpt corpus` or `node adapters/js/read.mjs tanstack corpus`.

## Measured results (2026-10-31, pinned versions above)

- `c02` quoted `---` in description. skills-ref refuses ("Invalid YAML"). Reported upstream as [agentskills#633](https://github.com/agentskills/agentskills/issues/633). The other four read it.
- `c14` indented `---` inside a literal block. skills-ref and flexigpt load the skill with the description cut to `Read receipts split by a line of dashes:` and no error. The rest of the description lands in the body. skills-ref splits on the bare substring. flexigpt trims each line before comparing it to `---`.
- `c06` two `description` keys. tanstack and hivemind load the second value. skills-ref, skill-tools and flexigpt refuse. A reviewer running a strict validator never sees the description a lenient runtime will use.
- `c04` and `c05`. tanstack keeps `''` literally and keeps ` # owner: finance team` inside the description. Its parser takes the rest of the line after the first colon.
- `c09` BOM (contested). skills-ref and hivemind skip, the other three load.
- `c10` unquoted colon (contested). tanstack and hivemind load through their fallbacks, the other three skip.
- 7 of 12 scored cases read the same in all five loaders.

## Design decisions

- Expectations follow the spec plus the client implementation guide (`docs/client-implementation/adding-skills-support.mdx` in agentskills/agentskills): delimiter lines, YAML parse, warn-and-load on name issues, skip on missing description or unparseable YAML. Where the guide and the YAML rules disagree, the case is `either`.
- Adapters are separate processes in each loader's own language. Reimplementing a parser would test the reimplementation.
- The registry is data. Adding a loader never touches the runner.
- `demo.sh` asserts the measured verdicts. When a loader fixes a bug, the demo should fail. Then update the check and the table in README.md together.

## Known limits

- Loader choice is what had a callable parse function on 2026-10-31. Big clients (Claude Code, Codex, goose, Gemini CLI, Cursor) are not in the table yet.
- skill-tools is a linter and rejects some files a lenient runtime loads. It agrees on every scored case here, but on name-format cases it would REFUSE by design.
- `ACCEPTED` on `c06` assumes YAML 1.2 duplicate-key rules. Some will argue last-wins is fine. The safety argument is that two tools disagree about which description is real.
- Why serde_yaml in hivemind keeps the last duplicate key is a guess (map insert overwrites). Not traced.

## Next steps (prioritized)

1. Add goose's loader. Done when `loaders.json` has a `goose` entry pinned to an `aaif-goose/goose` commit and `bash demo.sh` prints a 6-column matrix with 84 readings.
2. Compare `metadata` and `allowed-tools`. Done when `Reading` and `Expect` carry both, a case with nested metadata ([agentskills#514](https://github.com/agentskills/agentskills/issues/514)) exists, and at least one loader's verdict on it is printed.
3. Publish the corpus as a standalone vector set with `corpus_digest` in a VERSION file. Done when a second repo can run `go run github.com/ideator-labs/skillparity/cmd/skillparity -corpus <path>` against its own registry and get the same digest.
4. Open upstream issues for c14 (skills-ref, flexigpt), c06 and c04/c05 (tanstack), c06 (hivemind). Done when each issue links a case id and its report cell. Ask the user before filing anything.
5. Add a GitHub Action that runs a loader's adapter on PRs. Done when a sample workflow in `.github/workflows` fails on a MISREAD.

## Working rules

- Never edit a vendored parser. Re-vendor at a new commit and update the sha256 in `setup.sh`.
- Every new case needs `case.json` that passes `go test`, or `load: "either"` with a note saying why the spec does not decide.
- Pin every loader version in `loaders.json` and in the adapter's lockfile.
- Commits as `danielgerlag <daniel@gerlag.ca>`.

## Context and sources

- Agent Skills spec and client guide: https://github.com/agentskills/agentskills (docs/specification.mdx, docs/client-implementation/adding-skills-support.mdx).
- agentskills#633 (2026-10-31): https://github.com/agentskills/agentskills/issues/633
- agentskills#514 (metadata types): https://github.com/agentskills/agentskills/issues/514
- AAIF announcement of Agent Skills joining (2026-10-22): https://aaif.io/news/agentic-ai-foundation-announces-four-new-projects
- Loaders: https://www.npmjs.com/package/@tanstack/ai-skills , https://www.npmjs.com/package/@skill-tools/core , https://github.com/flexigpt/agentskills-go , https://github.com/hivemind-os/hivemind
