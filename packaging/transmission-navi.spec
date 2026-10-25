# SPDX-License-Identifier: MIT
%global debug_package %{nil}

Name:           transmission-navi
Version:        0.1.0
Release:        7%{?dist}
Summary:        Static media catalog and browser for Transmission

License:        MIT
URL:            https://github.com/BOPOHA/transmission-navi
Source0:        {{{ git_repo_pack }}}
Source1:        {{{ go_vendor_pack }}}

BuildRequires:  golang >= 1.23
BuildRequires:  gcc
BuildRequires:  systemd-rpm-macros
Requires:       transmission-daemon
%{?systemd_requires}

%description
Transmission Navi indexes torrents through the Transmission RPC API, enriches
movie titles with cached TMDB metadata, maps NAS file paths to SMB URLs, and
adds a responsive media catalog below Transmission's existing web interface.
Playback is provided through downloadable VLC-compatible M3U playlists.

%prep
%setup -T -b 0 -q -n transmission-navi
tar xzf %{SOURCE1}

%build
CGO_ENABLED=1 go build -mod=vendor -buildmode=pie -trimpath \
    -ldflags "-linkmode=external -s -w -X github.com/BOPOHA/transmission-navi/internal/buildinfo.Version=%{version}" \
    -o %{_builddir}/navi-indexer-bin ./cmd/indexer

%check
CGO_ENABLED=0 go test -mod=vendor ./...

%install
install -Dm755 %{_builddir}/navi-indexer-bin %{buildroot}%{_bindir}/navi-indexer
install -Dm644 docs/navi-indexer.1 %{buildroot}%{_mandir}/man1/navi-indexer.1

install -Dm640 packaging/transmission-navi.conf \
    %{buildroot}%{_sysconfdir}/transmission-navi/config.json
install -Dm644 packaging/transmission-navi-indexer.service \
    %{buildroot}%{_unitdir}/transmission-navi-indexer.service
install -Dm644 packaging/transmission-navi-indexer.timer \
    %{buildroot}%{_unitdir}/transmission-navi-indexer.timer

install -d %{buildroot}%{_datadir}/transmission/public_html/navi
install -m644 web/index.html web/app.js web/styles.css \
    %{buildroot}%{_datadir}/transmission/public_html/navi/
ln -s ../../../../../var/lib/transmission-navi/catalog.json \
    %{buildroot}%{_datadir}/transmission/public_html/navi/catalog.json

install -d %{buildroot}%{_sharedstatedir}/transmission-navi
install -d %{buildroot}%{_localstatedir}/cache/transmission-navi

install -Dm644 LICENSE %{buildroot}%{_licensedir}/transmission-navi/LICENSE
install -Dm644 README.md %{buildroot}%{_docdir}/transmission-navi/README.md
install -d %{buildroot}%{_docdir}/transmission-navi/docs
install -m644 docs/*.md %{buildroot}%{_docdir}/transmission-navi/docs/

%post
%systemd_post transmission-navi-indexer.timer

%preun
%systemd_preun transmission-navi-indexer.timer

%postun
%systemd_postun_with_restart transmission-navi-indexer.timer

%files
%license %{_licensedir}/transmission-navi/LICENSE
%doc %{_docdir}/transmission-navi/
%{_bindir}/navi-indexer
%{_mandir}/man1/navi-indexer.1*
%dir %{_datadir}/transmission/public_html/navi
%{_datadir}/transmission/public_html/navi/*
%{_unitdir}/transmission-navi-indexer.service
%{_unitdir}/transmission-navi-indexer.timer
%dir %{_sysconfdir}/transmission-navi
%config(noreplace) %attr(0640,root,transmission) %{_sysconfdir}/transmission-navi/config.json
%dir %attr(0755,transmission,transmission) %{_sharedstatedir}/transmission-navi
%dir %attr(0750,transmission,transmission) %{_localstatedir}/cache/transmission-navi

%changelog
* Sun Oct 25 2026 Anatolii Vorona <vorona.tolik@gmail.com> - 0.1.0-7
- Split multi-film torrents while preserving detected episode packs
- Improve TMDB lookup for transliterated titles and expose unmatched-title review links
- Clean release-group labels and default the library video filter to 2D

* Sun Oct 25 2026 Anatolii Vorona <vorona.tolik@gmail.com> - 0.1.0-6
- Add an Android VLC launch action for Chromecast playback testing

* Sun Oct 25 2026 Anatolii Vorona <vorona.tolik@gmail.com> - 0.1.0-5
- Add 2D and 3D media filtering with Quest-oriented labels
- Add a review shelf for titles that TMDB could not match

* Sun Oct 25 2026 Anatolii Vorona <vorona.tolik@gmail.com> - 0.1.0-4
- Keep RPM systemd integration limited to the catalog indexer timer

* Sun Oct 25 2026 Anatolii Vorona <vorona.tolik@gmail.com> - 0.1.0-3
- Run the indexer with Transmission's existing service account
- Remove the dedicated account and shadow-utils scriptlet dependency

* Sun Oct 25 2026 Anatolii Vorona <vorona.tolik@gmail.com> - 0.1.0-2
- Serve Navi from Transmission's existing web interface under /navi/

* Sun Oct 25 2026 Anatolii Vorona <vorona.tolik@gmail.com> - 0.1.0-1
- Package the indexer, static web UI, and systemd timer
