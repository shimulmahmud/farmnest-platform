package application

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ecommerce/backend/internal/uploads/domain"
)

type memStore struct{ files map[string][]byte }

func (m *memStore) Save(_ context.Context, name string, data []byte) error {
	m.files[name] = data
	return nil
}
func (m *memStore) URL(name string) string { return "/uploads/" + name }

func tinyPNG(t *testing.T) []byte {
	var b bytes.Buffer
	require.NoError(t, png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 4, 4))))
	return b.Bytes()
}

func TestUploadImage_StoresUnderRandomNameWithRealExtension(t *testing.T) {
	store := &memStore{files: map[string][]byte{}}
	svc := NewService(domain.DefaultValidator(), store)

	res, err := svc.UploadImage(context.Background(), tinyPNG(t))
	require.NoError(t, err)
	assert.Regexp(t, regexp.MustCompile(`^/uploads/[0-9a-f-]{36}\.png$`), res.URL)
	assert.Equal(t, "image/png", res.ContentType)
	assert.Len(t, store.files, 1)

	res2, err := svc.UploadImage(context.Background(), tinyPNG(t))
	require.NoError(t, err)
	assert.NotEqual(t, res.URL, res2.URL, "every upload gets its own name")
}

func TestUploadImage_RejectsNonPicturesAndStoresNothing(t *testing.T) {
	store := &memStore{files: map[string][]byte{}}
	svc := NewService(domain.DefaultValidator(), store)

	_, err := svc.UploadImage(context.Background(), []byte("<svg><script>alert(1)</script></svg>"))
	require.Error(t, err)
	assert.Empty(t, store.files)
}
