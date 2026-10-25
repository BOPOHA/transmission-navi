.PHONY: srpm

RPKG ?= rpkg
DNF ?= dnf
GIT ?= git

# Generate the dependency archive during Copr's network-enabled SRPM stage;
# architecture builders then compile and test entirely offline.
srpm:
	$(DNF) install -y golang rpkg
	$(GIT) config --system --add safe.directory "$$(pwd)"
	$(RPKG) srpm --spec "$(spec)" --outdir "$(outdir)"
