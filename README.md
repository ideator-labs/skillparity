# SkillParity

One SKILL.md corpus, every Agent Skills loader, one table of who reads what.

The [Agent Skills](https://agentskills.io) format is now an AAIF project and dozens of clients load it. Each client writes its own frontmatter parser. This repo runs the same 14 SKILL.md files through five real loaders and checks what each one read against what the spec says the file means.

Measured on 2026-10-31 with pinned versions:

| case | skills-ref | tanstack | skill-tools | flexigpt | hivemind |
|---|---|---|---|---|---|
| quoted `---` in description ([#633](https://github.com/agentskills/agentskills/issues/633)) | REFUSED | ok | ok | ok | ok |
| `''` in a single-quoted description | ok | MISREAD | ok | ok | ok |
| ` # comment` after the description | ok | MISREAD | ok | ok | ok |
| two `description` keys | skip | ACCEPTED (2nd) | skip | skip | ACCEPTED (2nd) |
| indented `---` inside a `\|` block | MISREAD (truncated) | ok | ok | MISREAD (truncated) | ok |

Only 7 of 12 scored cases read the same in all five loaders. The other two cases (BOM, unquoted colon) are contested by the spec and only reported.

```bash
bash setup.sh   # needs go, node, npm, uv, cargo, jq
bash demo.sh    # 9 pass/fail checks, exit 0 on pass
```

Start with [AGENTS.md](AGENTS.md). It has the data shape, the loader list with pins, how to add a loader, the limits, and next steps.

MIT licensed. `adapters/hivemind/vendor/parser.rs` is copied verbatim from [hivemind-os/hivemind](https://github.com/hivemind-os/hivemind) (MIT).
