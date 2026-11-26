package provider

import "testing"

func TestApplicationFunctionInstanceNestedObjectsAreBlocks(t *testing.T) {
	assertBlocks(t, NewApplicationFunctionInstanceResource(), "data")
}
