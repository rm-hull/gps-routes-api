package tests

import (
	"testing"

	"github.com/map-services/gps-routes-api/cmds"
)

func TestDbPing(t *testing.T) {
	cmds.PingDatabase()
}
