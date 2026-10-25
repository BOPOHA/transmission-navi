# Install and configure Transmission Navi

This guide covers the information to collect before installation, local RPM and
Copr installation, post-install configuration, service activation, upgrades,
playback, and troubleshooting.

## 1. Collect deployment information first

Collect these values before enabling the timer. Installing the RPM is safe: its
timer is disabled by default.

### Host and package information

On the NAS, record the distribution, Fedora/EPEL release, and architecture:

```sh
cat /etc/os-release
rpm -E %fedora 2>/dev/null || true
uname -m
```

The verification commands in this guide use `curl` and `jq`. Install them if
they are not already present:

```sh
sudo dnf install curl jq
```

Confirm that Transmission's existing web interface responds on the NAS:

```sh
curl -fsS http://127.0.0.1:9091/transmission/web/ >/dev/null
```

Add `--user USER:PASSWORD` when Transmission requires HTTP basic
authentication. Navi uses this same server, port, and authentication policy; it
does not start another listener. If Transmission uses a custom `rpc-url`,
replace `/transmission/` in the examples with that configured base path.

### Transmission RPC URL and authentication

The RPC URL normally ends in `/transmission/rpc`. Verify the session handshake
from the NAS. Replace the example host but do not put credentials into project
files:

```sh
rpc_url='http://127.0.0.1:9091/transmission/rpc'
headers_file=$(mktemp)
curl -sS -D "$headers_file" -o /dev/null \
  -X POST -H 'Content-Type: application/json' \
  --data '{"method":"session-get"}' "$rpc_url"
session_id=$(sed -n 's/^X-Transmission-Session-Id: *//Ip' "$headers_file" | tr -d '\r')
curl -sS -H 'Content-Type: application/json' \
  -H "X-Transmission-Session-Id: $session_id" \
  --data '{"method":"torrent-get","arguments":{"fields":["downloadDir"]}}' \
  "$rpc_url" | jq -r '.arguments.torrents[].downloadDir' | sort -u
rm -f -- "$headers_file"
```

The first request should return HTTP `409` with a session ID; the second should
return the configured torrent download directories. Add `--user USER:PASSWORD`
to both `curl` commands when Transmission uses HTTP basic authentication.
Prefer entering that password interactively or through a protected file rather
than saving it in shell history.

### Filesystem and SMB roots

Choose one common filesystem root that contains the Transmission download
directories and the SMB URL representing exactly the same root. For example:

```text
media_root: /absolute/NAS/path/to/torrents
smb_prefix: smb://nas-host/share/torrents/
```

From a playback computer, verify a known file before installation:

```sh
gio info 'smb://nas-host/share/torrents/path/to/known-video.mkv'
```

The command must report a regular file and the expected size. Do not configure
a narrower filesystem root if the catalog should play media stored in sibling
directories.

### TMDB API Read Access Token

TMDB metadata is optional, but posters, genres, ratings, release years, and age
filters depend on it. Each deployment owner must request their own credential.
Follow [Create a TMDB API credential](tmdb_setup.md) before installing.

Keep the **API Read Access Token**, not the separate API Key, ready for entry
with `sudoedit`. Do not paste it into issue reports, screenshots, Git, or a
shell command that will be retained in history.

### Pre-install worksheet

| Setting | Value to collect |
| --- | --- |
| Transmission RPC URL | Complete `http(s)://.../transmission/rpc` URL |
| Transmission user/password | Only when RPC authentication is enabled |
| Media root | Common absolute NAS filesystem path |
| SMB prefix | Equivalent `smb://host/share/path/` URL |
| TMDB language | Usually `en-US` |
| TMDB token | Personal API Read Access Token |
| Transmission web URL | Usually `http://NAS:9091/transmission/web/` |

## 2. Install the package

### Test a locally built RPM

Before the first public release, copy the Fedora 45 x86_64 RPM produced by
`make fc` to the NAS and install it with `dnf` so dependencies are resolved:

```sh
sudo dnf install ./transmission-navi-0.1.0-4.fc45.x86_64.rpm
```

