package utils

import (
	"fmt"

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
	strs := []attr.Value{}
	for _, value := range slice {
		strs = append(strs, value)
	}
	return types.SetValueMust(types.StringType, strs)
}

func MapToTypesMap(m map[string]interface{}) types.Map {
	tm := map[string]attr.Value{}
	for k, v := range m {
		switch val := v.(type) {
		case bool:
			tm[k] = types.BoolValue(val)
		case int64:
			tm[k] = types.Int64Value(val)
		case float64:
			tm[k] = types.Float64Value(val)
		case string:
			tm[k] = types.StringValue(val)
		}
	}
	return types.MapValueMust(types.StringType, tm)
}

func ConvertInterfaceToMap(data interface{}) (map[string]interface{}, error) {

	m, ok := data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("input data is not a map")
	}
	converted := make(map[string]interface{})

	for k, v := range m {
		if subMap, ok := v.(map[string]interface{}); ok {
			subConverted, err := ConvertInterfaceToMap(subMap)
			if err != nil {
				return nil, err
			}
			converted[k] = subConverted
		} else {
			converted[k] = v
		}
	}

	return converted, nil
}
