package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

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

func ExceedsValidRange(resp any, vl any) {
	summary := "Error: value exceeds the valid range"
	detail := "n32 exceeds int32 limits"
	if vl != nil {
		detail = fmt.Sprintf("n32 %v exceeds int32 limits", vl)
	}
	switch v := resp.(type) {
	case *resource.DeleteResponse:
		v.Diagnostics.AddError(summary, detail)
		return
	case *resource.CreateResponse:
		v.Diagnostics.AddError(summary, detail)
		return
	case *resource.SchemaResponse:
		v.Diagnostics.AddError(summary, detail)
		return
	case *resource.UpdateResponse:
		v.Diagnostics.AddError(summary, detail)
		return
	case *resource.ConfigureResponse:
		v.Diagnostics.AddError(summary, detail)
		return
	case *resource.ModifyPlanResponse:
		v.Diagnostics.AddError(summary, detail)
		return
	case *resource.ImportStateResponse:
		v.Diagnostics.AddError(summary, detail)
		return
	case *resource.UpgradeStateResponse:
		v.Diagnostics.AddError(summary, detail)
		return
	}
}

func SleepAfter429(response *http.Response) error {
	timeToSleep := response.Header.Get("retry-after")
	num, err := strconv.Atoi(timeToSleep)
	if err != nil {
		return err
	}
	time.Sleep((time.Duration(num) + 1) * time.Second)
	return nil
}

// RetryOn429 retries an API call if the response status is 429.
func RetryOn429[T any](apiCall func() (T, *http.Response, error), maxRetries int) (T, *http.Response, error) {
	var result T
	var response *http.Response
	var err error

	for i := 0; i < maxRetries; i++ {
		// Call the API function
		result, response, err = apiCall()

		// If response is not nil, ensure the body is closed before retrying
		if response != nil {
			defer response.Body.Close()
		}

		// If no error and not a 429, return successfully
		if err == nil && response.StatusCode != http.StatusTooManyRequests {
			return result, response, nil
		}

		// If error is not 429, return immediately
		if response.StatusCode != http.StatusTooManyRequests {
			return result, response, err
		}

		// Sleep before retrying
		if sleepErr := SleepAfter429(response); sleepErr != nil {
			return result, response, sleepErr
		}
	}

	return result, response, errors.New("max retries exceeded for API request")
}
