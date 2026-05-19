# mtgo-cli

Telegram MTProto debug and invoke CLI built on [mtgo](https://github.com/mtgo-labs/mtgo) — with higher accuracy and performance.

## Install

```bash
go install github.com/mtgo-labs/mtgo-cli/cmd/mtgo-cli@latest
```

This installs the `mtgo-cli` binary into your Go binary directory. Make sure it
is on your `PATH`:

```bash
export PATH="$PATH:$(go env GOPATH)/bin"
```

## Quick Start

```bash
# Set credentials
export MTGO_CLI_API_ID=12345
export MTGO_CLI_API_HASH=your_hash

# Get current user/bot info
mtgo-cli get-me --bot-token YOUR_BOT_TOKEN

# Start a persistent listener
mtgo-cli listen --bot-token YOUR_BOT_TOKEN

# In another terminal — invoke through the listener (fast, no reconnect)
mtgo-cli invoke messages.sendMessage '{"peer":{"_":"inputPeerUser","user_id":123,"access_hash":456},"message":"hello"}'

# High-level commands
mtgo-cli send-message @username "Hello!"
mtgo-cli get-user @username
mtgo-cli resolve-peer +1234567890
```

## Commands

| Command | Description |
|---|---|
| `invoke <method> [json]` | Call any TL method (full or fast path) |
| `listen` | Persistent listener with IPC server |
| `trace` | Listen with correlation ID tracing |
| `methods [prefix]` | List available TL methods |
| `get-me` | Current user/bot info |
| `send-message <peer> <text>` | Send text message |
| `get-user <peer>` | Get user info (users.getFullUser) |
| `get-chat <peer>` | Get chat info (messages.getFullChat) |
| `list-chats` | List recent dialogs |
| `list-messages <peer>` | List recent messages |
| `resolve-peer <id>` | Resolve peer to access info |
| `export-session` | Export session string |
| `completion <shell>` | Shell completions (bash, zsh, fish) |
| `version` | Print version |

## Peer Formats

- `@username` — resolve by public username
- `+1234567890` — resolve by phone number
- `me` or `self` — current user
- `12345678` — numeric user/chat ID (auto-detected)
- `channel:123456` / `user:123456` — explicit type

## Authentication

Priority: CLI flags > environment variables > config file.

| Flag | Env | Description |
|---|---|---|
| `--api-id` | `MTGO_CLI_API_ID` | Telegram API ID |
| `--api-hash` | `MTGO_CLI_API_HASH` | Telegram API Hash |
| `--bot-token` | `MTGO_CLI_BOT_TOKEN` | Bot token |
| `--session` | `MTGO_CLI_SESSION` | Session string (auto-detect) |
| `--phone` | `MTGO_CLI_PHONE` | Phone number |

Config file: `~/.mtgo-cli.json` (mode 0600)

```json
{
  "api_id": 12345,
  "api_hash": "your_api_hash",
  "bot_token": "123:ABC"
}
```

## Session Strings

Auto-detected formats:
- mtgo native
- Telethon
- Pyrogram
- GramJS
- mtcute

Export: `mtgo-cli export-session`

## Performance vs gotg-cli

1. **InvokeWithRawByte** — fast path skips full TL decode
2. **Constructor cache** — pre-built name↔type map, zero per-call reflection
3. **Peer cache** — `SavePeers: true` caches username→ID locally
4. **Session auto-detect** — `session.String()` handles 5 formats
5. **Connection reuse** — IPC socket avoids re-auth overhead
6. **Pooled JSON encoding** — `sync.Pool` buffers for JSON marshal

## Architecture

```
cmd/mtgo-cli/     CLI entrypoint (cobra commands)
invoke/           Dual-path TL invoke engine
ipc/              Unix socket IPC server/client
trace/            Correlation ID tracer
internal/
  config/         Config loading (CLI > env > file)
  client/         mtgo client factory
```

## License

MIT
