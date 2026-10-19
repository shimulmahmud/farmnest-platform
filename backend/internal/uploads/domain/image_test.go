package domain

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func canvas(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			img.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 120, A: 255})
		}
	}
	return img
}

func pngBytes(t *testing.T, w, h int) []byte {
	var b bytes.Buffer
	require.NoError(t, png.Encode(&b, canvas(w, h)))
	return b.Bytes()
}

func TestCheck_AcceptsRealPictures_AndUsesRealType(t *testing.T) {
	v := DefaultValidator()

	info, err := v.Check(pngBytes(t, 20, 10))
	require.NoError(t, err)
	assert.Equal(t, ".png", info.Ext)
	assert.Equal(t, "image/png", info.ContentType)
	assert.Equal(t, 20, info.Width)
	assert.Equal(t, 10, info.Height)

	var j bytes.Buffer
	require.NoError(t, jpeg.Encode(&j, canvas(16, 16), nil))
	info, err = v.Check(j.Bytes())
	require.NoError(t, err)
	assert.Equal(t, ".jpg", info.Ext)

	var g bytes.Buffer
	require.NoError(t, gif.Encode(&g, canvas(8, 8), nil))
	info, err = v.Check(g.Bytes())
	require.NoError(t, err)
	assert.Equal(t, ".gif", info.Ext)
}

func TestCheck_RejectsNonPictures(t *testing.T) {
	v := DefaultValidator()
	cases := map[string][]byte{
		"svg (can carry scripts)":    []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`),
		"html renamed to a picture":  []byte(`<html><body><script>alert(1)</script></body></html>`),
		"plain text":                 []byte("hello world"),
		"executable":                 append([]byte("MZ"), make([]byte, 600)...),
		"empty":                      {},
		"pdf":                        []byte("%PDF-1.4 something"),
	}
	for name, data := range cases {
		_, err := v.Check(data)
		assert.Error(t, err, name)
	}
}

func TestCheck_RejectsTruncatedPicture(t *testing.T) {
	data := pngBytes(t, 30, 30)
	_, err := DefaultValidator().Check(data[:20]) // looks like PNG at first, but the header is cut
	require.Error(t, err)
}

func TestCheck_Limits(t *testing.T) {
	small := Validator{MaxBytes: 100, MaxPixels: 1_000_000}
	_, err := small.Check(pngBytes(t, 50, 50))
	require.Error(t, err, "file larger than the byte limit")

	pixels := Validator{MaxBytes: 1 << 20, MaxPixels: 5000}
	_, err = pixels.Check(pngBytes(t, 100, 100))
	require.Error(t, err, "more pixels than allowed")
	_, err = pixels.Check(pngBytes(t, 50, 50))
	require.NoError(t, err)
}
