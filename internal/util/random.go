package util

import (
	"math/rand"
	"strings"
	"time"
)

var random *rand.Rand

const alphabet = "abcdefghijklmnopqrstuvxz"

func init() {
	random = rand.New(rand.NewSource(time.Now().UnixNano()))
}

// Random int between min and max
func RandomInt(min, max int64) int64 {
	return min + random.Int63n(max-min+1)
}

// Random string with len == n
func RandomString(n int) string {
	var sb strings.Builder
	k := len(alphabet)

	for i := 0; i < n; i++ {
		c := alphabet[random.Intn(k)]
		sb.WriteByte(c)
	}

	return sb.String()
}

func RandomPassword() string {
	return RandomString(28)
}

func RandomEmail() string {
	return RandomString(8) + "@mail.com"
}

func RandomFullName() string {
	return RandomString(8) + " " + RandomString(8)
}
