package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func SliceStringTypeToList(slice []types.String) types.List {
	if len(slice) == 0 {
		return types.ListValueMust(types.StringType, nil)
	}
	strs := []attr.Value{}
	for _, value := range slice {
		strs = append(strs, value)
	}
	return types.ListValueMust(types.StringType, strs)
}

func SliceIntTypeToList(slice []types.Int64) types.List {
	if len(slice) == 0 {
		return types.ListValueMust(types.Int64Type, nil)
	}
	integers := []attr.Value{}
	for _, value := range slice {
		integers = append(integers, value)
	}
	return types.ListValueMust(types.Int64Type, integers)
}

func SliceStringTypeToSet(slice []types.String) types.Set {
	if len(slice) == 0 {
		return types.SetNull(types.StringType)
	}
	strings := []attr.Value{}
	for _, value := range slice {
		strings = append(strings, value)
	}
	return types.SetValueMust(types.StringType, strings)
}

func SliceStringTypeToSetOrNull(slice []types.String) types.Set {
	if len(slice) == 0 {
		return types.SetValueMust(types.StringType, nil)
	}
	strings := []attr.Value{}
	for _, value := range slice {
		strings = append(strings, value)
	}
	return types.SetValueMust(types.StringType, strings)
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

func ConvertInterfaceToFloat64List(listInt interface{}) []types.Float64 {
	iListInt := listInt.([]interface{})
	var integers []types.Float64
	for _, v := range iListInt {
		if _, ok := v.(float64); !ok {
			return nil
		}
		integers = append(integers, types.Float64Value(v.(float64)))
	}
	return integers
}

func ConvertFloat64ToInterface(sliceInt []types.Float64) (interface{}, error) {
	var integers []float64
	for _, v := range sliceInt {
		integers = append(integers, v.ValueFloat64())
	}
	return integers, nil
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

// CheckInt64toInt32Security check parse direct int64 to int32.
func CheckInt64toInt32Security(n int64) (int32, error) {
	if n < math.MinInt32 || n > math.MaxInt32 {
		return 0, errors.New("Overflow")
	}
	return int32(n), nil
}
