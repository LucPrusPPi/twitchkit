# twitchkit

Twitch Drops **client kit** for people who build their own farmers.

Go module for day-to-day use. Optional C ABI (`capi`) if you need to embed the same logic in C#, C++, Python, etc.

This is not an official Twitch SDK. GQL hashes, PubSub topics, and spade URLs can change. Respect Twitch ToS.

## What you get

| Need | Package / API |
|------|----------------|
| Normalize + validate token | `auth` |
| GQL / Helix / minute-watched | `client` |
| Inventory progress | `drops.ParseInventoryProgress` |
| Finished drops ready to claim | `drops.ListClaimable` / `ClaimInventory` |
| Watch loop (55s) | `watch.Loop` |
| Drop progress / claim WS | `pubsub.Listen` |
| Soft retries | `retry.Do` |
| Pick busiest stream | `client.PickTopStream` |
| Proxy / timeout / UA | `client.NewWithOptions` |

Not included on purpose: multi-account orchestration, UI, databases, license servers.

## Install

```bash
go get github.com/LucPrusPPi/twitchkit@latest
```

## Minimal farmer shape

```go
c := client.New(token)
info, err := c.Validate()
if auth.IsInvalid(err) {
    // drop this account / ask user to re-login
}

_, _ = drops.ClaimInventory(c, info.UserID)

streams, _ := c.DropsStreams("Game Name")
stream := client.PickTopStream(streams, "")
target := watch.FromStream(*stream, info.UserID)

events := make(chan pubsub.Event, 16)
go pubsub.Listen(ctx, c.Token(), info.UserID, events)
go watch.Loop(ctx, c, target, 0) // 0 => 55s
```

## Examples

```bash
export TWITCH_TOKEN='oauth:...'
export TWITCH_GAME='Just Chatting'
# optional:
# export TWITCH_CHANNEL='some_streamer'
# export TWITCH_PROXY='http://127.0.0.1:8080'

go run ./examples/farm_simple   # reference farmer
go run ./examples/watch_one     # thinner smoke test
```

## Client options

```go
c, err := client.NewWithOptions(token, client.Options{
    Timeout: 45 * time.Second,
    Proxy:   "http://127.0.0.1:8080",
})
```

## Errors

- `auth.ErrInvalidToken` / `auth.IsInvalid(err)` — stop farming that token.
- `client.StatusError` — HTTP status from GQL/Helix/spade; `client.IsTransient(err)` for retry decisions.
- `retry.Do` retries transient failures and never retries invalid tokens.

## C ABI (optional)

Needs CGO and a C toolchain. On Windows, MinGW `gcc` is the reliable path for `c-shared` (MSVC/clang often trip over Go's cgo flags):

```bash
# Linux
CGO_ENABLED=1 go build -buildmode=c-shared -o libtwitchkit.so ./cmd/c-shared

# Windows (MinGW gcc in PATH)
set CGO_ENABLED=1
go build -buildmode=c-shared -o twitchkit.dll ./cmd/c-shared
```

Header: [`include/twitchkit.h`](include/twitchkit.h)

## License

MIT
