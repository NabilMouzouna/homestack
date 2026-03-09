# HomeStack — Project (PFE)

**Cross-OS local cloud server.** One machine runs a single binary; users on the same LAN access a web UI to upload and download files. No cloud dependency, no Node.js on the host, runs fully offline.  
**License:** [MIT](LICENSE)

---

## Download the binary

Pre-built binaries for each OS are in the **[`application/`](application/)** folder. Download the file that matches your system:

| OS | Architecture | File |
|----|---------------|------|
| **macOS** | Apple Silicon (M1/M2/M3) | [`application/homestack-darwin-arm64`](application/homestack-darwin-arm64) |
| **macOS** | Intel (x86_64) | [`application/homestack-darwin-amd64`](application/homestack-darwin-amd64) |
| **Linux** | x86_64 | [`application/homestack-linux-amd64`](application/homestack-linux-amd64) |
| **Linux** | ARM64 | [`application/homestack-linux-arm64`](application/homestack-linux-arm64) |
| **Windows** | x86_64 | [`application/homestack-windows-amd64.exe`](application/homestack-windows-amd64.exe) |
| **Windows** | ARM64 | [`application/homestack-windows-arm64.exe`](application/homestack-windows-arm64.exe) |

After downloading, see **[GUIDE.md](GUIDE.md)** for how to run: one-time `init` (create admin user), then `run` with a storage root and port. No Go or source code required on the target machine.

---

## 1. Project description

HomeStack turns a computer into a small, self-hosted file server (“local cloud”). The operator runs one binary, points it at a folder (or external drive / USB), and anyone on the same LAN can open a browser, sign in, and manage files there.

- **Single binary:** Copy and run; no install of runtimes (e.g. Node.js) on the host.
- **Multi-user from day one:** An admin account is created at first run; the admin creates other users via CLI or the embedded web UI. Users sign in with username/password and get access to the same storage root (no per-file RBAC in MVP).
- **Security:** The app only reads and writes under the chosen storage root (including delete/update). It does not touch system files or escalate privileges. All paths are validated to prevent escape.
- **Lightweight:** Target &lt;100 MB RAM and configurable CPU/RAM limits (e.g. max 50%).
- **Offline:** The web UI (Go templates + HTMX) and all assets are embedded in the binary; no CDN or internet required.

**MVP scope:** Centralized server with upload/download/list/delete, multi-user auth, admin via CLI and embedded UI. **Out of scope for MVP:** RBAC, file-type restrictions, HTTPS, auto-detect drives, public/remote access, developer API.

---

## 2. Requirements

### Functional

- One host runs the app; clients connect from other devices on the **same LAN**.
- **Storage root:** User selects a folder (or drive/USB) at startup; all file operations are confined to that path.
- **Users:** Admin created at first run (e.g. `homestack init --admin-user X --admin-password Y`). Admin can add users via **CLI** (`homestack user add bob`) and via **embedded web UI** (admin-only section when logged in as admin).
- **Web UI:** Login, file browser (list/upload/download/delete). Admin section for user management. Served by the same binary; no external scripts (fully offline).
- **Configurable:** Port (default e.g. 8080), max users (default 50), storage root (config file or flags).
- **Platforms:** Windows, macOS, Linux (single codebase, cross-compiled).

### Non-functional

- **Deployment:** Single static binary per OS; run in user mode when possible (high port to avoid root).
- **Resources:** &lt;100 MB RAM; configurable max CPU/RAM (e.g. 50%).
- **Data:** App data (users, sessions, config) lives in a fixed app data directory; file storage lives only under the selected path. No cloud or external services for MVP.
- **Discovery (optional):** Best-effort LAN hostname (e.g. mDNS) — low priority for MVP.

---

## 3. System design

### High-level

```
                    LAN
  [Client browsers]  <--->  [Host: homestack binary]
                                    |
                    +---------------+---------------+
                    |                               |
              App data dir                   Storage root
              (SQLite, config)               (user files)
              next to binary                 (drive/USB/folder)
```

