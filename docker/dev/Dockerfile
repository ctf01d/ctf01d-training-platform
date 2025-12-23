# syntax=docker/dockerfile:1
# check=error=true

ARG RUBY_VERSION=3.4.8
FROM docker.io/library/ruby:${RUBY_VERSION}-slim

WORKDIR /rails

# Инструменты для сборки гемов и клиент Postgres
RUN apt-get update -qq && \
    apt-get install --no-install-recommends -y build-essential curl git libjemalloc2 libpq-dev libyaml-dev pkg-config postgresql-client && \
    rm -rf /var/lib/apt/lists /var/cache/apt/archives

ENV RAILS_ENV="development" \
    BUNDLE_PATH="/bundle" \
    BUNDLE_WITHOUT=""

COPY docker/dev/entrypoint.sh /usr/local/bin/dev-entrypoint
RUN chmod +x /usr/local/bin/dev-entrypoint

ENTRYPOINT ["/usr/local/bin/dev-entrypoint"]
CMD ["./bin/rails", "server", "-b", "0.0.0.0"]
