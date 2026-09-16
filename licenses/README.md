# Licensing and third-party notices

Brokk Simplifier Bot uses [Apache-2.0](../LICENSE). [NOTICE](../NOTICE)
identifies the project; [THIRD_PARTY_NOTICES.txt](THIRD_PARTY_NOTICES.txt)
contains the bundled dependency license texts and Go runtime and Unicode notices.

## Maintenance

Maintain the notices as ordinary documentation. When adding or removing bundled
material, or when upstream license terms or attribution change, update the
relevant section of `THIRD_PARTY_NOTICES.txt`. A routine dependency or Go version
upgrade does not require approval, license hashes, or a separate policy update.
Update version and source references in the notices when they change.

There is no license scanner or CI approval gate. The supplemental Go and Unicode
text files are retained as source copies for maintaining the notices.

## Distribution

`scripts/release.sh` automatically includes `LICENSE`, `NOTICE`, and `licenses/`
in native archives and all npm packages. Consumers can keep these accompanying
files when redistributing the bundled code; no runtime prompts are needed.

Separately installed coding agents, Node.js, and Python interpreters are not
bundled in these distributions.
