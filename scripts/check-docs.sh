#!/usr/bin/env sh

# Keep repository knowledge discoverable without requiring a documentation
# platform or a third-party linter. The checks intentionally validate only
# conventions this repository owns.
set -eu

required_files='AGENTS.md ARCHITECTURE.md docs/README.md docs/ARCHITECTURE.md docs/OPERATIONS.md docs/QUALITY.md docs/plans/README.md'

for file in $required_files; do
  if [ ! -f "$file" ]; then
    printf 'Missing required documentation: %s\n' "$file" >&2
    exit 1
  fi
done

for file in AGENTS.md ARCHITECTURE.md $(find docs -type f -name '*.md' | sort); do
  if ! grep -q '^# ' "$file"; then
    printf 'Documentation file must have a level-one heading: %s\n' "$file" >&2
    exit 1
  fi
done

for reference in \
  'docs/README.md:ARCHITECTURE.md' \
  'docs/README.md:OPERATIONS.md' \
  'docs/README.md:QUALITY.md' \
  'docs/README.md:plans/README.md'; do
  file=${reference%%:*}
  target=${reference#*:}
  if ! grep -Fq "($target)" "$file"; then
    printf 'Documentation index must link to %s from %s\n' "$target" "$file" >&2
    exit 1
  fi
done

printf 'Documentation checks passed.\n'
