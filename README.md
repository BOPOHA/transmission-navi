# Transmission Navi

Transmission Navi is a self-hosted media browser for Transmission. A
dependency-free Go indexer reads Transmission RPC data, enriches playable
movies with cached TMDB metadata, and atomically publishes a static JSON
catalog. The responsive web UI provides search, sorting, sidebar filters,
offline catalog caching, and VLC-compatible playlists.

## Current features

- Transmission RPC client with automatic session-ID (`409`) retry
- Filename cleanup, release-group removal, and title/year extraction
- Separate catalog entries for films bundled in one torrent
- Automatic 2D/3D classification from common stereoscopic release tags
- TMDB posters, localized titles, genres, ratings, and age certifications
- Cyrillic retry for Russian titles stored as Latin transliterations
- Optional PoiskKino-to-TMDB resolver for unmatched Russian titles
- A review shelf for titles that configured metadata providers could not match
- Persistent TMDB cache (30 days for matches, 7 days for misses)
- Filesystem-to-SMB path mapping with URL-safe filenames
- Atomic `catalog.json` publication
- Responsive desktop sidebar and mobile filter drawer
- Search, download state, 2D/3D, genre, age, and sorting controls
- IndexedDB fallback to the latest browser-side catalog
- Per-title and per-file M3U playlist generation

RPM packaging, a hardened indexer service, and a scheduled index timer are
included. The packaged UI is served by Transmission itself at
`/transmission/web/navi/`; the original UI is preserved. Start with the
[installation and configuration guide](docs/installation.md).

## Requirements

- Transmission daemon 4.x on the host that serves the packaged UI
- A reachable Transmission RPC endpoint
- A filesystem root and matching SMB URL prefix
- A free TMDB API Read Access Token for metadata enrichment (optional)
- VLC with SMB support on computers used for playback

Building from source additionally requires Go 1.23 or newer. The local preview
target uses Python 3; neither Go nor Python is required when installing the RPM.

On Fedora, install VLC's SMB input module with:

```sh
sudo dnf install vlc-plugin-samba
```

## Install from Copr

After the first public Copr build succeeds, install it on a supported Fedora
NAS with:

```sh
sudo dnf copr enable vorona/transmission-navi
sudo dnf install transmission-navi
```

The repository is not considered released until that command can resolve a
signed package from Copr. For pre-release testing, install the locally built
RPM as described in the [installation guide](docs/installation.md). Do not
enable the timer immediately: first collect the Transmission RPC, filesystem,
SMB, and TMDB values, edit the protected configuration, and perform a one-shot
validation.

## Quick start

Create a machine-local configuration. Both `config.json` and generated data are
ignored by Git:

```sh
cp config.template.json config.json
chmod 600 config.json
$EDITOR config.json
```

`make clean` removes only the generated paths listed in the Makefile. Local
development files such as `config.json`, `.env`, `config.local.json`, and IDE
settings are preserved.

At minimum, set:

- `transmission_rpc`: the complete `/transmission/rpc` URL
- `media_root`: the filesystem root reported by Transmission
- `smb_prefix`: the same root as an `smb://` URL

Add `tmdb_read_access_token` to enable metadata. Each deployment owner should
create their own credential using the [TMDB setup guide](docs/tmdb_setup.md).
An optional `poiskkino_api_token` can resolve remaining Russian titles to TMDB
IDs; TMDB remains the canonical metadata provider.

Generate the catalog and start the local development preview (Python 3 is used
only for this preview server):

```sh
make index
make serve
```

Open <http://127.0.0.1:8080>. The UI checks for a newer catalog every 60
seconds. Re-run `make index` whenever Transmission data should be refreshed.

To use another configuration file:

```sh
make index CONFIG=/path/to/config.json
```

## Configuration

All JSON settings can be overridden with environment variables:

| JSON field | Environment variable | Purpose |
| --- | --- | --- |
| `transmission_rpc` | `NAVI_TRANSMISSION_RPC` | Transmission RPC URL |
| `transmission_username` | `NAVI_TRANSMISSION_USERNAME` | Optional RPC basic-auth user |
| `transmission_password` | `NAVI_TRANSMISSION_PASSWORD` | Optional RPC basic-auth password |
| `tmdb_read_access_token` | `NAVI_TMDB_READ_ACCESS_TOKEN` | Optional TMDB bearer token |
| `tmdb_language` | `NAVI_TMDB_LANGUAGE` | TMDB localization, default `en-US` |
| `tmdb_cache_path` | `NAVI_TMDB_CACHE_PATH` | Private metadata cache path |
| `tmdb_enrich_unmapped` | `NAVI_TMDB_ENRICH_UNMAPPED` | Enrich media without an SMB mapping |
| `poiskkino_api_token` | `NAVI_POISKINO_API_TOKEN` | Optional PoiskKino resolver token |
| `catalog_path` | `NAVI_CATALOG_PATH` | Generated static catalog path |
| `media_root` | `NAVI_MEDIA_ROOT` | NAS filesystem root |
| `smb_prefix` | `NAVI_SMB_PREFIX` | SMB URL corresponding to `media_root` |
| `request_timeout_seconds` | `NAVI_REQUEST_TIMEOUT_SECONDS` | Per-request timeout |

Environment variables take precedence over JSON. Keep credentials out of shell
history, Git, the static web directory, and `catalog.json`.

## VLC playback

Browsers cannot start desktop VLC with an arbitrary SMB URL unless every client
installs a custom protocol handler. Navi deliberately does not require one.
Instead, **VLC playlist** downloads a small `.m3u` containing the canonical
`smb://` URL. Open that playlist with VLC. **Copy link** copies the same SMB URL
for troubleshooting or manual use. On Android, **Play on this device** opens
the SMB movie in VLC on the current phone or tablet. VLC may then route it to a
renderer selected inside VLC, but Navi does not select a remote device itself.

The **Metadata not found** shelf shows the parsed lookup name and provides a
Kinopoisk search link for manual review. Navi does not scrape Kinopoisk pages.

## Verification

```sh
make test
make vet
make build
node --check web/app.js
```

The tests cover Transmission's session handshake, title parsing, SMB mapping,
atomic catalog/cache writes, JSON/environment configuration, TMDB matching,
bearer authentication, and certification normalization.

## RPM and Copr builds

The RPM uses a deterministic `rpkg` source archive and offline Go vendor bundle,
then compiles and tests in Mock:

```sh
make srpm
make rpm
make fc
```

See [RPM and Copr packaging](docs/packaging.md) for prerequisites, SCM settings,
release steps, and the supported-chroot policy.
