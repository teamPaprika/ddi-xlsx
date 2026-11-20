#!/usr/bin/env bash
# Fails when feature code changed since the base branch but the docs did not.
# CI runs it on every pull request and push to main. Claude Code and Codex run it with --hook
# before each shell command; the hook reads the tool call JSON on stdin and
# only checks `git push` and `gh pr create`. Exit 2 blocks the agent's call.
set -euo pipefail

if [[ ${1:-} == --hook ]]; then
  grep -Eq 'git( -C [^ ]+)? push|gh pr create' || exit 0
fi
cd "$(git rev-parse --show-toplevel)"

base=${DOCS_BASE:-origin/main}
if ! git rev-parse -q --verify "$base^{commit}" >/dev/null; then
  printf '문서 검사 실패. 비교 기준 %s의 커밋을 찾을 수 없습니다.\n' "$base" >&2
  exit 2
fi

changed=$(git diff --name-only "$base...HEAD" --)
docs=$(git diff --name-only --diff-filter=AMR "$base...HEAD" --)
code=$(grep -E '^((cmd|ddi|workbook|kosac|wasm)/.*\.go|npm/index\.(js|d\.ts)|kosac/xsd/.*\.xsd|kosac/template/.*\.xlsx|kosac/check/.*\.txt)$' <<<"$changed" | grep -v '_test\.go$' || true)

check_docs() {
  local code_pattern=$1 docs_pattern=$2 docs_path=$3
  local paths=() path commits commit exempt=true
  while IFS= read -r path; do
    [[ -n $path ]] && paths+=("$path")
  done <<<"$(grep -E "$code_pattern" <<<"$code" || true)"
  [[ ${#paths[@]} -gt 0 ]] || return 0
  grep -Eq "$docs_pattern" <<<"$docs" && return 0

  commits=$(git log --full-history --no-merges --format=%H "$base..HEAD" -- "${paths[@]}")
  if [[ -n $commits ]]; then
    while IFS= read -r commit; do
      if ! grep -Fxq '[no-docs]' <<<"$(git show -s --format=%B "$commit")"; then
        exempt=false
        break
      fi
    done <<<"$commits"
    [[ $exempt == true ]] && return 0
  fi

  printf '문서 검사 실패. %s 문서를 업데이트하세요.\n' "$docs_path" >&2
  printf '  %s\n' "${paths[@]}" >&2
  printf '문서 수정이 필요 없는 코드 커밋에는 [no-docs]와 이유를 별도 줄에 적으세요.\n' >&2
  return 2
}

check_docs '^kosac/check/' '^site/content/docs/kosac/validation\.md$' '재단 검사(validation.md)'
check_docs '^kosac/' '^site/content/docs/kosac/' '재단 코드북(site/content/docs/kosac/)'
check_docs '^(wasm/|npm/)' '^site/content/docs/npm\.md$' 'SDK(npm.md)'
check_docs '^(cmd|ddi|workbook)/' '^(site/content/docs/|README\.md$)' '사용자 안내'
