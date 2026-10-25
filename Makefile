.PHONY: all build builddep clean deps fc fmt index rpm serve srpm test vet

PRJNAME ?= transmission-navi
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
GO ?= go
RPKG ?= rpkg

PROJECT_ROOT := $(abspath $(dir $(lastword $(MAKEFILE_LIST))))
GOBASE ?= $(PROJECT_ROOT)
GOBIN ?= $(GOBASE)/bin
RPM_OUTDIR ?= $(GOBASE)/rpmbuild
SPEC := packaging/transmission-navi.spec
CONFIG ?= config.json
FEDORA_VERSION ?= $(shell rpm -E %fedora)
MOCK_CONFIG ?= fedora-$(FEDORA_VERSION)-$(shell uname -m)
LDFLAGS := -s -w \
	-X github.com/BOPOHA/transmission-navi/internal/buildinfo.Version=$(VERSION)
CLEAN_DIRS := \
	$(PROJECT_ROOT)/.cache \
	$(PROJECT_ROOT)/bin \
	$(PROJECT_ROOT)/dist \
	$(PROJECT_ROOT)/rpm-results \
	$(PROJECT_ROOT)/rpmbuild \
	$(PROJECT_ROOT)/vendor
CLEAN_FILES := \
	$(PROJECT_ROOT)/web/catalog.json

all: clean test build

deps:
	$(GO) mod download

build:
	mkdir -p $(GOBIN)
	CGO_ENABLED=0 $(GO) build -trimpath -ldflags="$(LDFLAGS)" -o $(GOBIN)/navi-indexer ./cmd/indexer

test:
	CGO_ENABLED=1 $(GO) test -race ./...

vet:
	CGO_ENABLED=0 $(GO) vet ./...

fmt:
	gofmt -w cmd internal pkg

index:
	$(GO) run ./cmd/indexer -config $(CONFIG)

serve:
	python3 -m http.server --bind 127.0.0.1 --directory web 8080

srpm:
	mkdir -p $(RPM_OUTDIR)/SRPMS
	rm -f $(RPM_OUTDIR)/SRPMS/*.src.rpm
	$(RPKG) srpm --spec $(SPEC) --outdir $(RPM_OUTDIR)/SRPMS
	@find $(RPM_OUTDIR)/SRPMS -name '*.src.rpm' -print

rpm: srpm
	@srpm="$$(find $(RPM_OUTDIR)/SRPMS -name '*.src.rpm' -print -quit)"; \
		test -n "$$srpm"; \
		rpmbuild --rebuild "$$srpm" --define "_topdir $(RPM_OUTDIR)"
	@find $(RPM_OUTDIR)/RPMS -name '*.rpm' -print

builddep: srpm
	@srpm="$$(find $(RPM_OUTDIR)/SRPMS -name '*.src.rpm' -print -quit)"; \
		test -n "$$srpm"; \
		dnf builddep --assumeno "$$srpm"

fc: srpm
	@srpm="$$(find $(RPM_OUTDIR)/SRPMS -name '*.src.rpm' -print -quit)"; \
		test -n "$$srpm"; \
		mock --no-clean -r $(MOCK_CONFIG) --resultdir=$(GOBASE)/rpm-results "$$srpm"

clean:
	@for path in $(CLEAN_DIRS) $(CLEAN_FILES); do \
		case "$$path" in \
		"$(PROJECT_ROOT)"/*) rm -rf -- "$$path" ;; \
		*) echo "refusing to clean path outside $(PROJECT_ROOT): $$path" >&2; exit 1 ;; \
		esac; \
	done
	@find "$(PROJECT_ROOT)" -maxdepth 1 -type f \
		\( -name '*.test' -o -name '*.out' \) -delete
