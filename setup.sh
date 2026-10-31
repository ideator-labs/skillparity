#!/usr/bin/env bash
# One-time setup: installs the pinned loaders. Safe to rerun.
set -euo pipefail
cd "$(dirname "$0")"

for tool in go node npm uv cargo jq; do
  command -v "$tool" >/dev/null || { echo "missing: $tool" >&2; exit 1; }
done

echo "hivemind parser.rs is the pinned upstream file"
echo "590c7047805b4ce12ab736c1d05ab40f75c9ea8ceeb4796533c6affd7ebdfac9  adapters/hivemind/vendor/parser.rs" | sha256sum -c -

(cd adapters/js && npm ci --no-audit --no-fund --silent)
(cd adapters/hivemind && cargo build --release --locked --quiet)
go build ./...
uv run --no-project --quiet \
  --with 'skills-ref @ git+https://github.com/agentskills/agentskills@69ef37e9424c0a7ea9dd2293b559e43ec8176379#subdirectory=skills-ref' \
  python -c 'import skills_ref; print("skills-ref ready")'
echo "setup ok"
