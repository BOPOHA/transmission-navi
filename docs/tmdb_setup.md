# Create a TMDB API credential

TMDB metadata enrichment requires a free developer credential. Each person who
deploys Transmission Navi must create their own credential; do not share a
token or reuse one copied from another installation.

## Request access

1. Create or sign in to your account at [The Movie Database](https://www.themoviedb.org/).
2. Open **Profile → Settings → API**, then request an API credential.
3. Choose the free **Developer** plan and select **Hardware/Embedded** as the
   type of use for a self-hosted NAS installation.
4. Complete the application form with your own accurate contact information.
   Use a project URL or another URL you control; do not claim an unrelated
   website as your application URL.
5. Read and accept TMDB's terms only if you agree with them, then submit the
   request.
6. After approval, copy the read-only **API Read Access Token** from the API
   settings page. Treat it like a password.

Suggested non-personal application fields:

| Field | Example |
| --- | --- |
| Application Name | `Transmission Navi` |
| Application URL | `https://github.com/your-account/your-navi-repository` |
| Type of Use | `Hardware/Embedded` |

Example **Application Summary**:

> Transmission Navi is a self-hosted, non-commercial media catalog for a personal
> Transmission installation. It parses movie titles from locally managed
> downloads and uses the TMDB API to display metadata such as posters, release
> years, genres, ratings, and age certifications. The metadata is shown only to
> users of the local installation. The application does not sell TMDB data or
> provide public downloads.

Adapt the summary if your actual deployment differs. In particular, do not
describe a commercial or public service as personal use.

## Keep the credential private

Do not put the token in Git, `config.template.json`, screenshots, issue reports,
or the browser-side JavaScript. The Go indexer can read it from the runtime
environment:

```sh
export NAVI_TMDB_READ_ACCESS_TOKEN='your-own-token'
```

Alternatively, copy `config.template.json` to the ignored `config.json` and set
the token there:

```json
{
  "tmdb_read_access_token": "your-own-token",
  "tmdb_language": "en-US",
  "tmdb_cache_path": ".cache/tmdb.json",
  "tmdb_enrich_unmapped": false
}
```

For additional Russian-title resolution, obtain a separate token from the
[PoiskKino API documentation](https://api.kinopoisk.dev/documentation) and add:

```json
{
  "poiskkino_api_token": "your-own-poiskkino-token"
}
```

PoiskKino is optional and is used only to resolve a match to a TMDB ID. Navi
then loads the final title, poster, genres, rating, and certification from
TMDB. The resolver is a third-party community service rather than an official
Yandex Kinopoisk API; review its current terms before enabling it.

Run it with `navi-indexer -config config.json`. Environment variables override
values from the JSON file. With `tmdb_enrich_unmapped` set to `false`, the
indexer enriches only torrents that map to the configured SMB root. This avoids
unnecessary API requests for media the UI cannot play. Runtime secrets must
remain on the machine running the Go indexer, never in `catalog.json` or the
static web directory.

Restrict a local configuration file to your account:

```sh
chmod 600 config.json
```

The form also asks for legal name, email, telephone number, and postal address.
Enter your own details directly on TMDB; this project neither needs nor stores
that personal information.
