// Command watch_one is a minimal drops watch smoke test.
//
//	set TWITCH_TOKEN=oauth:...
//	set TWITCH_GAME=Just Chatting
//	go run ./examples/watch_one
//
// Optional: TWITCH_CHANNEL=login pins a channel instead of picking top drops stream.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/LucPrusPPi/twitchkit/client"
	"github.com/LucPrusPPi/twitchkit/drops"
	"github.com/LucPrusPPi/twitchkit/pubsub"
	"github.com/LucPrusPPi/twitchkit/watch"
)

func main() {
	token := os.Getenv("TWITCH_TOKEN")
	game := os.Getenv("TWITCH_GAME")
	channel := os.Getenv("TWITCH_CHANNEL")
	if token == "" {
		fmt.Fprintln(os.Stderr, "TWITCH_TOKEN is required")
		os.Exit(1)
	}
	if game == "" && channel == "" {
		fmt.Fprintln(os.Stderr, "set TWITCH_GAME and/or TWITCH_CHANNEL")
		os.Exit(1)
	}

	c := client.New(token)
	info, err := c.Validate()
	if err != nil {
		fmt.Fprintf(os.Stderr, "validate: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("ok as %s (%s)\n", info.Login, info.UserID)

	var stream *client.StreamInfo
	if channel != "" {
		stream, err = c.LiveStreamByLogin(channel)
		if err != nil {
			fmt.Fprintf(os.Stderr, "stream by login: %v\n", err)
			os.Exit(1)
		}
		if stream == nil {
			fmt.Fprintf(os.Stderr, "channel %q is offline\n", channel)
			os.Exit(1)
		}
	} else {
		list, err := c.DropsStreams(game)
		if err != nil {
			fmt.Fprintf(os.Stderr, "drops streams: %v\n", err)
			os.Exit(1)
		}
		stream = client.PickTopStream(list, "")
		if stream == nil {
			fmt.Fprintf(os.Stderr, "no live drops streams for %q\n", game)
			os.Exit(1)
		}
	}
	fmt.Printf("watching %s - %s (%d viewers)\n", stream.UserLogin, stream.GameName, stream.ViewerCount)

	if inv, err := c.Inventory(info.UserID); err == nil {
		p := drops.ParseInventoryProgress(inv, game)
		if p.DropName != "" {
			fmt.Printf("inventory progress: %.1f%% (%s)\n", p.Percent, p.DropName)
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	events := make(chan pubsub.Event, 16)
	go func() {
		if err := pubsub.Listen(ctx, token, info.UserID, events); err != nil && ctx.Err() == nil {
			fmt.Fprintf(os.Stderr, "pubsub: %v\n", err)
		}
	}()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case ev := <-events:
				switch e := ev.(type) {
				case pubsub.ProgressEvent:
					fmt.Printf("progress drop=%s %.0f/%.0f min\n", e.DropID, e.Current, e.Required)
				case pubsub.ClaimEvent:
					fmt.Printf("claimable %s\n", e.DropInstanceID)
					if _, err := c.ClaimDrop(e.DropInstanceID, info.UserID); err != nil {
						fmt.Fprintf(os.Stderr, "claim: %v\n", err)
					} else {
						fmt.Println("claimed")
					}
				}
			}
		}
	}()

	target := watch.FromStream(*stream, info.UserID)
	fmt.Println("watch loop every 55s - Ctrl+C to stop")
	if err := watch.Loop(ctx, c, target, 55*time.Second); err != nil && ctx.Err() == nil {
		fmt.Fprintf(os.Stderr, "watch: %v\n", err)
		os.Exit(1)
	}
}
