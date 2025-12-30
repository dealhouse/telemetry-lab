#!/usr/bin/env bash
set -euo pipefail

required=(
  "README.md"
  "docs/adr/0001-tech-stack.md"
  "docs/diagrams/component.puml"
  "docs/diagrams/sequence_ingest_to_finding.puml"
  ".github/PULL_REQUEST_TEMPLATE.md"
)

for f in "${required[@]}"; do
  if [[ ! -f "$f" ]]; then
    echo "Missing required file: $f"
    exit 1
  fi
done

echo "Meta checks passed."
