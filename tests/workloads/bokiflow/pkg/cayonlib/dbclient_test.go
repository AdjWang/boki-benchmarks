package cayonlib

import (
	"log"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
)

func TestReadData(t *testing.T) {
	ListTables()
	tablename := "MovieId"
	key := "National Lampoon's Loaded Weapon 1"
	item := LibRead(tablename, aws.JSONValue{"K": key}, []string{"V"})
	log.Printf("[INFO] Read data from DB item: %+v", item)
}
