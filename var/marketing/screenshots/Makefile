# Regenerates marketing screenshots against a throwaway EasyPour instance
# (temporary SQLite database, fictional data). Run `make` here; everything is
# torn down afterwards. No containers are needed.

ROOT := $(abspath ../../..)
HERE := $(abspath .)
TMP := $(HERE)/tmp

BACKEND_ADDR := 127.0.0.1:18480
BACKEND_URL := http://$(BACKEND_ADDR)

.PHONY: all build up seed shoot manifest down clean

all:
	@$(MAKE) build up seed shoot manifest; rc=$$?; $(MAKE) down; exit $$rc

build:
	$(MAKE) -C $(ROOT)/service build
	[ -d $(ROOT)/frontend/node_modules ] || (cd $(ROOT)/frontend && npm ci)
	cd $(ROOT)/frontend && npx vite build --outDir $(TMP)/dist --emptyOutDir --logLevel warn

up:
	rm -rf $(TMP)/service $(TMP)/cookies
	mkdir -p $(TMP)/service
	cp service.yaml $(TMP)/service/config.yaml
	cp seed/menu.yaml $(TMP)/service/menu.yaml
	cd $(ROOT)/database/sqlite && DB_PATH=$(TMP)/service/easypour.db sql-migrate up >/dev/null
	cd $(TMP)/service && { PORT=$(BACKEND_ADDR) DB_PATH=$(TMP)/service/easypour.db EASYPOUR_STATIC_DIR=$(TMP)/dist \
		$(ROOT)/service/easypour-service -configdir . > $(TMP)/service.log 2>&1 & echo $$! > $(TMP)/service.pid; }
	@for i in $$(seq 30); do curl -fsS -X POST -H 'Content-Type: application/json' -d '{}' \
		$(BACKEND_URL)/easypour.v1.EasyPourService/Init >/dev/null 2>&1 && exit 0; sleep 1; done; \
		echo "Backend did not become ready; see $(TMP)/service.log"; exit 1

seed:
	./seed.sh $(BACKEND_URL)

shoot:
	repo-helper screenshot --config screenshots.ini

manifest:
	./generate-manifest.py

down:
	-@[ -f $(TMP)/service.pid ] && kill $$(cat $(TMP)/service.pid) 2>/dev/null; rm -f $(TMP)/service.pid

clean: down
	rm -rf $(TMP)
