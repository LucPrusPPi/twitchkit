package client

import "crypto/rand"

func cryptoRandRead(p []byte) (int, error) {
	return rand.Read(p)
}

func randomHex(n int) string {
	const hexdigits = "0123456789abcdef"
	rb := make([]byte, n)
	if _, err := rand.Read(rb); err != nil {
		// extremely unlikely; fall back to zeros-derived pattern
		for i := range rb {
			rb[i] = byte(i)
		}
	}
	b := make([]byte, n)
	for i := range b {
		b[i] = hexdigits[rb[i]%16]
	}
	return string(b)
}
