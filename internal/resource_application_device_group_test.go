package provider

import "testing"

func TestApplicationDeviceGroupNestedObjectsAreBlocks(t *testing.T) {
	assertBlocks(t, NewApplicationDeviceGroupResource(), "device_group")
}
