(() => {
  "use strict";

  const state = { movies: [], query: "", status: "all", video: "all", genres: new Set(), age: "all", sort: "added" };
  const elements = {
    catalog: document.querySelector("#catalog"),
    empty: document.querySelector("#empty"),
    template: document.querySelector("#card-template"),
    count: document.querySelector("#result-count"),
    title: document.querySelector("#result-title"),
    search: document.querySelector("#search"),
    sort: document.querySelector("#sort"),
    syncDot: document.querySelector("#sync-dot"),
    syncLabel: document.querySelector("#sync-label"),
    genreFilters: document.querySelector("#genre-filters"),
    activeFilters: document.querySelector("#active-filters"),
    clearGenres: document.querySelector("#clear-genres"),
    clearAll: document.querySelector("#clear-all"),
    filterCount: document.querySelector("#filter-count"),
    filterSidebar: document.querySelector("#filter-sidebar"),
    openFilters: document.querySelector("#open-filters"),
    closeFilters: document.querySelector("#close-filters"),
    filterScrim: document.querySelector("#filter-scrim")
  };

  const palette = [
    ["#857445", "#302815"], ["#4c6b61", "#182924"], ["#765268", "#2c1826"],
    ["#596a87", "#182236"], ["#7b5a3e", "#2d1f14"], ["#5f7042", "#1d2813"]
  ];

  function openDB() {
    return new Promise((resolve, reject) => {
      const request = indexedDB.open("navi-catalog", 1);
      request.onupgradeneeded = () => request.result.createObjectStore("catalog");
      request.onsuccess = () => resolve(request.result);
      request.onerror = () => reject(request.error);
    });
  }

  async function cacheGet() {
    const db = await openDB();
    return new Promise((resolve, reject) => {
      const transaction = db.transaction("catalog", "readonly");
      const request = transaction.objectStore("catalog").get("latest");
      request.onsuccess = () => resolve(request.result);
      request.onerror = () => reject(request.error);
    });
  }

  async function cachePut(catalog) {
    const db = await openDB();
    return new Promise((resolve, reject) => {
      const transaction = db.transaction("catalog", "readwrite");
      transaction.objectStore("catalog").put(catalog, "latest");
      transaction.oncomplete = resolve;
      transaction.onerror = () => reject(transaction.error);
    });
  }

  async function syncCatalog(silent = false) {
    try {
      const response = await fetch(`catalog.json?t=${Date.now()}`, { cache: "no-store" });
      if (!response.ok) throw new Error(`HTTP ${response.status}`);
      const catalog = await response.json();
      if (!Array.isArray(catalog.movies)) throw new Error("Invalid catalog format");
      state.movies = catalog.movies;
      buildSidebar();
      render();
      cachePut(catalog).catch(() => {});
      elements.syncDot.className = "sync-dot live";
      elements.syncLabel.textContent = `Updated ${relativeTime(catalog.updated_at)}`;
    } catch (error) {
      const cached = await cacheGet().catch(() => null);
      if (cached?.movies?.length && !state.movies.length) {
        state.movies = cached.movies;
        buildSidebar();
        render();
        elements.syncLabel.textContent = "Offline copy";
      } else if (!silent && !state.movies.length) {
        showLoadError(error);
      }
      elements.syncDot.className = "sync-dot error";
    }
  }

  function render() {
    const query = state.query.trim().toLocaleLowerCase();
    const movies = state.movies.filter(movie => {
      const matchesQuery = !query || [movie.title, movie.original_title, movie.torrent_name, ...(movie.genres || []), ...(movie.files || []).map(file => file.name)]
        .some(value => String(value || "").toLocaleLowerCase().includes(query));
      const playable = Boolean(movie.primary_play_url);
      const matchesStatus = state.status === "all" ||
        (state.status === "ready" && playable && movie.percent_done === 100) ||
        (state.status === "active" && movie.percent_done < 100) ||
        (state.status === "stopped" && movie.status === "stopped") ||
        (state.status === "tmdb_not_found" && movie.tmdb_status === "not_found");
      const videoMode = movie.video_mode || "2d";
      const matchesVideo = state.video === "all" || state.video === videoMode;
      const matchesGenres = [...state.genres].every(genre => (movie.genres || []).includes(genre));
      const matchesAge = state.age === "all" ||
        (state.age === "unrated" && movie.min_age == null) ||
        (state.age === "18" && movie.min_age === 18) ||
        (!Number.isNaN(Number(state.age)) && state.age !== "18" && movie.min_age != null && movie.min_age <= Number(state.age));
      return matchesQuery && matchesStatus && matchesVideo && matchesGenres && matchesAge;
    });
    movies.sort(comparator(state.sort));

    elements.catalog.replaceChildren(...movies.map(card));
    elements.empty.hidden = movies.length !== 0;
    elements.catalog.hidden = movies.length === 0;
    elements.count.textContent = `${movies.length} ${movies.length === 1 ? "title" : "titles"}`;
    elements.title.textContent = state.query ? `Results for “${state.query.trim()}”` : statusLabel(state.status);
    renderFilterState();
  }

  function buildSidebar() {
    const statusCounts = {
      all: state.movies.length,
      ready: state.movies.filter(movie => movie.primary_play_url && movie.percent_done === 100).length,
      active: state.movies.filter(movie => movie.percent_done < 100).length,
      stopped: state.movies.filter(movie => movie.status === "stopped").length,
      tmdb_not_found: state.movies.filter(movie => movie.tmdb_status === "not_found").length
    };
    Object.entries(statusCounts).forEach(([name, count]) => {
      const target = document.querySelector(`[data-count="${name}"]`);
      if (target) target.textContent = count;
    });

    const genreCounts = new Map();
    state.movies.forEach(movie => (movie.genres || []).forEach(genre => genreCounts.set(genre, (genreCounts.get(genre) || 0) + 1)));
    const buttons = [...genreCounts.entries()]
      .sort(([left], [right]) => left.localeCompare(right))
      .map(([genre, count]) => {
        const button = document.createElement("button");
        button.type = "button";
        button.className = "genre-filter";
        button.dataset.genre = genre;
        button.setAttribute("aria-pressed", String(state.genres.has(genre)));
        button.classList.toggle("active", state.genres.has(genre));
        const label = document.createElement("span");
        label.textContent = genre;
        const total = document.createElement("small");
        total.textContent = count;
        button.append(label, total);
        button.addEventListener("click", () => {
          state.genres.has(genre) ? state.genres.delete(genre) : state.genres.add(genre);
          button.classList.toggle("active", state.genres.has(genre));
          button.setAttribute("aria-pressed", String(state.genres.has(genre)));
          render();
        });
        return button;
      });
    elements.genreFilters.replaceChildren(...buttons);
  }

  function renderFilterState() {
    document.querySelectorAll("[data-status]").forEach(button => {
      const active = button.dataset.status === state.status;
      button.classList.toggle("active", active);
      button.setAttribute("aria-pressed", String(active));
    });
    document.querySelectorAll("[data-age]").forEach(button => {
      const active = button.dataset.age === state.age;
      button.classList.toggle("active", active);
      button.setAttribute("aria-pressed", String(active));
    });
    document.querySelectorAll("[data-video]").forEach(button => {
      const active = button.dataset.video === state.video;
      button.classList.toggle("active", active);
      button.setAttribute("aria-pressed", String(active));
    });
    elements.genreFilters.querySelectorAll("[data-genre]").forEach(button => {
      button.classList.toggle("active", state.genres.has(button.dataset.genre));
      button.setAttribute("aria-pressed", String(state.genres.has(button.dataset.genre)));
    });

    const activeCount = state.genres.size + Number(state.status !== "all") + Number(state.video !== "all") + Number(state.age !== "all");
    elements.filterCount.hidden = activeCount === 0;
    elements.filterCount.textContent = activeCount;
    elements.clearAll.disabled = activeCount === 0;
    elements.clearGenres.hidden = state.genres.size === 0;
    renderActiveChips();
  }

  function renderActiveChips() {
    const chips = [];
    if (state.status !== "all") chips.push(filterChip(statusLabel(state.status), () => { state.status = "all"; render(); }));
    if (state.video !== "all") chips.push(filterChip(state.video === "3d" ? "3D · Quest" : "2D", () => { state.video = "all"; render(); }));
    if (state.age !== "all") chips.push(filterChip(ageLabel(state.age), () => { state.age = "all"; render(); }));
    [...state.genres].sort().forEach(genre => chips.push(filterChip(genre, () => { state.genres.delete(genre); render(); })));
    elements.activeFilters.replaceChildren(...chips);
  }

  function filterChip(label, remove) {
    const button = document.createElement("button");
    button.type = "button";
    button.className = "filter-chip";
    button.setAttribute("aria-label", `Remove ${label} filter`);
    button.append(document.createTextNode(label));
    const close = document.createElement("span");
    close.textContent = "×";
    button.append(close);
    button.addEventListener("click", remove);
    return button;
  }

  function statusLabel(status) {
    return { all: "All media", ready: "Ready to play", active: "In progress", stopped: "Stopped", tmdb_not_found: "TMDB not found" }[status] || "All media";
  }

  function ageLabel(age) {
    return { "6": "Kids · up to 6", "12": "Family · up to 12", "16": "Teen · up to 16", "18": "Adults only", unrated: "Not rated" }[age] || "Any rating";
  }

  function card(movie) {
    const node = elements.template.content.firstElementChild.cloneNode(true);
    const color = palette[hash(movie.hash_string || movie.title) % palette.length];
    node.style.setProperty("--cover-a", color[0]);
    node.style.setProperty("--cover-b", color[1]);
    node.querySelector("h3").textContent = movie.title || movie.torrent_name;
    node.querySelector(".cover-letter").textContent = firstLetter(movie.title);
    node.querySelector(".cover-year").textContent = movie.year || "—";
    node.querySelector(".video-mode").hidden = (movie.video_mode || "2d") !== "3d";
    node.querySelector(".year").textContent = movie.year || "Unknown year";
    node.querySelector(".size").textContent = formatBytes(totalSize(movie.files));
    node.querySelector(".rating").textContent = movie.rating_tmdb ? `★ ${Number(movie.rating_tmdb).toFixed(1)}` : "";
    node.querySelector(".age").textContent = movie.min_age != null ? `${movie.min_age}+` : "";
    node.querySelector("h3").title = movie.overview || movie.title || "";
    if (movie.tmdb_status === "not_found") {
      const tmdbNote = node.querySelector(".tmdb-note");
      const lookup = `${movie.title || "Unknown title"}${movie.year ? ` (${movie.year})` : ""}`;
      tmdbNote.textContent = `TMDB did not find “${lookup}”. Source: ${movie.torrent_name}`;
      tmdbNote.hidden = false;
    }
    const genres = node.querySelector(".genres");
    (movie.genres || []).slice(0, 3).forEach(name => {
      const genre = document.createElement("span");
      genre.className = "genre";
      genre.textContent = name;
      genres.append(genre);
    });
    if (movie.poster_url) {
      const poster = node.querySelector(".cover-poster");
      poster.src = movie.poster_url;
      poster.alt = `Poster for ${movie.title}`;
      poster.addEventListener("load", () => { node.querySelector(".cover-letter").hidden = true; });
      poster.addEventListener("error", () => { poster.removeAttribute("src"); });
    }
    node.querySelector(".progress span").style.width = `${movie.percent_done || 0}%`;
    node.querySelector(".progress-label").textContent = `${movie.percent_done || 0}%`;

    const badge = node.querySelector(".badge");
    const ready = movie.percent_done === 100 && movie.primary_play_url;
    badge.textContent = ready ? "Ready" : movie.status || "Unknown";
    badge.classList.toggle("ready", ready);

    const play = node.querySelector(".play");
    const watchTV = node.querySelector(".watch-tv");
    const copyLink = node.querySelector(".copy-link");
    if (movie.primary_play_url) {
      watchTV.addEventListener("click", async () => {
        if (/Android/i.test(navigator.userAgent)) {
          window.location.href = androidVLCIntent(movie.primary_play_url);
          return;
        }
        await copyText(movie.primary_play_url);
        watchTV.textContent = "SMB link copied";
        setTimeout(() => { watchTV.textContent = "Watch on TV"; }, 1800);
      });
      play.addEventListener("click", () => downloadPlaylist(movie));
      copyLink.addEventListener("click", async () => {
        await copyText(movie.primary_play_url);
        copyLink.textContent = "Copied";
        setTimeout(() => { copyLink.textContent = "Copy link"; }, 1400);
      });
    } else {
      watchTV.disabled = true;
      play.disabled = true;
      play.textContent = "No SMB mapping";
      copyLink.disabled = true;
    }

    const files = node.querySelector(".files");
    (movie.files || []).forEach(file => {
      const row = document.createElement("div");
      row.className = "file";
      if (file.play_url) {
        const fileButton = document.createElement("button");
        fileButton.type = "button";
        fileButton.textContent = file.name;
        fileButton.title = "Download a VLC playlist for this file";
        fileButton.addEventListener("click", () => downloadPlaylist(movie, [file]));
        row.append(fileButton);
      } else {
        row.textContent = file.name;
      }
      row.append(` · ${formatBytes(file.size)}`);
      files.append(row);
    });
    const more = node.querySelector(".more");
    more.addEventListener("click", () => {
      files.hidden = !files.hidden;
      more.setAttribute("aria-expanded", String(!files.hidden));
    });
    return node;
  }

  function comparator(sort) {
    if (sort === "title") return (a, b) => String(a.title).localeCompare(String(b.title));
    if (sort === "year") return (a, b) => (b.year || 0) - (a.year || 0);
    if (sort === "progress") return (a, b) => (b.percent_done || 0) - (a.percent_done || 0);
    return (a, b) => Date.parse(b.added_at || 0) - Date.parse(a.added_at || 0);
  }

  function downloadPlaylist(movie, selectedFiles = movie.files || []) {
    const urls = selectedFiles.filter(file => file.play_url).map(file => file.play_url);
    if (!urls.length) return;
    const blob = new Blob(["#EXTM3U\n", ...urls.map(url => `${url}\n`)], { type: "audio/x-mpegurl" });
    const href = URL.createObjectURL(blob);
    const link = Object.assign(document.createElement("a"), { href, download: `${safeName(movie.title || "navi")}.m3u` });
    link.click();
    setTimeout(() => URL.revokeObjectURL(href), 1000);
  }

  function androidVLCIntent(smbURL) {
    const target = smbURL.replace(/^smb:\/\//i, "");
    return `intent://${target}#Intent;scheme=smb;type=video/*;package=org.videolan.vlc;end`;
  }

  async function copyText(value) {
    if (navigator.clipboard?.writeText) {
      try {
        await navigator.clipboard.writeText(value);
        return;
      } catch (_) {
        // HTTP deployments may deny Clipboard API access; use the legacy fallback.
      }
    }
    const input = Object.assign(document.createElement("textarea"), { value });
    input.style.position = "fixed";
    input.style.opacity = "0";
    document.body.append(input);
    input.select();
    document.execCommand("copy");
    input.remove();
  }

  function showLoadError(error) {
    elements.catalog.hidden = true;
    elements.empty.hidden = false;
    elements.empty.querySelector("h2").textContent = "Catalog is not ready";
    elements.empty.querySelector("p").textContent = `Run the indexer, then refresh. ${error.message}`;
    elements.syncLabel.textContent = "Catalog unavailable";
  }

  function totalSize(files = []) { return files.reduce((sum, file) => sum + (file.size || 0), 0); }
  function firstLetter(value = "?") { return Array.from(value.trim())[0]?.toLocaleUpperCase() || "?"; }
  function hash(value = "") { return Array.from(value).reduce((result, char) => ((result << 5) - result + char.charCodeAt(0)) | 0, 0) >>> 0; }
  function safeName(value) { return value.replace(/[^a-z0-9 _-]/gi, "").trim() || "navi"; }
  function formatBytes(bytes = 0) {
    if (!bytes) return "0 B";
    const units = ["B", "KB", "MB", "GB", "TB"];
    const exponent = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1);
    return `${(bytes / 1024 ** exponent).toFixed(exponent > 2 ? 1 : 0)} ${units[exponent]}`;
  }
  function relativeTime(value) {
    const seconds = Math.max(0, Math.round((Date.now() - Date.parse(value)) / 1000));
    if (seconds < 60) return "just now";
    if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`;
    if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ago`;
    return new Date(value).toLocaleDateString();
  }

  function setFilterDrawer(open) {
    elements.filterSidebar.classList.toggle("open", open);
    document.body.classList.toggle("filters-open", open);
    elements.openFilters.setAttribute("aria-expanded", String(open));
    if (open) elements.closeFilters.focus();
  }

  elements.search.addEventListener("input", event => { state.query = event.target.value; render(); });
  elements.sort.addEventListener("change", event => { state.sort = event.target.value; render(); });
  document.querySelectorAll("[data-status]").forEach(button => button.addEventListener("click", () => {
    state.status = button.dataset.status;
    render();
  }));
  document.querySelectorAll("[data-age]").forEach(button => button.addEventListener("click", () => {
    state.age = button.dataset.age;
    render();
  }));
  document.querySelectorAll("[data-video]").forEach(button => button.addEventListener("click", () => {
    state.video = button.dataset.video;
    render();
  }));
  elements.clearGenres.addEventListener("click", () => { state.genres.clear(); render(); });
  elements.clearAll.addEventListener("click", () => {
    state.status = "all";
    state.video = "all";
    state.age = "all";
    state.genres.clear();
    render();
  });
  elements.openFilters.addEventListener("click", () => setFilterDrawer(true));
  elements.closeFilters.addEventListener("click", () => setFilterDrawer(false));
  elements.filterScrim.addEventListener("click", () => setFilterDrawer(false));
  document.addEventListener("keydown", event => {
    if (event.key === "Escape" && elements.filterSidebar.classList.contains("open")) setFilterDrawer(false);
  });
  syncCatalog();
  setInterval(() => syncCatalog(true), 60_000);
})();
