package mongodb

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew_InvalidURI(t *testing.T) {
	_, err := New(context.Background(), "mongodb://invalid:99999", "testdb", 10, 2, 1)
	assert.Error(t, err)
}
