# twitchkit

Universal **Twitch Drops client library** in Go. Use it to build your own farmer, bot, or embed the client into another app via C ABI.

This is **not** an official Twitch SDK. Unofficial GQL / PubSub / spade paths may change; respect Twitch ToS and local law.

## Install (Go)

```bash
go get github.com/LucPrusPPi/twitchkit@latest
```

```go
import (
    "github.com/LucPrusPPi/twitchkit/client"
    "github.com/LucPrusPPi/twitchkit/watch"
)
```

## Packages

| Package | Role |
|---------|------|
| `auth` | Token normalize + OAuth validate |
| `gql` | Persisted-query builders |
| `client` | GQL / Helix / device bootstrap / minute-watched |
| `drops` | Inventory / campaign progress parsing |
| `watch` | Minute-watched loop |
| `pubsub` | `user-drop-events` WebSocket |
| `capi` | C ABI wrappers |

## Quick start

```bash
export TWITCH_TOKEN='oauth:...'
export TWITCH_GAME='Just Chatting'
# optional pin:
# export TWITCH_CHANNEL='some_streamer'
go run ./examples/watch_one
```

## Embed (C / C# / Python / …)

Build a shared library (**requires CGO + gcc/clang/MSVC**):

```bash
# Linux
CGO_ENABLED=1 go build -buildmode=c-shared -o libtwitchkit.so ./cmd/c-shared

# Windows (MinGW or LLVM in PATH)
set CGO_ENABLED=1
go build -buildmode=c-shared -o twitchkit.dll ./cmd/c-shared
```

Header: [`include/twitchkit.h`](include/twitchkit.h)

```c
twitchkit_handle h = TWITCHKIT_Create("oauth:...");
char *info = TWITCHKIT_Validate(h);
/* ... */
TWITCHKIT_Free(info);
TWITCHKIT_Destroy(h);
```

## API sketch

```go
c := client.New(token)
info, err := c.Validate()
streams, err := c.DropsStreams("Game Name")
stream, err := c.LiveStreamByLogin("channel")
_ = c.SendWatch(login, channelID, broadcastID, userID, gameName, gameID)
_, _ = c.ClaimDrop(dropInstanceID, userID)
```

## License

MIT
