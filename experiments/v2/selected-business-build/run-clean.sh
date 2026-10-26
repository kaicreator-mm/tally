#!/usr/bin/env bash
set -Eeuo pipefail
tmp=''
finish() {
  rc=$?
  echo "T02B_BUILD_EXIT=$rc"
  if [[ -n "$tmp" ]]; then
    if [[ "${KEEP_T02B_WORK:-0}" == "1" ]]; then echo "T02B_WORK=$tmp"
    else rm -rf "$tmp"; fi
  fi
}
trap finish EXIT
: "${DH:?Set DH to exact released DomainHarness git checkout}"
: "${TALLY:?Set TALLY to exact Tally PR checkout}"
: "${TALLY_B:?Set TALLY_B to exact T02a reviewed-source git checkout}"
DH=$(cd "$DH" && pwd -P)
TALLY=$(cd "$TALLY" && pwd -P)
TALLY_B=$(cd "$TALLY_B" && pwd -P)
[[ "$(git -C "$DH" rev-parse HEAD)" == "cee14399d9d0af4e20e34b9111861b16a465b690" ]] || { echo DH_SHA_MISMATCH; exit 2; }
[[ "$(git -C "$TALLY_B" rev-parse HEAD)" == "30a469398f4e50ca3185acea22c0961da854d274" ]] || { echo T02A_SHA_MISMATCH; exit 2; }
[[ "$(node -p 'process.versions.node.split(".")[0]')" == '24' ]] || { echo NODE24_REQUIRED; exit 2; }
BFILE="$TALLY_B/packages/tally-engineering-business/src/index.mjs"
[[ -f "$BFILE" && -f "$TALLY/experiments/v2/selected-business-build/producer.mjs" ]]
tmp=$(mktemp -d)
mkdir -p "$tmp/packed" "$tmp/producer"
echo "DH_RELEASE_SHA=$(git -C "$DH" rev-parse HEAD)"
echo "TALLY_SHA=$(git -C "$TALLY" rev-parse HEAD)"
echo "T02A_SOURCE_SHA=$(git -C "$TALLY_B" rev-parse HEAD)"
node --version
cd "$DH"
npm ci
npm run build
for package in \
  @kaicreator/domain-harness-contracts \
  @kaicreator/domain-harness-build \
  @kaicreator/domain-kernel-basic \
  @kaicreator/domain-standard-basic; do
  npm pack --pack-destination "$tmp/packed" -w "$package" >/dev/null
done
echo T02B_TARBALL_SHA256
sha256sum "$tmp"/packed/*.tgz
cp "$TALLY/experiments/v2/selected-business-build/producer.mjs" "$tmp/producer/producer.mjs"
printf '{"name":"tally-selected-build-producer","private":true,"type":"module"}\n' >"$tmp/producer/package.json"
cd "$tmp/producer"
npm install --save-exact "$tmp"/packed/*.tgz
npm ls --depth=0
TALLY_BUSINESS_SOURCE="$BFILE" node ./producer.mjs
