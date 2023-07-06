package utils

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
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

func SliceIntInterfaceTypeToList(sliceInt interface{}) (types.List, error) {
	var integers []attr.Value
	httpPortSlice, ok := sliceInt.([]interface{})
	if !ok {
		return types.ListNull(types.Float64Type), fmt.Errorf("slice Int is not a slice")
	}
	for _, v := range httpPortSlice {
		if _, ok := v.(float64); !ok {
			return types.List{}, nil
		}
		integers = append(integers, types.Float64Value(v.(float64)))
	}

	return types.ListValueMust(types.Float64Type, integers), nil
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

func ConvertStringToInterface(jsonArgs string) (interface{}, error) {
	var data map[string]interface{}
	err := json.Unmarshal([]byte(jsonArgs), &data)
	if err != nil {
		fmt.Println("Error:", err)
		return nil, err
	}
	return data, err
}

func ConvertInterfaceToString(jsonArgs interface{}) (string, error) {
	jsonArgsStr, err := json.Marshal(jsonArgs)
	if err != nil {
		fmt.Println("Error:", err)
		return "", err
	}

	return string(jsonArgsStr), err
}

func AtoiNoError(strToConv string, resp *resource.ReadResponse) int32 {
	intReturn, err := strconv.ParseInt(strToConv, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Value Conversion error ",
			"Could not convert String to Int",
		)
		return 0
	}
	if intReturn > math.MaxInt32 || intReturn < math.MinInt32 {
		resp.Diagnostics.AddError(
			"Value Overflow error",
			"Converted value exceeds the range of int32",
		)
		return 0
	}
	return int32(intReturn)
}
