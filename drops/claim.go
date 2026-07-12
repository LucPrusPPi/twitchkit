package drops

import (
	"encoding/json"
	"fmt"

	"github.com/LucPrusPPi/twitchkit/client"
)

// ClaimResult is one successful or failed claim attempt.
type ClaimResult struct {
	Claimable Claimable
	Err       error
}

// ClaimInventory loads Inventory and claims every finished drop.
// It returns one result per claimable found (before claims). Failed claims
// keep Err set; callers can retry those instance IDs.
func ClaimInventory(c *client.Client, userID string) ([]ClaimResult, error) {
	inv, err := c.Inventory(userID)
	if err != nil {
		return nil, err
	}
	return ClaimAll(c, userID, inv)
}

// ClaimAll claims each entry from ListClaimable(inv).
func ClaimAll(c *client.Client, userID string, inv json.RawMessage) ([]ClaimResult, error) {
	list := ListClaimable(inv)
	if len(list) == 0 {
		return nil, nil
	}
	out := make([]ClaimResult, 0, len(list))
	for _, item := range list {
		_, err := c.ClaimDrop(item.DropInstanceID, userID)
		out = append(out, ClaimResult{Claimable: item, Err: err})
	}
	return out, nil
}

// ClaimFirst claims FirstClaimable if present.
func ClaimFirst(c *client.Client, userID string, inv json.RawMessage) (*Claimable, error) {
	item, ok := FirstClaimable(inv)
	if !ok {
		return nil, nil
	}
	if _, err := c.ClaimDrop(item.DropInstanceID, userID); err != nil {
		return &item, fmt.Errorf("claim %s: %w", item.DropInstanceID, err)
	}
	return &item, nil
}
