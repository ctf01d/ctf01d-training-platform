#!/usr/bin/env bash
set -euo pipefail

# Локальная разработка: ставим гемы внутрь volume.
bundle check || bundle install

# Прогоняем миграции и сидаем данные перед запуском сервера.
./bin/rails db:prepare
./bin/rails db:seed

exec /rails/bin/docker-entrypoint "$@"
