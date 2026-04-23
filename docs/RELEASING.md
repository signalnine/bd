# Release Process

Cutting a new bd release.

## Preconditions

- `main` is green on CI
- Working tree is clean: `git status` shows nothing
- You're the maintainer (the release workflow only runs on the canonical repo)

## Version bump

```bash
./scripts/update-versions.sh X.Y.Z
```

This updates every version string in the tree (Go, marketplace.json, plugin.json, nix derivation, etc). Verify with:

```bash
./scripts/check-versions.sh
```

If any file is out of sync, fix it manually and re-run `check-versions.sh`.

## Release

1. Commit the version bump.
2. Tag and push:
   ```bash
   git tag vX.Y.Z
   git push
   git push origin vX.Y.Z
   ```
3. The release workflow (`.github/workflows/release.yml`) runs goreleaser for Linux/Windows/Android/FreeBSD, then a separate job for macOS native builds. Total runtime: ~15 min.
4. Don't poll. GitHub's API rate limit is 5000/hr and `gh run watch` can easily burn it. Wait 15 min, then one `gh run view <run-id>` check.
5. On success, release artifacts are on https://github.com/signalnine/bd/releases/tag/vX.Y.Z.

## If CI fails

Never reuse a failed tag. Bump to the next patch version, fix, and re-tag:

```bash
./scripts/update-versions.sh X.Y.(Z+1)
# ... fix whatever broke ...
git tag vX.Y.(Z+1)
git push origin vX.Y.(Z+1)
```

## Verification

Install the release binary in a scratch dir and smoke-test:

```bash
curl -fsSL https://raw.githubusercontent.com/signalnine/bd/main/scripts/install.sh | bash
bd version
cd $(mktemp -d)
git init -q && git commit --allow-empty -qm init
bd init --prefix smoke -q
bd create "smoke test" -t task -p 1
bd list
```

## Release notes

GoReleaser builds release notes from commit messages following Conventional Commits (`feat:`, `fix:`). For significant releases, edit the GitHub release on the web and expand the notes.
