---
name: mtgo-cli
description: Use mtgo-cli — a fast Telegram MTProto CLI — to invoke any TL method, send messages and media, get user/chat info, create and manage groups, download files, resolve peers, and debug the Telegram API from the terminal. Use for quick Telegram automation without writing Go code: sending messages or photos, checking chats, creating groups, managing members, exporting sessions, tracing RPC calls, or exploring the TL schema. Triggers on "mtgo-cli", "Telegram CLI", "Telegram API from terminal", "send Telegram message command line", "MTProto debug", and any request to interact with Telegram via CLI. Prefer over raw MTProto libraries when the user wants immediate results without writing code. Covers bot and userbot auth (bot token, phone, QR, session strings from Telethon/Pyrogram), IPC listener for fast repeated commands, dual-path invoke engine with JSON and fast modes, and smart peer resolution (@username, phone, me, IDs).
---

# mtgo-cli

mtgo-cli is a CLI tool for calling the Telegram MTProto API directly from the terminal. It has higher accuracy and performance than gotg-cli thanks to its dual-path invoke engine, smart peer resolution, and auto-detection of 5 session formats.

## Quick Reference

**Binary:** `mtgo-cli` (or `go run ./cmd/mtgo-cli/` from the repo)
**Repo:** `github.com/mtgo-labs/mtgo-cli`
**Env vars:** `MTGO_CLI_API_ID`, `MTGO_CLI_API_HASH`, `MTGO_CLI_BOT_TOKEN`, `MTGO_CLI_SESSION`, `MTGO_CLI_PHONE`
**Config file:** `~/.mtgo-cli.json` (JSON with `api_id`, `api_hash`, `bot_token`, `session`, `phone`, `socket_path`, `store_path`)
**Session store:** `~/.mtgo-cli-session.json` (persistent JSON session, enable with `--store`)
**Related skill:** For mtgo library APIs, client patterns, and raw TL usage, install the mtgo skill with `npx skills add https://github.com/mtgo-labs/mtgo`.

## Authentication

ALWAYS prefer environment variables over CLI flags — CLI flags are visible in `ps aux`. Set these before running:

```bash
export MTGO_CLI_API_ID=12345
export MTGO_CLI_API_HASH=your_hash_here
export MTGO_CLI_BOT_TOKEN=123:ABC   # or MTGO_CLI_SESSION or MTGO_CLI_PHONE
```

The priority order is: env vars > CLI flags > config file. Choose ONE auth method — bot token, phone, or session string. If you have a session string from Telethon/Pyrogram/GramJS/mtcute, use `MTGO_CLI_SESSION` — it auto-detects the format.

For file-based secrets, prefer the `-file` variants to avoid leaking values in shell history:

```bash
mtgo-cli --api-id 12345 --api-hash-file ~/.secrets/api_hash --session-file ~/.secrets/session get-me
```

File-based flags (`--api-hash-file`, `--session-file`, `--bot-token-file`) require the file to have mode `0600` or stricter. Env vars are unset after reading to prevent child process inheritance.

### Userbot testing and BotFather setup

For normal bot-auth API checks, a bot token is enough. Do not require a userbot by default.

When the task involves testing a bot as a real user, creating groups, adding/promoting bots, clicking inline buttons, or creating a new bot through BotFather, ask the user whether they want to set up a userbot test account and/or create a bot via BotFather.

If they choose userbot testing, request:

- `api_id` and `api_hash` from https://my.telegram.org
- a real user session string via `MTGO_CLI_SESSION`, or a phone login via `MTGO_CLI_PHONE`
- the bot username to test, or permission to create one via BotFather

Bot sessions cannot create groups, add members, promote admins, or act as a real user in bot interaction tests. Use `MTGO_CLI_BOT_TOKEN` for bot-auth commands and user sessions for user-account operations.

For the fastest repeated usage, start a persistent listener that reuses one connection:

```bash
# Terminal 1 — start the listener
mtgo-cli listen &

# Terminal 2+ — all other commands now route through the IPC socket automatically
mtgo-cli get-me
mtgo-cli invoke messages.getHistory '{"peer":{"_":"inputPeerSelf"},"limit":10}'
```

## Commands

### Invoke — call any TL method

The core command. Takes a TL method name and optional JSON parameters. Interface fields (like `InputPeer`, `InputUser`) use `"_"` to specify the constructor type.

```bash
# Simple methods (no params needed)
mtgo-cli invoke help.getConfig

# Get self info
mtgo-cli invoke users.getFullUser '{"id":{"_":"inputUserSelf"}}'

# Send a message to a user
mtgo-cli invoke messages.sendMessage '{"peer":{"_":"inputPeerUser","user_id":123,"access_hash":456},"message":"Hello!","random_id":789}'

# Get message history
mtgo-cli invoke messages.getHistory '{"peer":{"_":"inputPeerSelf"},"limit":5,"offset_id":0,"offset_date":0,"add_offset":0,"max_id":0,"min_id":0,"hash":0}' --format json

# Fast path — skip TL decode for bulk operations
mtgo-cli invoke help.getConfig --fast

# Limit hex dump size for fast path (default 256 bytes, 0 = unlimited)
mtgo-cli invoke messages.getMessages --fast --max-bytes 0

# Pretty JSON output
mtgo-cli invoke users.getFullUser '{"id":{"_":"inputUserSelf"}}' --format json
```

