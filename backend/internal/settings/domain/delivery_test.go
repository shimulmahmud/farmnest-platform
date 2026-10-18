package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFee_ByAreaAndMethod(t *testing.T) {
	d := DefaultDelivery()

	m, fee, err := d.Fee("", "", 500)
	require.NoError(t, err)
	assert.Equal(t, "standard", m, "empty method = first enabled")
	assert.Equal(t, 60, fee, "empty area = inside Dhaka")

	_, fee, _ = d.Fee("standard", "outside_dhaka", 500)
	assert.Equal(t, 120, fee)
	_, fee, _ = d.Fee("express", "inside_dhaka", 500)
	assert.Equal(t, 120, fee)
	_, fee, _ = d.Fee("express", "outside_dhaka", 500)
	assert.Equal(t, 200, fee)
}

func TestFee_FreeStandardAtThreshold_ButNotExpress(t *testing.T) {
	d := DefaultDelivery()
	_, fee, _ := d.Fee("standard", "outside_dhaka", 2000)
	assert.Equal(t, 0, fee)
	_, fee, _ = d.Fee("express", "inside_dhaka", 9000)
	assert.Equal(t, 120, fee)

	d.FreeShippingThreshold = 0
	_, fee, _ = d.Fee("standard", "inside_dhaka", 999999)
	assert.Equal(t, 60, fee, "threshold 0 = never free")
}

func TestFee_RejectsUnknownAndDisabled(t *testing.T) {
	d := DefaultDelivery()
	_, _, err := d.Fee("teleport", "", 100)
	require.Error(t, err)
	_, _, err = d.Fee("standard", "mars", 100)
	require.Error(t, err)

	d.Methods[1].Enabled = false
	_, _, err = d.Fee("express", "", 100)
	require.Error(t, err, "a switched-off method cannot be chosen")
	assert.Len(t, d.EnabledMethods(), 1)
}

func TestValidate(t *testing.T) {
	d := DefaultDelivery()
	require.NoError(t, d.Validate())

	bad := DefaultDelivery()
	bad.Methods[0].FeeInsideDhaka = -1
	require.Error(t, bad.Validate(), "negative fee")

	bad = DefaultDelivery()
	bad.Methods[0].FeeOutsideDhaka = MaxFee + 1
	require.Error(t, bad.Validate(), "typo-sized fee")

	bad = DefaultDelivery()
	bad.Methods[0].Enabled, bad.Methods[1].Enabled = false, false
	require.Error(t, bad.Validate(), "everything switched off")

	bad = DefaultDelivery()
	bad.Methods[0].ID = "drone"
	require.Error(t, bad.Validate(), "unknown method id")

	bad = DefaultDelivery()
	bad.Methods[1].ID = "standard"
	require.Error(t, bad.Validate(), "duplicate id")

	bad = DefaultDelivery()
	bad.Methods[0].Label = "  "
	require.Error(t, bad.Validate(), "name required")

	bad = DefaultDelivery()
	bad.FreeShippingThreshold = -5
	require.Error(t, bad.Validate())
}
