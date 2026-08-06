#!/usr/bin/env bash
# Enforces the package layering rules from docs/arch/03-project-structure.md.
# A red result here means a layer was crossed — fix the direction, don't suppress.
set -euo pipefail

mod=github.com/yurifrl/tennon
fail=0

echo "== domain must import only the standard library =="
imports=$(go list -f '{{join .Imports "\n"}}' ./domain/ | grep -E 'github.com|k8s.io|database/sql|net/http' || true)
if [[ -n "$imports" ]]; then
  echo "FAIL: domain imports infrastructure:"; echo "$imports"; fail=1
else
  echo "OK: domain is pure"
fi

echo "== delivery packages (api, proxy, controller, cli) must not import each other =="
for p in api proxy controller cli; do
  hit=$(go list -f '{{join .Imports "\n"}}' ./$p/ | grep -E "$mod/(api|proxy|controller|cli)\$" || true)
  if [[ -n "$hit" ]]; then
    echo "FAIL: $p imports a sibling delivery package:"; echo "$hit"; fail=1
  else
    echo "OK: $p"
  fi
done

echo "== infrastructure (store, runtime, secret, auth) must import only domain =="
for p in store runtime secret auth; do
  hit=$(go list -f '{{join .Imports "\n"}}' ./$p/ | grep -E "$mod/(api|proxy|controller|cli|store|runtime|secret|auth|web)\$" | grep -v "$mod/domain\$" || true)
  if [[ -n "$hit" ]]; then
    echo "FAIL: $p imports a non-domain internal package:"; echo "$hit"; fail=1
  else
    echo "OK: $p"
  fi
done

echo "== import graph must be acyclic =="
if go list -deps ./... >/dev/null 2>&1; then
  echo "OK: acyclic"
else
  echo "FAIL: import cycle"; fail=1
fi

exit $fail
