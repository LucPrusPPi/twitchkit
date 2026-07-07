package drops

import (
	"encoding/json"
	"testing"
)

func TestListClaimable(t *testing.T) {
	inv := json.RawMessage(`{
	  "data": {
	    "currentUser": {
	      "inventory": {
	        "dropCampaignsInProgress": [
	          {
	            "id": "camp1",
	            "name": "Demo Campaign",
	            "game": {"name": "Just Chatting", "displayName": "Just Chatting"},
	            "timeBasedDrops": [
	              {
	                "id": "drop1",
	                "name": "Badge",
	                "requiredMinutesWatched": 60,
	                "self": {
	                  "isClaimed": false,
	                  "currentMinutesWatched": 60,
	                  "dropInstanceID": "inst-1"
	                }
	              },
	              {
	                "id": "drop2",
	                "name": "Emote",
	                "requiredMinutesWatched": 120,
	                "self": {
	                  "isClaimed": false,
	                  "currentMinutesWatched": 10,
	                  "dropInstanceID": "inst-2"
	                }
	              }
	            ]
	          }
	        ]
	      }
	    }
	  }
	}`)

	list := ListClaimable(inv)
	if len(list) != 1 {
		t.Fatalf("want 1 claimable, got %d", len(list))
	}
	if list[0].DropInstanceID != "inst-1" {
		t.Fatalf("instance: %q", list[0].DropInstanceID)
	}
	if list[0].DropName != "Badge" {
		t.Fatalf("name: %q", list[0].DropName)
	}
}
