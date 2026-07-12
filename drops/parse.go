package drops

import (
	"encoding/json"
	"strings"
)

// Progress is a parsed farming progress snapshot.
type Progress struct {
	Percent  float64
	DropName string
}

// ParseInventoryProgress reads dropCampaignsInProgress from an Inventory response.
func ParseInventoryProgress(inv json.RawMessage, targetGame string) Progress {
	var root map[string]any
	if err := json.Unmarshal(inv, &root); err != nil {
		return Progress{}
	}
	campaigns := digArray(root, "data", "currentUser", "inventory", "dropCampaignsInProgress")
	return bestProgressFromCampaigns(campaigns, targetGame)
}

// FindActiveCampaignID returns the first ACTIVE/UPCOMING campaign matching targetGame.
func FindActiveCampaignID(campaignsResp json.RawMessage, targetGame string) string {
	var root map[string]any
	if err := json.Unmarshal(campaignsResp, &root); err != nil {
		return ""
	}
	list := digArray(root, "data", "currentUser", "dropCampaigns")
	for _, c := range list {
		cm, ok := c.(map[string]any)
		if !ok {
			continue
		}
		status, _ := cm["status"].(string)
		if status != "ACTIVE" && status != "UPCOMING" {
			continue
		}
		if campaignMatchesTarget(cm, targetGame) {
			if id, _ := cm["id"].(string); id != "" {
				return id
			}
		}
	}
	return ""
}

// ParseCampaignDetailsProgress reads timeBasedDrops from DropCampaignDetails.
func ParseCampaignDetailsProgress(det json.RawMessage, targetGame string) Progress {
	var root map[string]any
	if err := json.Unmarshal(det, &root); err != nil {
		return Progress{}
	}
	camp, _ := digMap(root, "data", "user", "dropCampaign")
	if camp == nil {
		return Progress{}
	}
	if !campaignMatchesTarget(camp, targetGame) {
		return Progress{}
	}
	drops := asArray(camp["timeBasedDrops"])
	return bestProgressFromDrops(drops)
}

func campaignMatchesTarget(c map[string]any, targetGame string) bool {
	if targetGame == "" {
		return true
	}
	game, _ := c["game"].(map[string]any)
	if game != nil {
		if name, _ := game["name"].(string); strings.EqualFold(name, targetGame) {
			return true
		}
		if dn, _ := game["displayName"].(string); strings.EqualFold(dn, targetGame) {
			return true
		}
	}
	cname, _ := c["name"].(string)
	return strings.Contains(strings.ToLower(cname), strings.ToLower(targetGame))
}

func bestProgressFromCampaigns(campaigns []any, targetGame string) Progress {
	best := Progress{}
	for _, c := range campaigns {
		cm, ok := c.(map[string]any)
		if !ok || !campaignMatchesTarget(cm, targetGame) {
			continue
		}
		p := bestProgressFromDrops(asArray(cm["timeBasedDrops"]))
		if p.Percent >= best.Percent {
			best = p
		}
	}
	return best
}

func bestProgressFromDrops(drops []any) Progress {
	var firstUnclaimed *Progress
	for idx, raw := range drops {
		d, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		self, _ := d["self"].(map[string]any)
		if self != nil {
			if claimed, _ := self["isClaimed"].(bool); claimed {
				continue
			}
		}
		cur := asFloat(self, "currentMinutesWatched")
		req := asFloat(d, "requiredMinutesWatched")
		if req == 0 {
			req = 1
		}
		pct := (cur / req) * 100
		if pct > 100 {
			pct = 100
		}
		label := dropLabel(d, idx)
		p := Progress{Percent: pct, DropName: label}
		if firstUnclaimed == nil {
			cp := p
			firstUnclaimed = &cp
		}
		if req > 0 && cur < req {
			return p
		}
	}
	if firstUnclaimed != nil {
		return *firstUnclaimed
	}
	return Progress{}
}

func dropLabel(d map[string]any, idx int) string {
	if name, _ := d["name"].(string); name != "" {
		return name
	}
	if edges, ok := d["benefitEdges"].([]any); ok && len(edges) > 0 {
		if e, ok := edges[0].(map[string]any); ok {
			if b, ok := e["benefit"].(map[string]any); ok {
				if name, _ := b["name"].(string); name != "" {
					return name
				}
			}
		}
	}
	if id, _ := d["id"].(string); id != "" {
		return id
	}
	return "drop_" + itoa(idx+1)
}

func digArray(root map[string]any, path ...string) []any {
	m := root
	for i, key := range path {
		if i == len(path)-1 {
			return asArray(m[key])
		}
		next, _ := m[key].(map[string]any)
		if next == nil {
			return nil
		}
		m = next
	}
	return nil
}

func digMap(root map[string]any, path ...string) (map[string]any, bool) {
	m := root
	for _, key := range path {
		next, ok := m[key].(map[string]any)
		if !ok || next == nil {
			return nil, false
		}
		m = next
	}
	return m, true
}

func asArray(v any) []any {
	a, _ := v.([]any)
	return a
}

func asFloat(m map[string]any, key string) float64 {
	if m == nil {
		return 0
	}
	switch n := m[key].(type) {
	case float64:
		return n
	case json.Number:
		f, _ := n.Float64()
		return f
	default:
		return 0
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
