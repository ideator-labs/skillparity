// Read every corpus case with one JS loader and print one JSON reading per line.
// Usage: node read.mjs <tanstack|skill-tools> <corpus-dir>
import { readFileSync, readdirSync, statSync } from 'node:fs'
import { join } from 'node:path'
import { parseSkill } from '@tanstack/ai-skills'
import { parseSkillContent } from '@skill-tools/core'

const loaders = {
  tanstack(raw, dirName) {
    const s = parseSkill(raw, { dirName })
    return { name: s.metadata.name, description: s.metadata.description, body: s.body }
  },
  'skill-tools'(raw, dirName, dir) {
    const r = parseSkillContent(raw, join(dir, 'SKILL.md'), dir)
    if (!r.ok) {
      const e = r.diagnostics.find((d) => d.severity === 'error')
      throw new Error(e ? `${e.ruleId}: ${e.message}` : 'not ok')
    }
    return { name: r.skill.metadata.name, description: r.skill.metadata.description, body: r.skill.body }
  },
}

const [id, root] = process.argv.slice(2)
const load = loaders[id]
if (!load) throw new Error(`unknown loader ${id}`)

for (const c of readdirSync(root).sort()) {
  const caseDir = join(root, c)
  if (!statSync(caseDir).isDirectory()) continue
  const dirName = readdirSync(caseDir).sort().find((n) => statSync(join(caseDir, n)).isDirectory())
  const dir = join(caseDir, dirName)
  const out = { loader: id, case: c, loaded: false }
  try {
    Object.assign(out, load(readFileSync(join(dir, 'SKILL.md'), 'utf8'), dirName, dir), { loaded: true })
  } catch (e) {
    out.error = String(e.message ?? e).split('\n')[0]
  }
  console.log(JSON.stringify(out))
}
