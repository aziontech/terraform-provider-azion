package utils

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func SliceStringTypeToList(slice []types.String) types.List {
	if len(slice) == 0 {
		return types.ListNull(types.StringType)
	}
	strs := []attr.Value{}
	for _, value := range slice {
		strs = append(strs, value)
	}
	return types.ListValueMust(types.StringType, strs)
}

func SliceStringTypeToSet(slice []types.String) types.Set {
	if len(slice) == 0 {
		return types.SetNull(types.StringType)
	}
	strs := []attr.Value{}
	for _, value := range slice {
		strs = append(strs, value)
	}
	return types.SetValueMust(types.StringType, strs)
}
