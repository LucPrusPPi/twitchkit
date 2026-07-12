package client

// PickTopStream returns the stream with the highest viewer count.
// If skipLogin is non-empty, that channel is preferred to be avoided when
// another stream exists (useful when rotating after no progress).
func PickTopStream(streams []StreamInfo, skipLogin string) *StreamInfo {
	if len(streams) == 0 {
		return nil
	}
	pool := streams
	if skipLogin != "" {
		var filtered []StreamInfo
		for _, s := range streams {
			if !equalFoldASCII(s.UserLogin, skipLogin) {
				filtered = append(filtered, s)
			}
		}
		if len(filtered) > 0 {
			pool = filtered
		}
	}
	best := pool[0]
	for _, s := range pool[1:] {
		if s.ViewerCount > best.ViewerCount {
			best = s
		}
	}
	return &best
}

func equalFoldASCII(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}
