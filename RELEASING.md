# Releasing simplifier-bot

Push a version tag on the commit to release:

```sh
git tag v0.2.2
git push origin v0.2.2
```

The tag workflow runs `scripts/release.sh`: create the GitHub release, build and
upload Linux/macOS amd64/arm64 archives and installer checksums, then publish the
four native npm packages and the launcher. Versions with a prerelease suffix use
npm's `next` tag and GitHub's prerelease flag.

npm uses trusted publishing for `publish-packages.yml` in the `packages-publish`
environment. All five packages need that publisher configured.
