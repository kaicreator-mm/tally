#!/usr/bin/env bash
set -euo pipefail

# Tally v1.5.4 Full Regression. Local/Build-Host only: no GitHub Actions/CI.

echo '== full repository test suite =='
go test ./... -count=1

echo '== full repository build =='
go build ./...

echo 'PASS: v1.5.4 full regression'
