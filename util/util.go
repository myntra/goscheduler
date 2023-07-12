package util

import (
	"github.com/gocql/gocql"
)

// IsZeroUUID checks and returns true if uuid is zero uuid (i.e 00000000-0000-0000-0000-000000000000 ) else false
func IsZeroUUID(uuid gocql.UUID) bool{
	for x := 0; x < 16; x++ {
		if uuid[x] != 0 {
			return false
		}
	}
	return true
}
