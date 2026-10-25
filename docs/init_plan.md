# Transmission Navi: implementation and release status

This document records the implemented proof of concept, packaged architecture,
and remaining public-release checks. It replaces the original speculative plan.

## Architecture

```text
Transmission RPC ──► navi-indexer ──► catalog.json ──► static web UI
                           │
                           └──► TMDB API / private metadata cache

catalog file paths ── media_root → smb_prefix mapping ──► M3U playlist
```

The indexer is a one-shot process. It obtains torrent metadata through RPC and
does not scan or mount the NAS filesystem. The packaged systemd timer invokes
it every 15 minutes after an administrator explicitly enables the timer.

## Repository layout

```text
cmd/indexer/          one-shot catalog generator
pkg/catalog/          catalog model, SMB mapping, atomic writer
pkg/config/           JSON and environment configuration
pkg/parser/           release-name title/year heuristics
pkg/tmdb/             TMDB client, matching, ratings, private cache
pkg/transmission/     Transmission RPC and session handshake
web/                  dependency-free static SPA
docs/tmdb_setup.md    user instructions for obtaining a TMDB token
config.template.json  credential-free configuration template
```

Runtime files (`config.json`, `.cache/`, `web/catalog.json`, and `bin/`) are
ignored by Git.

## Implemented components

### Transmission integration

- Calls `torrent-get` for IDs, hashes, names, download directories, completion,
  status, dates, and file data.
- Handles Transmission's initial `409 Conflict` response by retaining
  `X-Transmission-Session-Id` and retrying automatically.
- Supports optional HTTP basic authentication.

### Parsing and SMB mapping

- Removes common resolution, source, codec, audio, HDR, and release tags.
- Extracts a four-digit release year when present.
- Maps only paths located beneath `media_root` to `smb_prefix`.
- Escapes individual URL path segments, including Unicode filenames.
- Selects the largest mapped video as the title's primary playback file.
- Classifies common `3D`, `SBS`, `OU`, `TAB`, and `MVC` filename tags so 3D
  releases can be separated from ordinary 2D playback.

### TMDB enrichment

- Authenticates with TMDB's **API Read Access Token** as a bearer token.
- Searches by parsed title/year and scores exact-title and exact-year matches.
- Loads localized title, original title, overview, release year, genres, TMDB
  rating, poster URL, and release certifications.
- Normalizes supported certifications to `0`, `6`, `12`, `16`, or `18`.
- Caches matches for 30 days and misses for 7 days using a private atomic file.
- By default, skips TMDB calls for torrents without a playable SMB mapping.
- Records matched, unmatched, skipped, and transient-error states per title so
  unmatched parsed names can be reviewed in the UI.

See [Create a TMDB API credential](tmdb_setup.md) for user setup and secret
handling.

### Catalog generation

- Writes an indented JSON catalog through a same-directory temporary file and
  atomic rename.
- Includes Transmission state, mapped files, TMDB metadata, and a millisecond
  catalog version.
- Never includes Transmission or TMDB credentials.

Representative movie data:

```json
{
  "title": "The Addams Family",
  "original_title": "The Addams Family",
  "year": 1991,
  "genres": ["Comedy", "Fantasy"],
  "rating_tmdb": 7.1,
  "min_age": 12,
  "poster_url": "https://image.tmdb.org/t/p/w500/example.jpg",
  "percent_done": 100,
  "status": "stopped",
  "primary_play_url": "smb://nas-host/share/The.Addams.Family.1991.mkv"
}
```

### Web UI

- Uses plain HTML, CSS, JavaScript, and the browser's native IndexedDB API; no
  Dexie or other runtime library is required.
- Provides a compact searchable header, sticky desktop sidebar, mobile filter
  drawer, multi-genre tags, age groups, Transmission states, sorting, active
  filter chips, 2D/3D selection, an unmatched-TMDB review shelf, responsive
  cards, and offline catalog fallback.
- Downloads M3U playlists containing canonical `smb://` URLs and can copy a
  primary SMB URL.
- Does not use a `vlc://` custom scheme. Desktop browsers do not register that
  handler by default, and a client-side launcher was explicitly kept out of
  scope.

## Configuration

The indexer accepts `-config <file>`. Environment variables override matching
JSON fields. The current schema is maintained in `config.template.json`:

```json
{
  "transmission_rpc": "http://nas-host:9091/transmission/rpc",
  "transmission_username": "",
  "transmission_password": "",
  "tmdb_read_access_token": "",
  "tmdb_language": "en-US",
  "tmdb_cache_path": ".cache/tmdb.json",
  "tmdb_enrich_unmapped": false,
  "catalog_path": "web/catalog.json",
  "media_root": "/absolute/path/to/torrents",
  "smb_prefix": "smb://nas-host/share/torrents/",
  "request_timeout_seconds": 15
}
```

There is no `navi-config` CLI yet. Local users edit their ignored `config.json`
or provide environment variables.

## Verified proof of concept

The current implementation has been exercised against a live Transmission
instance with 254 torrents:

- RPC web endpoint, session handshake, and `torrent-get` succeeded.
- All torrents mapped successfully after selecting the common NAS/SMB root.
- TMDB enriched 164 titles; 90 unmatched titles were negatively cached.
- A second indexing run used the cache without repeating those API requests.
- Unicode SMB paths resolved successfully through the desktop GIO stack.
- VLC opened an SMB-hosted Matroska file after installing its Samba plugin.
- Desktop and mobile UI layouts were checked in a headless browser.

Generated data, LAN addresses, local paths, and credentials are not committed.

## RPM packaging and operations

The packaging implementation now uses:

1. Installed paths:
   - `/usr/bin/navi-indexer`
   - `/etc/transmission-navi/config.json`
   - `/var/cache/transmission-navi/tmdb.json`
   - `/var/lib/transmission-navi/catalog.json`
   - `/usr/share/transmission/public_html/navi/`
2. The existing `transmission` service account from the required daemon package
   and protected `%config(noreplace)` configuration.
3. A hardened oneshot indexer service with a persistent 15-minute timer.
4. Static UI files in Transmission's existing web root under `navi/`. No
   upstream file is replaced, and Transmission remains the only HTTP server.
5. Reproducible `rpkg` source generation and offline Mock compilation using the
   same pattern as the tested sibling `hamsterd` project.

The Fedora 45 x86_64 `make fc` build for `0.1.0-4` has completed successfully,
with zero unfiltered rpmlint errors or warnings. The source is public at
<https://github.com/BOPOHA/transmission-navi>. Remaining release work is to
build the enabled Copr chroots and perform an install/upgrade test on the target
NAS. See
[installation.md](installation.md) and [packaging.md](packaging.md).

## Acceptance status

| Requirement | Status |
| --- | --- |
| Transmission session handshake and torrent indexing | Complete |
| Filename parsing and atomic catalog output | Complete |
| TMDB bearer authentication, metadata, and cache | Complete |
| Search, status, genre, age, and sorting UI | Complete |
| Unicode-safe SMB URLs and M3U generation | Complete |
| Direct browser-to-desktop VLC launch | Out of scope; M3U used instead |
| RPM spec and systemd scheduler | Fedora 45 Mock passed |
| Transmission-hosted `/transmission/web/navi/` integration | Packaged; target NAS validation pending |
| Public Git repository | Complete |
| Public Copr release | Pending |
