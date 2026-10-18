package transport

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSamePhone(t *testing.T) {
	assert.True(t, samePhone("+8801712345678", "01712345678"))
	assert.True(t, samePhone("017-1234 5678", "8801712345678"))
	assert.False(t, samePhone("01712345678", "01812345678"))
	assert.False(t, samePhone("123", "123"), "too short to compare")
}

func TestNormalizeArea(t *testing.T) {
	assert.Equal(t, "inside_dhaka", normalizeArea(""))
	assert.Equal(t, "inside_dhaka", normalizeArea("   "))
	assert.Equal(t, "outside_dhaka", normalizeArea(" Outside_Dhaka "))
}

func TestDeliveryNow_FallsBackToDefaultsWithoutSettings(t *testing.T) {
	h := &Handler{}
	d := h.deliveryNow(context.Background())
	require.NotEmpty(t, d.Methods)
	_, fee, err := d.Fee("standard", "inside_dhaka", 100)
	require.NoError(t, err)
	assert.Equal(t, 60, fee)
}
