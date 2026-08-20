package docket

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
)

// UID is uuid-v4 + 16-byte salt. Face id stays TASK-0001 / MS-0001.
var uidPat = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}\+[0-9a-f]{32}$`)

func NewUID() (string, error) {
	var guid [16]byte
	if _, err := rand.Read(guid[:]); err != nil {
		return "", err
	}
	guid[6] = (guid[6] & 0x0f) | 0x40
	guid[8] = (guid[8] & 0x3f) | 0x80
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x+%s", guid[0:4], guid[4:6], guid[6:8], guid[8:10], guid[10:], hex.EncodeToString(salt)), nil
}

func CheckUID(s string) error {
	if !uidPat.MatchString(strings.ToLower(strings.TrimSpace(s))) {
		return fmt.Errorf("uid required (uuid-v4+32-hex-salt)")
	}
	return nil
}

func fillUID(uid *string) error {
	if strings.TrimSpace(*uid) == "" {
		v, err := NewUID()
		if err != nil {
			return err
		}
		*uid = v
		return nil
	}
	return CheckUID(*uid)
}
