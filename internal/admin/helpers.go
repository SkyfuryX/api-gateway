package admin

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
)

func cliBackCheck(s string) bool {
	lower := strings.ToLower(s)
	switch lower {
	case "back", "exit", "quit":
		return true
	default:
		return false
	}
}

func GenerateAdminAPIKey() string {
	bytes := make([]byte, 24)
	rand.Read(bytes)
	return fmt.Sprintf("gw_admin_%s", hex.EncodeToString(bytes))
}
