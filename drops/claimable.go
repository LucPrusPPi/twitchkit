package drops

import "encoding/json"

// Claimable is a time-based drop that finished watching and can be claimed.
type Claimable struct {
	DropInstanceID string
	DropID         string
	DropName       string
	CampaignID     string
	CampaignName   string
	GameName       string
	CurrentMinutes float64
	RequiredMinutes float64
}

// ListClaimable returns every finished-but-unclaimed drop in an Inventory response.
func ListClaimable(inv json.RawMessage) []Claimable {
	var root map[string]any
	if err := json.Unmarshal(inv, &root); err != nil {
		return nil
	}
	campaigns := digArray(root, "data", "currentUser", "inventory", "dropCampaignsInProgress")
	var out []Claimable
	for _, raw := range campaigns {
		camp, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		campaignID, _ := camp["id"].(string)
		campaignName, _ := camp["name"].(string)
		gameName := ""
		if game, _ := camp["game"].(map[string]any); game != nil {
			gameName, _ = game["displayName"].(string)
			if gameName == "" {
				gameName, _ = game["name"].(string)
			}
		}
		for idx, dRaw := range asArray(camp["timeBasedDrops"]) {
			d, ok := dRaw.(map[string]any)
			if !ok {
				continue
			}
			self, _ := d["self"].(map[string]any)
			if self == nil {
				continue
			}
			if claimed, _ := self["isClaimed"].(bool); claimed {
				continue
			}
			cur := asFloat(self, "currentMinutesWatched")
			req := asFloat(d, "requiredMinutesWatched")
			if req <= 0 || cur < req {
				continue
			}
			instanceID, _ := self["dropInstanceID"].(string)
			if instanceID == "" {
				continue
			}
			dropID, _ := d["id"].(string)
			out = append(out, Claimable{
				DropInstanceID:  instanceID,
				DropID:          dropID,
				DropName:        dropLabel(d, idx),
				CampaignID:      campaignID,
				CampaignName:    campaignName,
				GameName:        gameName,
				CurrentMinutes:  cur,
				RequiredMinutes: req,
			})
		}
	}
	return out
}

// FirstClaimable returns the first claimable drop, if any.
func FirstClaimable(inv json.RawMessage) (Claimable, bool) {
	list := ListClaimable(inv)
	if len(list) == 0 {
		return Claimable{}, false
	}
	return list[0], true
}
