# Build RPMs and publish through Copr

Transmission Navi follows the same proven `rpkg`/vendor/Mock workflow used by
the sibling `hamsterd` project. The source tree remains vendor-free. During the
network-enabled SRPM stage, `packaging/rpkg.macros` creates a deterministic Go
vendor archive; architecture builders compile and test with `-mod=vendor`
without network access.

## Build prerequisites

Install the Fedora packaging tools:

```sh
sudo dnf install golang git make mock rpm-build rpkg
sudo usermod -aG mock "$USER"
```

Log out and back in after joining the `mock` group.

## Local build targets

```sh
make test       # race-enabled Go tests
make vet        # static analysis
make build      # current-platform binaries
make srpm       # source RPM in rpmbuild/SRPMS
make rpm        # rebuild the source RPM locally
make fc         # clean Fedora Mock build for the current release/architecture
```

Override the Mock chroot when needed:

```sh
make fc MOCK_CONFIG=fedora-45-x86_64
```

Lint completed artifacts with the security exceptions documented by the
package:

```sh
rpmlint -r packaging/transmission-navi.rpmlintrc rpm-results/*.rpm
```

Artifacts are written below `rpmbuild/` and `rpm-results/`; both are ignored by
Git. `make clean` removes only the generated directories and files explicitly
listed as `CLEAN_DIRS` and `CLEAN_FILES` in the Makefile. It preserves local
configuration, credentials, and IDE settings used for development.

## Copr SCM configuration

The Copr source must be publicly cloneable. Push the release commit and tag,
then verify the exact URL before each release build:

```sh
git ls-remote https://github.com/BOPOHA/transmission-navi.git refs/heads/main
```

Only then create the `vorona/transmission-navi` Copr project, add an SCM
package, and use:

| Copr field | Value |
| --- | --- |
| Clone URL | `https://github.com/BOPOHA/transmission-navi.git` |
| Committish | `main` or a release tag |
| Subdirectory | leave empty |
| Spec file | `packaging/transmission-navi.spec` |
| Build SRPM with | `make srpm` |
| Build dependencies | `golang` |

The Copr source builder runs `.copr/Makefile`, which installs `golang` and
`rpkg`, marks the checkout safe, and invokes the repository macro configuration.

Enable only distributions that provide Go 1.23 or newer. Build at least
`x86_64` and add `aarch64` only after a successful test build. Do not advertise
a release/architecture in user documentation until its Copr build succeeds.

## Release procedure

1. Update `Version`, `Release`, and `%changelog` in
   `packaging/transmission-navi.spec`. Increment `Release` for packaging-only
   changes; reset it to `1` when `Version` changes.
2. Run `make test`, `make vet`, `make build`, and `node --check web/app.js`.
3. Commit the release changes and confirm `git status --short` is empty.
4. Run `make fc` from that clean commit. The source archive is generated from
   `HEAD`, so uncommitted changes are intentionally excluded.
5. Run `rpmlint -r packaging/transmission-navi.rpmlintrc` against the spec,
   SRPM, and binary RPM. Require zero unfiltered errors and warnings.
6. Install the local RPM on a test NAS and follow `docs/installation.md`.
   Verify indexing, the timer, both the original Transmission UI and the Navi
   subpath, upgrade behavior, and VLC playlist playback from a separate client.
7. Create a signed release tag after the local install test passes, then push
   the commit and tag to the public repository.
8. Confirm `git ls-remote` can see the pushed commit/tag and trigger Copr builds
   for every enabled chroot.
9. Install the Copr-built RPM on the test NAS, repeat the smoke checks, and
   confirm `dnf upgrade transmission-navi` resolves from the Copr repository.

Do not describe the package as published until both the source repository and
the advertised Copr project are publicly reachable. A successful local Mock
build proves package construction, not public availability.

The package intentionally does not enable its timer or open firewall ports in
scriptlets. It depends on `transmission-daemon`, installs only a `navi/`
subdirectory in Transmission's public web root, and does not replace any files
owned by `transmission-common`. The indexer reuses the daemon package's
`transmission` account; no additional system account is created. Administrators
must configure and validate the RPC, SMB, and TMDB settings before opting into
periodic indexing.
