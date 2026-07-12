package gql

// Inventory builds the Inventory persisted query body.
func Inventory() map[string]any {
	return persisted("Inventory", InventoryHash, map[string]any{
		"fetchRewardCampaigns": false,
	})
}

// Campaigns builds ViewerDropsDashboard.
func Campaigns() map[string]any {
	return persisted("ViewerDropsDashboard", CampaignsHash, map[string]any{
		"fetchRewardCampaigns": false,
	})
}

// ClaimDrop builds DropsPage_ClaimDropRewards.
func ClaimDrop(dropInstanceID string) map[string]any {
	return persisted("DropsPage_ClaimDropRewards", ClaimHash, map[string]any{
		"input": map[string]any{"dropInstanceID": dropInstanceID},
	})
}

// GameDirectory builds DirectoryPage_Game.
func GameDirectory(slug string, limit int, dropsOnly bool) map[string]any {
	filters := []string{}
	if dropsOnly {
		filters = []string{"DROPS_ENABLED"}
	}
	return persisted("DirectoryPage_Game", GameDirectoryHash, map[string]any{
		"limit":              limit,
		"slug":               slug,
		"imageWidth":         50,
		"includeCostreaming": false,
		"options": map[string]any{
			"broadcasterLanguages":   []any{},
			"freeformTags":           nil,
			"includeRestricted":      []string{"SUB_ONLY_LIVE"},
			"recommendationsContext": map[string]any{"platform": "web"},
			"sort":                   "VIEWER_COUNT",
			"systemFilters":          filters,
			"tags":                   []any{},
			"requestID":              "JIRA-VXP-2397",
		},
		"sortTypeIsRecency": false,
	})
}

// StreamInfo builds VideoPlayerStreamInfoOverlayChannel.
func StreamInfo(channelLogin string) map[string]any {
	return persisted("VideoPlayerStreamInfoOverlayChannel", StreamInfoHash, map[string]any{
		"channel": channelLogin,
	})
}

// CurrentDrop builds DropCurrentSessionContext.
func CurrentDrop(channelID string) map[string]any {
	return persisted("DropCurrentSessionContext", CurrentDropHash, map[string]any{
		"channelID":    channelID,
		"channelLogin": "",
	})
}

// CampaignDetails builds DropCampaignDetails.
func CampaignDetails(channelLogin, campaignID string) map[string]any {
	return persisted("DropCampaignDetails", CampaignDetailsHash, map[string]any{
		"channelLogin": channelLogin,
		"dropID":       campaignID,
	})
}

// SendSpadeEvents builds the inline SendEvents mutation (GZIP_B64 payload).
func SendSpadeEvents(payloadGzipB64 string) map[string]any {
	return map[string]any{
		"query": `mutation SendEvents($input: SendSpadeEventsInput!) { sendSpadeEvents(input: $input) { statusCode } }`,
		"variables": map[string]any{
			"input": map[string]any{
				"data":       payloadGzipB64,
				"repository": "twilight",
				"encoding":   "GZIP_B64",
			},
		},
	}
}

func persisted(operation, hash string, variables map[string]any) map[string]any {
	return map[string]any{
		"operationName": operation,
		"extensions": map[string]any{
			"persistedQuery": map[string]any{
				"version":    1,
				"sha256Hash": hash,
			},
		},
		"variables": variables,
	}
}
