# 에이전트 작업 규칙

- 작업 전에 `site/content/docs/contributing.md`를 읽으세요. 개발 환경, 검사 명령, 커밋 규칙이 있습니다.
- 기능 코드를 바꾸면, 같은 브랜치에서 `site/content/docs/`의 해당 페이지도 고치세요.
- 문서를 바꿀 필요가 없으면, 커밋 메시지에 `[no-docs]`와 이유를 쓰세요.
- `git push`와 `gh pr create` 전에 훅이 `scripts/check-docs.sh`를 실행합니다. 훅이 실패하면, 위 두 규칙 중 하나를 따르세요.
- 문서는 쉬운 한국어로 쓰세요. 규칙은 `contributing.md`의 "쓰기 규칙"에 있습니다.
