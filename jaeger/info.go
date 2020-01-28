package jaeger

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"

	"github.com/kiali/kiali/config"
)

func getJaegerInfo(client http.Client, endpoint *url.URL) (*JaegerInfo, int, error) {
	jaegerConfig := config.Get().ExternalServices.Tracing
	integration := true
	error := ""
	if !jaegerConfig.Enabled {
		return nil, http.StatusNoContent, nil
	}

	u, err := url.Parse(jaegerConfig.InClusterURL)
	if err != nil {
		integration = false
		error = "Error parsing in cluster url for Jaeger : " + err.Error()
	} else {
		u.Path = path.Join(u.Path, "/api/services")
		resp, code, err := makeRequest(client, u.String(), nil)
		if err != nil || code != 200 {
			integration = false
			error = "Error with internal connection with Jaeger"
			if err != nil {
				error += ": " + err.Error()
			}
		} else {
			var response JaegerResponse
			if errMarshal := json.Unmarshal([]byte(resp), &response); errMarshal != nil {
				error = fmt.Sprintf("Error unmarshalling Jaeger response, check the endpoint configuration.\nErr: %v\nResponse: %s", errMarshal, string(resp))
				integration = false
			}
		}
	}
	jaegerIntegration = integration

	info := &JaegerInfo{
		URL:                jaegerConfig.URL,
		NamespaceSelector:  jaegerConfig.NamespaceSelector,
		Integration:        integration,
		IntegrationMessage: error,
	}

	return info, http.StatusOK, nil
}
