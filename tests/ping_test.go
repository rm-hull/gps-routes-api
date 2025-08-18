package tests

import (
	"testing"

	"github.com/rm-hull/gps-routes-api/cmds"
)

func TestDbPing(t *testing.T) {
	cmds.PingDatabase()
}