- **Host:** Runs `homestack` (Go binary). Binds to a configurable port (e.g. 8080). Serves HTTP (no HTTPS in MVP).
- **App data:** One directory (e.g. `./homestack_data/` or `$XDG_CONFIG_HOME/homestack` / `%AppData%/HomeStack`). Contains SQLite DB (users, sessions, config) and optionally config file. Never written into the storage root.
- **Storage root:** User-selected path (drive, USB, or folder). All upload/download/list/delete operations are restricted to this tree; path validation prevents escape (symlinks, `..`).
- **Clients:** Browsers on the same LAN; access via `http://<host-ip>:port` (or hostname if mDNS is used). Authenticate with username/password; session via signed cookie or token. Admin sees an extra section for user management.

### Components

| Component        | Responsibility |
|-----------------|----------------|
| **CLI**         | `homestack init`, `homestack user add/list/remove`, `homestack run`; config and admin user creation. |
| **HTTP server** | Serves web UI (Go templates + HTMX) and REST-style API (list/upload/download/delete). Auth middleware on all routes; admin-only middleware on `/admin` (or equivalent). |
| **Auth**        | Password hashing (bcrypt/scrypt), session (signed cookie or JWT). Admin flag in DB; only admin can access admin UI/API and create users. |
| **Storage**     | Path canonicalization and guard: all file I/O under storage root only. |
| **DB**          | SQLite in app data dir: users, sessions, config. Migrations in repo. |

### Key decisions (ADRs)

- **ADR-001:** Go, single binary, web UI as embedded static assets.
- **ADR-002:** App data (SQLite, config) in app data dir; file storage only on user-selected path.
- **ADR-003:** Multi-user auth; admin via CLI and embedded UI; no RBAC in MVP.
- **ADR-004:** Go `html/template` + HTMX; HTMX embedded so the app runs fully offline.

Detailed ADRs live in `Tandem/docs/adr/`.

---

## 4. Tech stack

| Layer      | Choice |
|-----------|--------|
| **Language** | Go 1.21+ |
| **Server**   | `net/http` or minimal router (e.g. chi, echo) |
| **CLI**      | `flag` or `cobra` |
| **Frontend** | Go `html/template` + HTMX (embedded in binary; no CDN) |
| **Database** | SQLite (single file in app data dir); `database/sql` + cgo-free driver preferred |
| **Auth**     | Bcrypt/scrypt for passwords; signed cookie or JWT for sessions |
| **Deploy**   | Single static binary per OS (Windows, macOS, Linux); CI builds GOOS/GOARCH matrix |

### Repo layout (Tandem)

- `cmd/homestack` — main and CLI entrypoint  
- `internal/server` — HTTP handlers, routes  
- `internal/auth` — sessions, password hashing  
- `internal/storage` — path validation, file ops under root  
- `internal/db` — SQLite schema, queries, migrations  
- `internal/config` — config and env  
- `web/` — Go templates and static assets (HTMX, CSS) embedded in binary  
- `docs/adr/`, `docs/requests/`, `docs/specs/` — ADRs, request, specs  

Stack and conventions are detailed in `Tandem/ai-context/stack.md`.

---

## 5. Cost and operations (MVP)

- **Hosting:** None; runs on the user’s machine.
- **Run cost:** Only power and hardware; no subscription or external services.
- **Scaling:** Single host; max users configurable (default 50).
- **Ops:** User starts the binary (manually or via a service). Updates = replace binary and restart. No managed DB or Kubernetes.
- **External services:** None for MVP. Optional later: mDNS (LAN), HTTPS with user-provided cert.

---

## 6. Repository structure (this repo)

- **`application/`** — Pre-built binaries for each OS (download from here; see **Download the binary** above).
- **`GUIDE.md`** — Step-by-step build, run, and test instructions for users and testers.
- **`Tandem/`** — Development workspace: tasks, ADRs, specs, scripts, AI context (when present in the repo).

For implementation details, task breakdown, and conventions, see `Tandem/docs/` and `Tandem/ai-context/`.
