package tests

import (
	"testing"

	"github.com/rm-hull/gps-routes-api/cmds"
)

func TestImportData(t *testing.T) {
	cmds.ImportData("../data/backup", 5)
}
