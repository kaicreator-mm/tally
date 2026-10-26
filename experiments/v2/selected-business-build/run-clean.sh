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
# Immediately remove host credential from the producer/npm/build environment.
# Re-export only for the final isolated read-only native consumer process.
native_token="${TALLY_GITHUB_READ_TOKEN:-}"
unset TALLY_GITHUB_READ_TOKEN
if [[ "${T04B_NATIVE:-0}" == "1" ]]; then
  [[ -n "$native_token" ]] || { echo T04B_NATIVE_HOST_TOKEN_REQUIRED; exit 3; }
  [[ "${TALLY_NATIVE_EXPECTED_SHA:-}" =~ ^[0-9a-f]{40}$ ]] || { echo T04B_SOURCE_PIN_REQUIRED; exit 3; }
fi
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
for package in @kaicreator/domain-harness-contracts @kaicreator/domain-harness-build @kaicreator/domain-harness-microkernel @kaicreator/domain-harness-host-node; do
  npm pack --pack-destination "$tmp/packed" -w "$package" >/dev/null
done
for dir in "$DH/domain-packages/kernel/basic" "$DH/domain-packages/standard/basic"; do
  npm pack --pack-destination "$tmp/packed" "$dir" >/dev/null
done
echo T02B_TARBALL_SHA256
sha256sum "$tmp"/packed/*.tgz
cp "$TALLY/experiments/v2/selected-business-build/producer.mjs" "$tmp/producer/producer.mjs"
if [[ "${T04B_NATIVE:-0}" == "1" ]]; then
  # Install ONLY reviewed existing parser/provider dependencies; bundle source,
  # do not rebuild Octokit, GitHub REST, YAML or JCS.
  npm --prefix "$TALLY/packages/tally-ads-v5-validator" install --ignore-scripts --no-audit --no-fund
  npm --prefix "$TALLY/packages/tally-github-native-c" install --ignore-scripts --no-audit --no-fund
  cp "$TALLY/experiments/v2/selected-business-build/bundle-native.mjs" "$tmp/producer/bundle-native.mjs"
  cp "$TALLY/experiments/v2/selected-business-build/native-readonly-shim.mjs" "$tmp/producer/native-readonly-shim.mjs"
fi
cp "$TALLY/experiments/v2/selected-business-build/readiness-capability.mjs" "$tmp/producer/readiness-capability.mjs"
printf '{"name":"tally-selected-build-producer","private":true,"type":"module"}\n' >"$tmp/producer/package.json"
cd "$tmp/producer"
npm install --save-exact "$tmp"/packed/kaicreator-domain-harness-build-*.tgz "$tmp"/packed/kaicreator-domain-harness-contracts-*.tgz "$tmp"/packed/kaicreator-domain-kernel-basic-*.tgz "$tmp"/packed/kaicreator-domain-standard-basic-*.tgz
npm ls --depth=0
if [[ "${T04B_NATIVE:-0}" == "1" ]]; then
  npm install --save-exact --ignore-scripts --no-audit --no-fund esbuild@0.25.12
  T04B_C_OUTPUT="$tmp/producer/native-bundled.mjs" T04B_IMPORTS_OUTPUT="$tmp/producer/native-imports.json" TALLY="$TALLY" node ./bundle-native.mjs
  # Existing reviewed native provider (which may include CJS in upstream npm
  # deps) is an installed, declared closed-world npm package, NOT in sealed
  # moduleSource. The sealed C bytes only static re-export this npm package.
  mkdir -p "$tmp/native-provider"
  cp "$tmp/producer/native-bundled.mjs" "$tmp/native-provider/index.mjs"
  printf '{"name":"@kaicreator/tally-native-reader-bundle","version":"0.0.0","private":true,"type":"module","exports":"./index.mjs"}\\n' > "$tmp/native-provider/package.json"
  npm pack --pack-destination "$tmp/packed" "$tmp/native-provider" >/dev/null
  T04B_NATIVE=1 T02B_SEALED_APP_OUTPUT="$tmp/consumer/sealed-app.json" TALLY_BUSINESS_SOURCE="$BFILE" node ./producer.mjs
else
  T02B_SEALED_APP_OUTPUT="$tmp/consumer/sealed-app.json" TALLY_BUSINESS_SOURCE="$BFILE" node ./producer.mjs
fi
# A new Node process in an independent npm root consumes only public
# Contracts + Microkernel, never BUILD/Compiler/unselected Domain Packages.
cp "$TALLY/experiments/v2/selected-business-build/consumer.mjs" "$tmp/consumer/consumer.mjs"
printf '{"name":"tally-selected-runtime-consumer","private":true,"type":"module"}\n' >"$tmp/consumer/package.json"
cd "$tmp/consumer"
npm install --save-exact "$tmp"/packed/kaicreator-domain-harness-contracts-*.tgz "$tmp"/packed/kaicreator-domain-harness-microkernel-*.tgz "$tmp"/packed/kaicreator-domain-harness-host-node-*.tgz 'better-sqlite3@^12'
if [[ "${T04B_NATIVE:-0}" == "1" ]]; then
  npm install --save-exact "$tmp"/packed/kaicreator-tally-native-reader-bundle-*.tgz
fi
npm ls --depth=0
if [[ "${T04B_NATIVE:-0}" == "1" ]]; then
  cp "$TALLY/experiments/v2/selected-business-build/consumer-native-sqlite.mjs" "$tmp/consumer/consumer-native-sqlite.mjs"
  TALLY_GITHUB_READ_TOKEN="$native_token" node ./consumer-native-sqlite.mjs
else
  node ./consumer.mjs
  cp "$TALLY/experiments/v2/selected-business-build/consumer-sqlite.mjs" "$tmp/consumer/consumer-sqlite.mjs"
  node ./consumer-sqlite.mjs
fi
