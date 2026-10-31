"""Read every corpus case with skills-ref (the Agent Skills reference library) and print one JSON reading per line."""
import json
import sys
from pathlib import Path

from skills_ref.parser import find_skill_md, parse_frontmatter, read_properties

LOADER = "skills-ref"


def skill_dir(case_dir: Path) -> Path:
    return next(p for p in sorted(case_dir.iterdir()) if p.is_dir())


def read(case_dir: Path) -> dict:
    d = skill_dir(case_dir)
    out = {"loader": LOADER, "case": case_dir.name, "loaded": False}
    try:
        props = read_properties(d)
        _, body = parse_frontmatter(find_skill_md(d).read_text(encoding="utf-8"))
    except Exception as e:  # any refusal is a reading, not a crash
        out["error"] = f"{type(e).__name__}: {e}".splitlines()[0]
        return out
    out.update(loaded=True, name=props.name, description=props.description, body=body)
    return out


for case in sorted(p for p in Path(sys.argv[1]).iterdir() if p.is_dir()):
    print(json.dumps(read(case)))
