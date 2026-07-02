package provider

import (
	"io"
	"net/http"
)

func readAPIErrorBody(response *http.Response) string {
	if response == nil || response.Body == nil {
		return ""
	}

	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return err.Error()
	}

	return string(bodyBytes)
}
