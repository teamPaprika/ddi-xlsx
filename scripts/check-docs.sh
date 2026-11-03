#!/usr/bin/env bash
# Fails when feature code changed since the base branch but the docs did not.
# CI runs it on every pull request. Claude Code and Codex run it with --hook
# before each shell command; the hook reads the tool call JSON on stdin and
# only checks `git push` and `gh pr create`. Exit 2 blocks the agent's call.
set -euo pipefail

if [[ ${1:-} == --hook ]]; then
  grep -Eq 'git( -C [^ ]+)? push|gh pr create' || exit 0
fi
cd "$(git rev-parse --show-toplevel)"

base=${DOCS_BASE:-origin/main}
git rev-parse -q --verify "$base^{commit}" >/dev/null || exit 0

if git log --format=%B "$base..HEAD" | grep -qF '[no-docs]'; then
  exit 0
fi

changed=$(git diff --name-only "$base...HEAD")
code=$(grep -E '^((cmd|ddi|workbook|kosac|wasm)/.*\.go|npm/index\.(js|d\.ts)|kosac/xsd/.*\.xsd|kosac/template/.*\.xlsx)$' <<<"$changed" | grep -v '_test\.go$' || true)
[[ -z $code ]] && exit 0
grep -Eq '^(site/content/|README\.md$)' <<<"$changed" && exit 0

cat >&2 <<EOF
문서 검사 실패: 기능 코드가 바뀌었지만 문서가 바뀌지 않았습니다.

바뀐 기능 코드($base 기준):
$(sed 's/^/  /' <<<"$code")

아래 중 하나를 하세요.
1. 바뀐 동작을 site/content/docs/의 해당 페이지에 쓰고 커밋하세요.
2. 문서를 바꿀 필요가 없으면, 커밋 메시지에 [no-docs]와 이유를 쓰세요.

규칙: site/content/docs/contributing.md의 "문서 규칙"
EOF
exit 2
