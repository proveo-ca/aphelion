#!/usr/bin/env bash
set -euo pipefail

pattern='latest_public_posts|x_[[:alnum:]_]+_live_fixture|live_child_fixture'  # keep public-safe: no exact private handles

if rg -n "$pattern" . \
  -g '!bin/**' \
  -g '!vendor/**' \
  -g '!scripts/check-no-live-child-fixtures.sh'; then
  echo "live child/account fixture leaked into repo; use generic fixture names" >&2
  exit 1
fi

production_pattern='idolum-(email|x)|[[:alnum:]_]+_telegram_child_bot_runner|[[:alnum:]_-]+\.status\.v1'

if rg -n "$production_pattern" runtime core session durableagent tool \
  -g '!**/*_test.go'; then
  echo "live child-specific production code leaked into repo; use generic child/adaptor contracts" >&2
  exit 1
fi
