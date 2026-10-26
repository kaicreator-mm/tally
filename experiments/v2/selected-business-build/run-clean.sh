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
: "${TALLY_B:?Set TALLY_B to Tally v2 merged B source git checkout}"
DH=$(cd "$DH" && pwd -P)
TALLY=$(cd "$TALLY" && pwd -P)
TALLY_B=$(cd "$TALLY_B" && pwd -P)
[[ "$(git -C "$DH" rev-parse HEAD)" == "cee14399d9d0af4e20e34b9111861b16a465b690" ]] || { echo DH_SHA_MISMATCH; exit 2; }
# This integration PR exercises the exact proposed Business bytes, not the
# old merged B. CI uses GitHub's synthetic PR merge checkout for both paths.
[[ "$(git -C "$TALLY_B" rev-parse HEAD)" == "$(git -C "$TALLY" rev-parse HEAD)" ]] || { echo T04A_SOURCE_CHECKOUT_MISMATCH; exit 2; }
[[ "$(node -p 'process.versions.node.split(".")[0]')" == '24' ]] || { echo NODE24_REQUIRED; exit 2; }
BFILE="$TALLY_B/packages/tally-engineering-business/src/index.mjs"
[[ -f "$BFILE" && -f "$TALLY/experiments/v2/selected-business-build/producer.mjs" ]]
tmp=$(mktemp -d)
mkdir -p "$tmp/packed" "$tmp/producer" "$tmp/consumer"
echo "DH_RELEASE_SHA=$(git -C "$DH" rev-parse HEAD)"
echo "TALLY_SHA=$(git -C "$TALLY" rev-parse HEAD)"
echo "T02A_SOURCE_SHA=$(git -C "$TALLY_B" rev-parse HEAD)"
node --version
cd "$DH"
npm ci
npm run build
# Contracts/BUILD are root npm workspaces, selected K/S Domain Packages
# are independent package directories and are intentionally NOT workspaces.
for package in @kaicreator/domain-harness-contracts @kaicreator/domain-harness-build @kaicreator/domain-harness-microkernel; do
  npm pack --pack-destination "$tmp/packed" -w "$package" >/dev/null
done
for dir in "$DH/domain-packages/kernel/basic" "$DH/domain-packages/standard/basic"; do
  npm pack --pack-destination "$tmp/packed" "$dir" >/dev/null
done
echo T02B_TARBALL_SHA256
sha256sum "$tmp"/packed/*.tgz
cp "$TALLY/experiments/v2/selected-business-build/producer.mjs" "$tmp/producer/producer.mjs"
cp "$TALLY/experiments/v2/selected-business-build/readiness-capability.mjs" "$tmp/producer/readiness-capability.mjs"
printf '{"name":"tally-selected-build-producer","private":true,"type":"module"}\n' >"$tmp/producer/package.json"
cd "$tmp/producer"
npm install --save-exact "$tmp"/packed/kaicreator-domain-harness-build-*.tgz "$tmp"/packed/kaicreator-domain-harness-contracts-*.tgz "$tmp"/packed/kaicreator-domain-kernel-basic-*.tgz "$tmp"/packed/kaicreator-domain-standard-basic-*.tgz
npm ls --depth=0
T02B_SEALED_APP_OUTPUT="$tmp/consumer/sealed-app.json" TALLY_BUSINESS_SOURCE="$BFILE" node ./producer.mjs
# A new Node process in an independent npm root consumes only public
# Contracts + Microkernel, never BUILD/Compiler/unselected Domain Packages.
cp "$TALLY/experiments/v2/selected-business-build/consumer.mjs" "$tmp/consumer/consumer.mjs"
printf '{"name":"tally-selected-runtime-consumer","private":true,"type":"module"}\n' >"$tmp/consumer/package.json"
cd "$tmp/consumer"
npm install --save-exact "$tmp"/packed/kaicreator-domain-harness-contracts-*.tgz "$tmp"/packed/kaicreator-domain-harness-microkernel-*.tgz
npm ls --depth=0
node ./consumer.mjs
