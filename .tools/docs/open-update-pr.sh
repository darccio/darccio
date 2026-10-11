#!/usr/bin/env bash
# Runs only after docs-update and the complete guarded build have succeeded.
set -euo pipefail
case "$SOURCE_REPOSITORY" in
  darccio/mergo|darccio/zas|darccio/darna) ;;
  *) printf '%s\n' 'Unsupported documentation source.' >&2; exit 1 ;;
esac
[[ "$SOURCE_COMMIT" =~ ^[0-9a-f]{40}$ ]] || exit 1
if git diff --quiet -- .zas/projects.yml; then
  printf '%s\n' 'Documentation pin is already current.'
  exit 0
fi
project=${SOURCE_REPOSITORY#darccio/}
branch="bot/docs-$project-$SOURCE_COMMIT"
base_commit=$(git rev-parse HEAD)
if ! gh api "repos/darccio/darccio/git/ref/heads/$branch" >/dev/null 2>&1; then
  jq -n --arg ref "refs/heads/$branch" --arg sha "$base_commit" \
    '{ref:$ref, sha:$sha}' > "$RUNNER_TEMP/docs-branch.json"
  gh api --method POST repos/darccio/darccio/git/refs --input "$RUNNER_TEMP/docs-branch.json" >/dev/null
fi
current=$(gh api "repos/darccio/darccio/contents/.zas/projects.yml?ref=$branch" --jq '.sha')
proposed=$(git hash-object .zas/projects.yml)
# Contents API commits only this reviewed manifest, never fetched source files.
if [[ "$current" != "$proposed" ]]; then
jq -n --arg branch "$branch" --arg sha "$current" \
  --arg message "build(docs): update $project source pin" \
  --arg content "$(base64 -w0 .zas/projects.yml)" \
  '{branch:$branch, sha:$sha, message:$message, content:$content}' > "$RUNNER_TEMP/docs-content.json"
gh api --method PUT repos/darccio/darccio/contents/.zas/projects.yml \
  --input "$RUNNER_TEMP/docs-content.json" >/dev/null
fi
if [[ $(gh pr list --repo darccio/darccio --head "$branch" --state open --json number --jq 'length') == 0 ]]; then
  cat > "$RUNNER_TEMP/docs-pr.md" <<BODY
Update the reviewed $project documentation source to $SOURCE_COMMIT.

The update workflow fetched only allowlisted Markdown and the original license, rendered them as data, and passed the guarded site build with all vanity-import and metadata checks. Merging this pin update triggers Cloudflare Pages through the existing main-branch Git integration.
BODY
  gh pr create --repo darccio/darccio --base main --head "$branch" \
    --title "Docs: update $project source" --body-file "$RUNNER_TEMP/docs-pr.md"
fi
# GITHUB_TOKEN-created PRs do not trigger ordinary push/PR workflows. Explicit
# dispatch attaches Site checks to this branch without requiring another secret.
gh workflow run site-checks.yml --repo darccio/darccio --ref "$branch"
