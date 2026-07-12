// Command farm_simple is a reference one-account farmer.
//
// Flow:
//  1. validate token
//  2. claim anything already finished in inventory
//  3. pick a drops stream (or pinned channel)
//  4. send minute-watched on an interval
//  5. listen for PubSub progress / claim events
//
//	set TWITCH_TOKEN=oauth:...
//	set TWITCH_GAME=Just Chatting
//	go run ./examples/farm_simple
//
// Optional:
//
//	TWITCH_CHANNEL=login   pin a channel
//	TWITCH_PROXY=http://127.0.0.1:8080
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/LucPrusPPi/twitchkit/auth"
	"github.com/LucPrusPPi/twitchkit/client"
	"github.com/LucPrusPPi/twitchkit/drops"
	"github.com/LucPrusPPi/twitchkit/pubsub"
	"github.com/LucPrusPPi/twitchkit/retry"
	"github.com/LucPrusPPi/twitchkit/watch"
)

func main() {
	log.SetFlags(log.Ltime | log.Lmsgprefix)
	log.SetPrefix("farm ")

	token := os.Getenv("TWITCH_TOKEN")
	game := os.Getenv("TWITCH_GAME")
	channel := os.Getenv("TWITCH_CHANNEL")
	proxy := os.Getenv("TWITCH_PROXY")
	if token == "" {
		log.Fatal("TWITCH_TOKEN is required")
	}
	if game == "" && channel == "" {
		log.Fatal("set TWITCH_GAME and/or TWITCH_CHANNEL")
	}

	c, err := client.NewWithOptions(token, client.Options{Proxy: proxy})
	if err != nil {
		log.Fatalf("client: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var info auth.TokenInfo
	err = retry.Do(ctx, retry.Config{}, func() error {
		var e error
		info, e = c.Validate()
		return e
	})
	if err != nil {
		if auth.IsInvalid(err) {
			log.Fatalf("token rejected — re-auth required: %v", err)
		}
		log.Fatalf("validate: %v", err)
	}
	log.Printf("signed in as %s (%s)", info.Login, info.UserID)

	if results, err := drops.ClaimInventory(c, info.UserID); err != nil {
		log.Printf("inventory claim sweep failed: %v", err)
	} else {
		for _, r := range results {
			if r.Err != nil {
				log.Printf("claim failed %s (%s): %v", r.Claimable.DropName, r.Claimable.DropInstanceID, r.Err)
				continue
			}
			log.Printf("claimed %s", r.Claimable.DropName)
		}
	}

	stream, err := resolveStream(ctx, c, game, channel)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("watching %s — %s (%d viewers)", stream.UserLogin, stream.GameName, stream.ViewerCount)

	if inv, err := c.Inventory(info.UserID); err == nil {
		p := drops.ParseInventoryProgress(inv, game)
		if p.DropName != "" {
			log.Printf("progress %.1f%% on %s", p.Percent, p.DropName)
		}
	}

	events := make(chan pubsub.Event, 32)
	go func() {
		if err := pubsub.Listen(ctx, c.Token(), info.UserID, events); err != nil && !errors.Is(err, context.Canceled) {
			log.Printf("pubsub stopped: %v", err)
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
					log.Printf("drop %s: %.0f / %.0f min", e.DropID, e.Current, e.Required)
				case pubsub.ClaimEvent:
					log.Printf("claim ready: %s", e.DropInstanceID)
					if err := retry.Do(ctx, retry.Config{Attempts: 3}, func() error {
						_, err := c.ClaimDrop(e.DropInstanceID, info.UserID)
						return err
					}); err != nil {
						log.Printf("claim error: %v", err)
					} else {
						log.Printf("claimed %s", e.DropInstanceID)
					}
				}
			}
		}
	}()

	// Periodic inventory sweep catches claimables if PubSub is quiet.
	go func() {
		t := time.NewTicker(5 * time.Minute)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				results, err := drops.ClaimInventory(c, info.UserID)
				if err != nil {
					log.Printf("claim sweep: %v", err)
					continue
				}
				for _, r := range results {
					if r.Err != nil {
						log.Printf("claim sweep fail %s: %v", r.Claimable.DropName, r.Err)
						continue
					}
					log.Printf("claim sweep ok %s", r.Claimable.DropName)
				}
			}
		}
	}()

	target := watch.FromStream(*stream, info.UserID)
	log.Print("minute-watched every 55s — Ctrl+C to stop")
	err = watch.Loop(ctx, c, target, 55*time.Second)
	if err != nil && !errors.Is(err, context.Canceled) {
		log.Fatalf("watch: %v", err)
	}
}

func resolveStream(ctx context.Context, c *client.Client, game, channel string) (*client.StreamInfo, error) {
	var stream *client.StreamInfo
	err := retry.Do(ctx, retry.Config{Attempts: 4, Base: time.Second}, func() error {
		if channel != "" {
			s, err := c.LiveStreamByLogin(channel)
			if err != nil {
				return err
			}
			if s == nil {
				return fmt.Errorf("channel %q is offline", channel)
			}
			stream = s
			return nil
		}
		list, err := c.DropsStreams(game)
		if err != nil {
			return err
		}
		stream = client.PickTopStream(list, "")
		if stream == nil {
			return fmt.Errorf("no live drops streams for %q", game)
		}
		return nil
	})
	return stream, err
}
