package infrastructure

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalStore_SavesFileAndGivesURL(t *testing.T) {
	dir := t.TempDir()
	s, err := NewLocalStore(filepath.Join(dir, "up"), "/uploads/")
	require.NoError(t, err)

	require.NoError(t, s.Save(context.Background(), "a.png", []byte("data")))
	got, err := os.ReadFile(filepath.Join(dir, "up", "a.png"))
	require.NoError(t, err)
	assert.Equal(t, "data", string(got))
	assert.Equal(t, "/uploads/a.png", s.URL("a.png"))
}

func TestLocalStore_RefusesPathTricksAndOverwrites(t *testing.T) {
	s, err := NewLocalStore(t.TempDir(), "/uploads")
	require.NoError(t, err)
	ctx := context.Background()

	for _, bad := range []string{"../evil.png", "a/b.png", `a\b.png`, "", "/etc/passwd"} {
		assert.Error(t, s.Save(ctx, bad, []byte("x")), bad)
	}

	require.NoError(t, s.Save(ctx, "same.png", []byte("one")))
	assert.Error(t, s.Save(ctx, "same.png", []byte("two")), "an existing file is never overwritten")
}