Use the actual artifact filename if its version, release, distribution, or
architecture differs. An upgrade test uses the same command with the newer RPM.

### Install from Copr after publication

After a successful public Copr build, enable the project repository and install:

```sh
sudo dnf copr enable vorona/transmission-navi
sudo dnf install transmission-navi
```

If Copr reports that the project does not exist, or `dnf` cannot resolve the
package, the public release has not completed. Do not treat local Mock success
as proof that a Copr artifact is available.

This is a project-owned third-party repository, not an official Fedora package.
Report packaging issues to this project rather than Fedora Bugzilla.

Installation adds these units but does not activate the timer:

- `transmission-navi-indexer.service`: one catalog refresh
- `transmission-navi-indexer.timer`: refresh every 15 minutes

It installs the static UI under Transmission's packaged web root:

```text
/usr/share/transmission/public_html/navi/
```

Transmission continues to own port 9091 and its original UI remains available
at `/transmission/web/`. Navi is available at `/transmission/web/navi/`. Adjust
both URLs when Transmission has a non-default `rpc-url` base path.

The indexer reuses the unprivileged `transmission` account created by the
required `transmission-daemon` package. It does not create another system user
and does not read torrent contents because file metadata comes from RPC.

## 3. Configure after installation

The package installs `/etc/transmission-navi/config.json` as a protected
`%config(noreplace)` file. Back it up and edit it:

```sh
sudo cp -a /etc/transmission-navi/config.json \
  /etc/transmission-navi/config.json.before-local-setup
sudoedit /etc/transmission-navi/config.json
```

Update these fields using the pre-install worksheet:

```json
{
  "transmission_rpc": "http://127.0.0.1:9091/transmission/rpc",
  "transmission_username": "",
  "transmission_password": "",
  "tmdb_read_access_token": "paste-your-own-token-here",
  "tmdb_language": "en-US",
  "tmdb_cache_path": "/var/cache/transmission-navi/tmdb.json",
  "tmdb_enrich_unmapped": false,
  "catalog_path": "/var/lib/transmission-navi/catalog.json",
  "media_root": "/absolute/NAS/path/to/torrents",
  "smb_prefix": "smb://nas-host/share/torrents/",
  "request_timeout_seconds": 15
}
```

Do not change the packaged cache or catalog paths unless the systemd service
sandbox is updated to grant write access to the replacement locations.

For repeatable automation, non-secret placeholders can be replaced with `sed`:

```sh
sudo sed -i \
  -e 's|http://127.0.0.1:9091/transmission/rpc|http://NAS:9091/transmission/rpc|' \
  -e 's|/srv/torrents|/absolute/NAS/path/to/torrents|' \
  -e 's|smb://nas-host/torrent/|smb://NAS/share/torrents/|' \
  /etc/transmission-navi/config.json
```

Replace the uppercase placeholders first. Use `sudoedit` for passwords and the
TMDB token so they do not appear in process listings or shell history. Confirm
that the file remains protected:

```sh
sudo chown root:transmission /etc/transmission-navi/config.json
sudo chmod 0640 /etc/transmission-navi/config.json
```

## 4. Validate before enabling services

Run one index operation and inspect its log:

```sh
sudo systemctl start transmission-navi-indexer.service
sudo systemctl --no-pager --full status transmission-navi-indexer.service
sudo journalctl -u transmission-navi-indexer.service -n 100 --no-pager
```

Success reports the indexed and TMDB-enriched title counts. Confirm that a
catalog was written without exposing its full contents publicly:

```sh
sudo -u transmission test -s /var/lib/transmission-navi/catalog.json
sudo -u transmission jq \
  '{movies: (.movies | length), playable: ([.movies[] | select(.primary_play_url)] | length)}' \
  /var/lib/transmission-navi/catalog.json
```

Verify the packaged catalog link and both Transmission web pages. Add
`--user USER:PASSWORD` to the `curl` commands when required:

```sh
readlink /usr/share/transmission/public_html/navi/catalog.json
curl -f http://127.0.0.1:9091/transmission/web/ >/dev/null
curl -f http://127.0.0.1:9091/transmission/web/navi/ >/dev/null
curl -f http://127.0.0.1:9091/transmission/web/navi/catalog.json >/dev/null
```