**Discovering constructor names and parameters:** Use `mtgo-cli methods <prefix>` to find TL method names, then check https://corefork.telegram.org/methods for method parameters, return types, and known errors. Use https://corefork.telegram.org/schema for raw TL constructors and interface types.

### Methods — discover TL methods

List all available TL methods, optionally filtered by prefix:

```bash
mtgo-cli methods                # all methods
mtgo-cli methods messages.      # methods starting with "messages."
mtgo-cli methods users.get      # methods starting with "users.get"
mtgo-cli methods --format json  # machine-readable list
```

Only TL functions (methods you can call) are listed — constructors and types are excluded.

### High-level commands

These wrap common TL methods with smart peer resolution — the peer argument accepts `@username`, `+1234567890`, `me`, or a numeric ID:

```bash
mtgo-cli get-me                       # current user/bot info
mtgo-cli get-user @durov              # user profile (users.getFullUser)
mtgo-cli get-user +1234567890         # user by phone
mtgo-cli get-user me                  # self
mtgo-cli get-chat @channelname        # chat/channel/user info (auto-routes)
mtgo-cli send-message @username "Hi"  # send text message
mtgo-cli send-photo @username photo.jpg "caption"  # send photo
mtgo-cli send-file @username file.pdf            # send document
mtgo-cli download @username 1234                 # download media from message
mtgo-cli resolve-peer @username       # resolve to access info
mtgo-cli create-group "Test Suite"    # create basic group (userbot only)
mtgo-cli add-bot 5282748388 @bot      # add bot to group (userbot only)
mtgo-cli promote-bot channel:ID @bot  # promote bot to admin (userbot only)
mtgo-cli export-session               # export session string
```

For `send-message`, a random_id is generated automatically.

### List commands

```bash
mtgo-cli list-chats --limit 10        # recent dialogs
mtgo-cli list-messages @username --limit 20  # message history
```

### Listener and tracing

```bash
mtgo-cli listen     # persistent client + IPC server (all commands reuse this connection)
mtgo-cli trace      # listen + correlation ID logging (shows RPC request/response chains)
```

When a listener is running, all other commands automatically route through its Unix socket (`$XDG_RUNTIME_DIR/mtgo-cli.sock`). No re-authentication, no reconnection. This is the fastest way to run multiple commands.

### Group management (userbot only)

```bash
# Create a group
mtgo-cli create-group "Test Suite"
# → returns chat_id (e.g. 5282748388)

# Add a bot to the group (accepts raw chat ID or resolved peer)
mtgo-cli add-bot 5282748388 @my_bot

# Promote bot to admin (requires channel/supergroup)
mtgo-cli promote-bot channel:123456 @my_bot
```

### Media commands

```bash
# Send a photo
mtgo-cli send-photo @username photo.jpg "Check this out"

# Send a document
mtgo-cli send-file @username report.pdf

# Download media from a message
mtgo-cli download @username 2667
mtgo-cli download @username 2667 /tmp/output.jpg
```

### Utility

```bash
mtgo-cli version                    # build version
mtgo-cli completion bash            # shell completion script
mtgo-cli completion zsh
mtgo-cli completion fish
```

## Peer Format

These formats work everywhere a peer is needed:

| Input | Example | Resolution |
|---|---|---|
| `@username` | `@durov` | Public username lookup |
| Phone | `+1234567890` | Contact lookup |
| `me` / `self` | `me` | Current user |
| Numeric ID | `123456789` | Auto-detected as user/chat |
| Explicit channel | `channel:1234` | Forced channel type |
| Explicit chat | `chat:1234` | Forced basic group type |
| Explicit user | `user:1234` | Forced user type |

## JSON Constructor Format

TL interface fields (marked with `_` in the TL schema) need a constructor name to deserialize correctly:

```json
{
  "_": "inputPeerUser",
  "user_id": 123456,
  "access_hash": 789012345
}
```

The `"_"` key tells the deserializer which concrete type to create. This is the standard TL JSON encoding. Common constructors:

- `inputPeerSelf` — yourself
- `inputPeerUser` — a user (needs `user_id` + `access_hash`)
- `inputPeerChat` — a basic group (needs `chat_id`)
- `inputPeerChannel` — a channel/supergroup (needs `channel_id` + `access_hash`)
- `inputPeerEmpty` — empty placeholder
- `inputUserSelf` — your own user object
- `inputUser` — another user (needs `user_id` + `access_hash`)

## Common Workflows

