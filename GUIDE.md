# HomeStack MVP – Build & Run Guide

This guide shows how to build the `homestack` binary and test the MVP end‑to‑end like a real user on your machine.

---

## 1. Prerequisites

- **Go:** 1.21 or later installed (`go version`).
- **OS:** macOS, Linux, or Windows (commands below use macOS/Linux paths; adjust for Windows).

All commands below assume this directory:

```bash
cd /Users/nabilmouzouna/School/PFE/homestack-mvp
```

---

## 2. Build the binary

```bash
cd /Users/nabilmouzouna/School/PFE/homestack-mvp
go build -o homestack ./cmd/homestack
```

You should now have a `homestack` executable in this folder.

### Build for all OS (cross-compile)

From the project root you can build binaries for macOS (Intel + Apple Silicon), Linux, and Windows in one go:

```bash
chmod +x scripts/build-all.sh
./scripts/build-all.sh
```

Outputs go to **`application/`** (same folder linked from the README for downloads):

| File | Platform |
|------|----------|
| `homestack-darwin-arm64` | Apple Silicon Mac (M1/M2/M3) |
| `homestack-darwin-amd64` | Intel Mac |
| `homestack-linux-amd64` | Linux (x86_64) |
| `homestack-linux-arm64` | Linux (ARM64) |
| `homestack-windows-amd64.exe` | Windows (x86_64) |
| `homestack-windows-arm64.exe` | Windows (ARM64) |

Copy the right binary from `application/` to another machine; no Go or source code is required. Commit the `application/` folder when pushing release binaries so users can download from the repo. To avoid committing binaries, add `application/` to `.gitignore` and use CI (e.g. GitHub Actions) to publish artifacts instead.

---

## 3. One‑time setup: create the admin user

Run `init` once to create the first admin user in the local SQLite DB:

```bash
./homestack init \
  --admin-user admin \
  --admin-password your-strong-password
```

Notes:

- App data (including `homestack.db`) is created under the **app data directory** (by default `homestack_data/` next to the binary).
- You only need to run `init` again if you intentionally reset/clear the app data directory.

---

## 4. Choose a storage root

Pick a folder that will act as the “local cloud” root (all files are stored under this directory). For example:

```bash
mkdir -p /Users/nabilmouzouna/School/PFE/homestack-storage
```

This folder is the **only** place the server will read/write files (enforced by `storage.Guard`).

---

## 5. Run the server

Start the HTTP server on port 8080 and point it at your storage root:

```bash
./homestack run \
  --storage-root /Users/nabilmouzouna/School/PFE/homestack-storage \
  --port 8080
```

You should see a log line similar to:

```text
Listening on http://localhost:8080 (storage root: /Users/nabilmouzouna/School/PFE/homestack-storage)
```

Keep this terminal open; the process must stay running.

---

## 6. Use the app from a browser

### 6.1 Login

1. Open a browser on the host machine and visit:

   ```text
   http://localhost:8080 OR your-machine's-ip-address:8080
   ```

2. You should see the **login page** (dark theme, Poppins font, orange accent).
3. Log in with the admin credentials you created in step 3:

   - Username: `admin`
   - Password: `your-strong-password`

4. On success, you should be redirected to the **dashboard/file browser**.

If you enter wrong credentials, you should see an error message like “Invalid username or password.”

### 6.2 Dashboard – list, upload, download, delete

On the dashboard:

- You should see:
  - Top app bar with “HomeStack” and user avatar.
  - Breadcrumb showing the current path (e.g. `Home`).
  - Toolbar with **Upload** (and optionally **New folder**).
  - File list area (empty state if there are no files yet).

#### Upload a file

1. Click **Upload**.
2. Choose a small file from your machine.
3. After the upload completes:
   - The file should appear in the list with correct name and size.
   - The file should exist on disk under:

     ```bash
     ls /Users/nabilmouzouna/School/PFE/homestack-storage
     ```

#### Download a file

1. In the file list, use the **Download** action for that file.
2. The browser should download it; opening the downloaded file should match the original.

#### Delete a file

1. Use the **Delete** action (trash icon or button) for the file.
2. Confirm the deletion in the UI.
3. The file should disappear from:
   - The file list in the dashboard.
   - The storage folder on disk.

---

## 7. Admin section (user management)

If your UI task is fully implemented, logging in as an admin should also give you access to an **Admin** section:

1. Use the user menu (top-right) to navigate to **Admin**.
2. On the Users screen, you should see:
   - `admin` with an “Admin” badge (not removable).
   - A form to **Add user** (username + password + confirm password).
3. Adding a user:
   - Creates a new user row in the list.
   - Allows that user to log in via the login page and use the same storage root.

---

## 8. Testing from another device on the LAN (optional)

1. Find your host machine’s LAN IP (on macOS, for Wi‑Fi, for example):

   ```bash
   ipconfig getifaddr en0
   ```

2. From a phone or another laptop on the same Wi‑Fi, open:

   ```text
   http://<YOUR-LAN-IP>:8080
   ```

3. Repeat the login and file operations. Everything should work the same as on localhost.

---

## 9. Quick MVP checklist

You can consider the MVP manually verified when all of these hold:

- [ ] `homestack init` creates an admin user successfully (no errors).
- [ ] `homestack run` starts without errors for a valid storage root.
- [ ] Login works; invalid credentials are rejected with a clear error.
- [ ] After login, you see the dashboard UI, not a plain text page.
- [ ] You can upload files and they appear both in the UI and on disk under the storage root.
- [ ] You can download files and they match the originals.
- [ ] You can delete files and they disappear from both UI and disk.
- [ ] Unauthenticated requests to file APIs do not return data (they are rejected).

If any step fails, capture:

- The command you ran.
- The URL you opened.
- The exact error message or behavior.

Then you can use this information to debug or to ask for help with a precise failing scenario.

---

## 10. Run the binary on another Mac (or other OS)

The built binary is **self-contained**: no Go, Node, or source code is needed on the target machine.

1. Download the right binary from the **[`application/`](application/)** folder (see README table), or copy it from your local `application/` after running `./scripts/build-all.sh`.
   - **Apple Silicon Mac:** `homestack-darwin-arm64` (rename to `homestack` if you like).
   - **Intel Mac:** `homestack-darwin-amd64`.
2. On the target Mac, in a terminal:
   ```bash
   chmod +x homestack
   ./homestack init --admin-user admin --admin-password your-password
   ./homestack run --storage-root /path/to/folder --port 8080
   ```
3. Open `http://localhost:8080` (or `http://<that-Mac-IP>:8080` from another device on the LAN).

App data (SQLite, users) is created next to the binary in `homestack_data/` by default, or set `HOMESTACK_APP_DATA` to another directory.

