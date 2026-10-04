#!/usr/bin/env bash
set -euo pipefail

module_path=$(go list -m -f '{{.Path}}')

check_dependencies() {
  local target=$1
  local approved_package=${2:-}
  local dependencies package
  dependencies=$(go list -deps -f '{{if not .Standard}}{{.ImportPath}}{{end}}' "$target")
  while IFS= read -r package; do
    case "$package" in
      ""|"$module_path"|"$module_path"/*) ;;
      *)
        if [ -z "$approved_package" ] || [ "$package" != "$approved_package" ]; then
          echo "Unreviewed third-party dependency in $target: $package" >&2
          return 1
        fi
        ;;
    esac
  done <<< "$dependencies"
}

# The distributed command still uses only the standard library. Library
# projections also use the reviewed Go coverage parser, and no other package.
check_dependencies ./cmd/testtranslator
check_dependencies ./... golang.org/x/tools/cover

# Retain the actual upstream license and patent notice, and detect any changes
# when the reviewed dependency version is updated.
tools_dir=$(go list -m -f '{{.Dir}}' golang.org/x/tools)
test -n "$tools_dir"
cmp "$tools_dir/LICENSE" .github/licenses/golang.org-x-tools/LICENSE
cmp "$tools_dir/PATENTS" .github/licenses/golang.org-x-tools/PATENTS
