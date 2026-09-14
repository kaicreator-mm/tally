#!/usr/bin/env bash
# 检查给定区间 base..head 的每个非合并提交都带 Refs: 挂钩。
#
# 用法: scripts/check-trailers.sh <base-sha> <head-sha>
#
# 追溯链的数据来源是提交信息尾部的 `Refs: REQ-xx`(见 CLAUDE.md「提交信息格式」与铁律 4)。
# 缺了挂钩,这段提交就挂不到条款上,视图 A/B 会出现断链 —— 所以作为 required status check
# 在合并前阻断,而不是靠服务端 hook(服务端 hook 对 PR 流程不可见、也难审计)。
set -euo pipefail

base="${1:?用法: check-trailers.sh <base-sha> <head-sha>}"
head="${2:?用法: check-trailers.sh <base-sha> <head-sha>}"

missing=0
while IFS= read -r sha; do
  [ -z "$sha" ] && continue
  if ! git log -1 --format=%B "$sha" | grep -qiE '^Refs:'; then
    echo "缺少 Refs: 挂钩: $sha  $(git log -1 --format=%s "$sha")"
    missing=1
  fi
done < <(git rev-list --no-merges "$base..$head")

if [ "$missing" -ne 0 ]; then
  echo ""
  echo "在提交信息尾部加一行 'Refs: REQ-xx'(确无对应条款时写 'Refs: N/A'),然后重新推送。"
  exit 1
fi
echo "所有提交都带 Refs: 挂钩。"
