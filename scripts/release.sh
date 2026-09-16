#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

tag=${1:?usage: bash scripts/release.sh vX.Y.Z}
[[ "$tag" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$ ]] || exit 1
version=${tag#v}
repo=BrokkAi/simplifier-bot
dist=$(mktemp -d)
trap 'rm -rf "$dist"' EXIT
npm_tag=latest
release_flags=()
if [[ "$version" == *-* ]]; then
    npm_tag=next
    release_flags+=(--prerelease)
fi

gh release create "$tag" --repo "$repo" --verify-tag --generate-notes "${release_flags[@]}"
test -n "$(gh release view "$tag" --repo "$repo" --json url --jq .url)"

for os in linux darwin; do
    for arch in amd64 arm64; do
        cpu=$arch
        [[ "$arch" != amd64 ]] || cpu=x64
        package="$dist/$os-$cpu"
        mkdir -p "$package/bin"
        CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build -trimpath \
            -ldflags "-s -w -X main.version=$tag" -o "$package/bin/bsb" ./cmd/bsb
        cp LICENSE NOTICE "$package/"
        cp -R licenses "$package/"
        node - "$package" "$version" "$os" "$cpu" <<'JS'
const fs = require('node:fs');
const [dir, version, os, cpu] = process.argv.slice(2);
fs.writeFileSync(`${dir}/package.json`, JSON.stringify({
  name: `@brokkai/simplifier-bot-${os}-${cpu}`, version,
  os: [os], cpu: [cpu], license: 'Apache-2.0',
  repository: 'github:BrokkAi/simplifier-bot',
  files: ['bin', 'LICENSE', 'NOTICE', 'licenses']
}, null, 2));
JS
        asset="brokk-simplifier-bot-$tag-$os-$arch.tar.gz"
        tar -czf "$dist/$asset" -C "$package/bin" bsb -C "$package" LICENSE NOTICE licenses
        (cd "$dist" && shasum -a 256 "$asset") >> "$dist/checksums.txt"
        gh release upload "$tag" "$dist/$asset" --repo "$repo"
    done
done
gh release upload "$tag" "$dist/checksums.txt" --repo "$repo"
test "$(gh release view "$tag" --repo "$repo" --json assets --jq '.assets | length')" = 5

for os in linux darwin; do
    for cpu in x64 arm64; do
        (cd "$dist/$os-$cpu" && npm publish --access public --tag "$npm_tag")
    done
done

mkdir -p "$dist/launcher"
cp npm/bsb.cjs LICENSE NOTICE README.md "$dist/launcher/"
node - "$dist/launcher" "$version" <<'JS'
const fs = require('node:fs');
const [dir, version] = process.argv.slice(2);
const optionalDependencies = {};
for (const os of ['linux', 'darwin']) {
  for (const cpu of ['x64', 'arm64']) {
    optionalDependencies[`@brokkai/simplifier-bot-${os}-${cpu}`] = version;
  }
}
fs.writeFileSync(`${dir}/package.json`, JSON.stringify({
  name: '@brokkai/simplifier-bot', version, license: 'Apache-2.0',
  repository: 'github:BrokkAi/simplifier-bot',
  bin: {bsb: 'bsb.cjs'}, files: ['bsb.cjs', 'LICENSE', 'NOTICE', 'README.md'],
  optionalDependencies
}, null, 2));
JS
(cd "$dist/launcher" && npm publish --access public --tag "$npm_tag")