### Get your own info
```bash
mtgo-cli get-me --format json
```

### Look up a user
```bash
mtgo-cli get-user @username --format json
```

### Send a message to a resolved peer
```bash
# Resolve first to get access_hash
PEER=$(mtgo-cli resolve-peer @username --format json | jq -r '.access_hash')
# Then send
mtgo-cli send-message @username "Hello from mtgo-cli!"
```

### Debug a TL method response
```bash
mtgo-cli trace &  # start tracing listener
# In another terminal:
mtgo-cli invoke messages.getHistory '{"peer":{"_":"inputPeerSelf"},"limit":1,"offset_id":0,"offset_date":0,"add_offset":0,"max_id":0,"min_id":0,"hash":0}'
# Watch the trace output for correlation IDs and timing
```

### Export a session for reuse
```bash
# To stdout (warning printed to stderr about security)
mtgo-cli export-session

# To file (mode 0600)
mtgo-cli export-session --output ~/.secrets/my_session.txt

# Later:
MTGO_CLI_SESSION=$(cat ~/.secrets/my_session.txt) mtgo-cli get-me
```

## Output Format

Use `--format json` for programmatic output. Default is colored text. Use `--no-color` to disable ANSI codes. The `--debug` flag logs full request/response payloads to stderr (contains sensitive data — don't use in shared terminals).

## Global Flags

| Flag | Type | Description |
|---|---|---|
| `--api-id` | int32 | Telegram API ID |
| `--api-hash` | string | API hash (prefer `--api-hash-file`) |
| `--api-hash-file` | string | Read API hash from file (mode 0600 required) |
| `--session` | string | Session string (prefer `--session-file`) |
| `--session-file` | string | Read session from file (mode 0600 required) |
| `--bot-token` | string | Bot token (prefer `--bot-token-file`) |
| `--bot-token-file` | string | Read bot token from file (mode 0600 required) |
| `--phone` | string | Phone number for user login |
| `--store` | string | JSON session store path (e.g. `~/.mtgo-cli-session.json`) |
| `--socket` | string | Override IPC socket path |
| `--config` | string | Config file path (default `~/.mtgo-cli.json`) |
| `--no-color` | bool | Disable colored output |
| `--debug` | bool | Verbose debug output to stderr |
| `--format` | string | Output format: `text` (default) or `json` |

## Security

- Never pass credentials via CLI flags — they appear in `ps aux`. Always use `MTGO_CLI_*` environment variables, file-based flags, or the config file.
- The config file (`~/.mtgo-cli.json`) warns if mode > `0600`. File-based flags reject files with mode > `0600`.
- The IPC socket is mode `0600` and only accepts same-UID connections (SO_PEERCRED check). Symlinks are rejected.
- Session strings grant full account access — treat them like passwords.
- `--debug` logs full payloads including session tokens to stderr.
- Env vars are unset after reading to prevent child process inheritance.
- The IPC server blocks dangerous methods: `auth.logOut`, `auth.resetAuthorizations`, `account.deleteAccount`, `account.setPrivacy`, `account.resetPassword`.
- `trace` redacts params/results for sensitive methods (`auth.*`, `account.*Password*`, `messages.requestWebView`, `messages.prolongWebView`) as `[REDACTED]`.

### IPC Limits

| Limit | Value |
|---|---|
| Max concurrent connections | 64 |
| Max request size | 10 MB |
| Read deadline | 30 seconds |
| Socket path | `$XDG_RUNTIME_DIR/mtgo-cli.sock` (fallback: `~/.local/run/`, then `/tmp/`) |
| Peer validation | Same UID only (SO_PEERCRED) |
| Symlink protection | Socket path and config files reject symlinks |

## Performance Tips

1. **Use the listener** — start `mtgo-cli listen` once, then all other commands reuse the connection via IPC. This avoids re-auth overhead.
2. **Use `--fast`** for bulk operations — skips full TL decode on responses, returns raw bytes.
3. **Use `--format json`** — parse with `jq` rather than parsing colored text output.
4. **Resolve peers once** — use `resolve-peer` to get the access hash, then use numeric IDs in subsequent calls.
5. **Bypass JSON for precision** — `invoke` routes through JSON which can lose precision on large int64 values (e.g., access_hash). Commands `send-message`, `get-user`, and `get-chat` bypass this by using the Go API directly. For raw `invoke` calls with access_hash, prefer the `--fast` flag or verify the value is exact.

## Error Handling

Errors are printed to stderr in the format: `Error: RPC error: CODE: message`

Common errors:
- `FLOOD_WAIT: N` — wait N seconds before retrying
- `PEER_ID_INVALID` — the peer ID or access hash is wrong (re-resolve)
- `RANDOM_ID_EMPTY` — the method requires a `random_id` field
- `USER_IS_BOT` — bots can't perform this action on themselves
- `SESSION_PASSWORD_NEEDED` — 2FA is required (user must provide password)
