# mods.adon.hloth.dev backend

Built with Go and its standard library, nothing else.

## Build

Needs Go 1.27:

```sh
CGO_ENABLED=0 go build -trimpath -buildvcs=false -ldflags='-s -w' -o mods.adon.hloth.dev-backend .
```

The result is one static binary with no runtime dependencies.

## Run

```sh
STEAM_API_KEY=… JWT_SECRET=… \
PUBLIC_URL=https://mods.adon.hloth.dev \
DOWNLOADS_DIR=/srv/mods.adon.hloth.dev \
./mods.adon.hloth.dev-backend
```

| Variable | Default | Purpose |
| --- | --- | --- |
| `STEAM_API_KEY` | required | Steam Web API key, for `GetOwnedGames` |
| `JWT_SECRET` | required | Signs the `adon-mods-token` cookie. Changing it signs everyone out. |
| `PUBLIC_URL` | required | The website. Redirects and the Steam callback are built from it by appending paths, so a trailing slash is refused. |
| `DOWNLOADS_DIR` | required | Folder that holds the paid downloads. It may be missing at startup. |
| `HOST` | `127.0.0.1` | Listen address |
| `PORT` | `3001` | Listen port |

The backend reads the environment only, never a `.env` file. It exits with status 1 and names the missing or invalid variables, and stops gracefully on SIGTERM.

## Routes

| Route | What it does |
| --- | --- |
| `GET /auth/steam?file=<key>` | Redirects to Steam's OpenID login |
| `GET /auth/steam/callback` | Checks the OpenID answer with Steam, then ownership of At Dead of Night, sets the cookie, and returns to `/download-page/<key>` |
| `GET /download?file=<key>` | Sends the file to a holder of a valid cookie, with single-range resume; otherwise redirects to `/purchase-to-download/<key>` |

The keys and their files are in `internal/download`.

## Layout

`main.go` loads the config and runs the server until SIGTERM. Everything else is in `internal/`:

| Package | Owns |
| --- | --- |
| `config` | Environment variables and the upstream origins |
| `server` | Routes, handlers, graceful shutdown |
| `steam` | OpenID login URL and assertion checks, `check_authentication`, `GetOwnedGames` |
| `license` | The `adon-mods-token` JWT, its cookie, and the cache of ownership answers |
| `download` | The files, with single-range resume and a per-chunk write deadline |
| `upstream` | The HTTP client for Steam, with its timeouts, redirect policy, size cap and logging |
| `httpx` | Redirects and JSON errors without extras, panic recovery, the request-body deadline |
| `compat` | The Bun, Elysia and WHATWG URL rules the old backend's answers depended on |

## Tests

```sh
go test ./...
```

Unit tests sit next to each package. `internal/server` holds the end-to-end tests; some run the server on a real socket and listen on a free port in `127.0.0.1:39560-39599`.

`go build -tags parity` makes a binary for the parity harness. It reads `TEST_STEAM_COMMUNITY_ORIGIN` and `TEST_STEAM_API_ORIGIN` so a mock can stand in for Steam. Builds without the tag have no such hook.