The `catalog.json` link is intentionally unresolved before the first successful
index run. If the original UI works but `/navi/` returns `404`, confirm that the
installed Transmission package uses `/usr/share/transmission/public_html` as
its web root.

Only after the one-shot index and web checks work, enable periodic indexing:

```sh
sudo systemctl enable --now transmission-navi-indexer.timer
systemctl list-timers transmission-navi-indexer.timer
```

## 5. Open Navi on the LAN

Navi requires no additional firewall rule: it is served by the already-running
Transmission daemon. If Transmission's port is intentionally restricted, keep
that policy and permit it only on a trusted home-network zone. For the usual
port, an example rule is:

```sh
sudo firewall-cmd --get-active-zones
sudo firewall-cmd --permanent --zone=home --add-port=9091/tcp
sudo firewall-cmd --reload
```

Do not add a duplicate rule if Transmission is already reachable. Use the zone
actually assigned to the trusted LAN interface. The catalog inherits
Transmission's RPC authentication settings and contains media titles and SMB
paths; never expose Transmission directly to the public internet.

Open `http://NAS-IP:9091/transmission/web/navi/` from a playback computer. The
original interface stays at `http://NAS-IP:9091/transmission/web/`.

## 6. Configure VLC on playback computers

The NAS package does not install VLC on client computers. On Fedora clients:

```sh
sudo dnf install vlc vlc-plugin-samba
```

In Navi, **VLC playlist** downloads an M3U containing the canonical SMB URL.
Open the playlist with VLC. **Copy link** copies the same URL for testing:

```sh
vlc 'smb://nas-host/share/torrents/path/to/video.mkv'
```

On Android, **Watch on TV** opens the selected SMB movie in VLC for Android.
Use VLC's Cast button to select a Chromecast or Chromecast-enabled TV. Test
direct TV access separately in VLC for Android TV under **Browse → Local
network → SMB**; successful playback and seeking there confirms that the TV's
VLC installation and stored NAS credentials can read the share.

## 7. Upgrade and reconfigure

After Copr publication, upgrade normally:

```sh
sudo dnf upgrade transmission-navi
```

RPM preserves a locally changed `config.json`. Review a new packaged template
when an upgrade creates an `.rpmnew` file:

```sh
sudo find /etc/transmission-navi -maxdepth 1 \
  -name '*.rpmnew' -o -name '*.rpmsave'
```

If an `.rpmnew` file is listed, compare it with the active configuration:

```sh
sudo diff -u /etc/transmission-navi/config.json \
  /etc/transmission-navi/config.json.rpmnew
```

Merge new fields deliberately, then refresh the catalog:

```sh
sudo systemctl start transmission-navi-indexer.service
```

Static UI upgrades are visible on the next browser reload; Transmission does
not need to restart.

After changing `media_root` or `smb_prefix`, regenerate the catalog immediately.
Existing browser tabs refresh within 60 seconds.

## Troubleshooting

### Indexer fails

```sh
sudo journalctl -u transmission-navi-indexer.service -n 200 --no-pager
sudo -u transmission /usr/bin/navi-indexer \
  -config /etc/transmission-navi/config.json
```

Common causes are an incorrect RPC URL, missing Transmission credentials, an
invalid TMDB token, DNS/network failure, or a filesystem/SMB root mismatch.

### UI loads but the catalog does not

```sh
systemctl status transmission-daemon.service
ls -lZ /var/lib/transmission-navi/catalog.json
readlink -f /usr/share/transmission/public_html/navi/catalog.json
curl -v http://127.0.0.1:9091/transmission/web/navi/catalog.json
```

Add Transmission credentials to `curl` when configured. Do not move the
catalog outside `/var/lib/transmission-navi` without adjusting the indexer
service's `ReadWritePaths` sandbox and the packaged web link.

### Playlist opens but VLC cannot read SMB

Confirm `vlc-plugin-samba` is installed and test the copied canonical SMB URL
directly. Also verify SMB authentication in the desktop file manager or with
`gio info`.
