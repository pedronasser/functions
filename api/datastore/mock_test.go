package datastore

import (
	"testing"

	"github.com/pedronasser/functions/api/datastore/internal/datastoretest"
)

func TestDatastore(t *testing.T) {
	datastoretest.Test(t, NewMock())
}