//go:build cgo

// Package capi exposes a minimal C ABI for embedding twitchkit in non-Go apps.
//
// Build a shared library:
//
//	go build -buildmode=c-shared -o libtwitchkit.so ./cmd/c-shared
//	go build -buildmode=c-shared -o twitchkit.dll ./cmd/c-shared
package capi

/*
#include <stdlib.h>
*/
import "C"
import (
	"encoding/json"
	"sync"
	"unsafe"

	"github.com/LucPrusPPi/twitchkit/auth"
	"github.com/LucPrusPPi/twitchkit/client"
)

var (
	mu      sync.Mutex
	clients = map[uintptr]*client.Client{}
	nextID  uintptr = 1
)

func store(c *client.Client) uintptr {
	mu.Lock()
	defer mu.Unlock()
	id := nextID
	nextID++
	clients[id] = c
	return id
}

func get(id uintptr) *client.Client {
	mu.Lock()
	defer mu.Unlock()
	return clients[id]
}

func del(id uintptr) {
	mu.Lock()
	defer mu.Unlock()
	delete(clients, id)
}

func cstr(s string) *C.char {
	return C.CString(s)
}

func gostr(p *C.char) string {
	if p == nil {
		return ""
	}
	return C.GoString(p)
}

//export TWITCHKIT_Create
func TWITCHKIT_Create(token *C.char) C.uintptr_t {
	c := client.New(gostr(token))
	return C.uintptr_t(store(c))
}

//export TWITCHKIT_Destroy
func TWITCHKIT_Destroy(handle C.uintptr_t) {
	del(uintptr(handle))
}

//export TWITCHKIT_Validate
// Returns JSON {"login":"...","user_id":"..."} on success, or NULL on failure.
// Caller must TWITCHKIT_Free the result.
func TWITCHKIT_Validate(handle C.uintptr_t) *C.char {
	c := get(uintptr(handle))
	if c == nil {
		return nil
	}
	info, err := c.Validate()
	if err != nil {
		return nil
	}
	b, err := json.Marshal(map[string]string{
		"login":   info.Login,
		"user_id": info.UserID,
	})
	if err != nil {
		return nil
	}
	return cstr(string(b))
}

//export TWITCHKIT_ClaimDrop
// Returns 1 on HTTP/GQL success (response body ignored), 0 on failure.
func TWITCHKIT_ClaimDrop(handle C.uintptr_t, dropInstanceID, userID *C.char) C.int {
	c := get(uintptr(handle))
	if c == nil {
		return 0
	}
	_, err := c.ClaimDrop(gostr(dropInstanceID), gostr(userID))
	if err != nil {
		return 0
	}
	return 1
}

//export TWITCHKIT_SendWatch
func TWITCHKIT_SendWatch(
	handle C.uintptr_t,
	channelLogin, channelID, broadcastID, userID, gameName, gameID *C.char,
) C.int {
	c := get(uintptr(handle))
	if c == nil {
		return 0
	}
	err := c.SendWatch(
		gostr(channelLogin),
		gostr(channelID),
		gostr(broadcastID),
		gostr(userID),
		gostr(gameName),
		gostr(gameID),
	)
	if err != nil {
		return 0
	}
	return 1
}

//export TWITCHKIT_NormalizeToken
// Caller must TWITCHKIT_Free the result.
func TWITCHKIT_NormalizeToken(token *C.char) *C.char {
	return cstr(auth.Normalize(gostr(token)))
}

//export TWITCHKIT_Free
func TWITCHKIT_Free(p *C.char) {
	if p != nil {
		C.free(unsafe.Pointer(p))
	}
}
