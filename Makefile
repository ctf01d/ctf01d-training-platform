.PHONY: codegen database-attach database-remove database-reset database-run database-stop fmt install lint server-build server-run test \
	web-rails-image-build web-rails-image-save web-rails-image-build-and-save deploy
# -----------------------------------------------------------------------------
# Docker images (production)

# Image name can be overridden: make WEB_RAILS_IMAGE=myrepo/web-rails web-rails-image-build
WEB_RAILS_IMAGE ?= ctf01d/web-rails
GIT_TAG := $(shell git describe --tags --always 2>/dev/null || echo dev)

# Build production image for web-rails app
web-rails-image-build:
	docker build -t $(WEB_RAILS_IMAGE):$(GIT_TAG) -f web-rails/Dockerfile web-rails

# Export the built image into a tar file under dist/
web-rails-image-save:
	@mkdir -p dist
	@if ! docker image inspect $(WEB_RAILS_IMAGE):$(GIT_TAG) >/dev/null 2>&1; then \
		echo "Image $(WEB_RAILS_IMAGE):$(GIT_TAG) not found. Run 'make web-rails-image-build' first."; \
		exit 1; \
	fi
	docker image save $(WEB_RAILS_IMAGE):$(GIT_TAG) -o dist/web-rails-$(GIT_TAG).tar
	@echo "Saved to dist/web-rails-$(GIT_TAG).tar"

# Convenience: build then export
web-rails-image-build-and-save: web-rails-image-build web-rails-image-save

# -----------------------------------------------------------------------------
# Remote deploy helper (rsync with excludes)

DEPLOY_HOST ?= own-vds-france
DEPLOY_TARGET ?= ctf01d-training-platform

RSYNC_EXCLUDES = \
	--exclude .git \
	--exclude .github \
	--exclude .vscode \
	--exclude .idea \
	--exclude .aider.tags.cache.v4 \
	--exclude .cursor-free-vip \
	--exclude .DS_Store \
	--exclude '*.swp' \
	--exclude tmp \
	--exclude log \
	--exclude dist \
	--exclude vendor \
	--exclude node_modules

deploy:
	rsync -az $(RSYNC_EXCLUDES) ./ $(DEPLOY_HOST):$(DEPLOY_TARGET)
