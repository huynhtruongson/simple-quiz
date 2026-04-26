package utils

import "time"

func GenerateID() string {
	const refChars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"

	ts := time.Now().UTC().UnixNano()
	b := make([]byte, 8)
	for i := 0; i < 8; i++ {
		b[i] = refChars[ts%36]
		ts /= 36
	}
	return string(b)
}
