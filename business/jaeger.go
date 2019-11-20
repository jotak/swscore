package business

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"time"

	jaegerModels "github.com/jaegertracing/jaeger/model/json"

	"github.com/kiali/kiali/config"
	"github.com/kiali/kiali/log"
	"github.com/kiali/kiali/util/httputil"
)

type JaegerService struct {
	auth          config.Auth
	businessLayer *Layer
}

func (in *JaegerService) makeRequest(endpoint string, body io.Reader) (response []byte, status int, err error) {
	response = nil
	status = 0
	client, err := in.getClient()
	if err != nil {
		return
	}
	req, err := http.NewRequest(http.MethodGet, endpoint, body)
	if err != nil {
		return
	}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	response, err = ioutil.ReadAll(resp.Body)
	status = resp.StatusCode
	return
}

func (in *JaegerService) getClient() (client http.Client, err error) {
	timeout := time.Duration(5000 * time.Millisecond)
	client = http.Client{}
	transport, err := httputil.AuthTransport(&in.auth, &http.Transport{})
	if err != nil {
		return
	}
	client = http.Client{Transport: transport, Timeout: timeout}
	return
}

func getErrorTracesFromJaeger(namespace string, service string, requestToken string) (errorTraces int, err error) {
	errorTraces = 0
	err = nil
	if !config.Get().ExternalServices.Tracing.Enabled {
		return -1, errors.New("jaeger is not available")
	}
	if config.Get().ExternalServices.Tracing.Enabled {
		// Be sure to copy config.Auth and not modify the existing
		auth := config.Get().ExternalServices.Tracing.Auth
		if auth.UseKialiToken {
			auth.Token = requestToken
		}

		u, errParse := url.Parse(config.Get().ExternalServices.Tracing.InClusterURL)
		if !config.Get().InCluster {
			u, errParse = url.Parse(config.Get().ExternalServices.Tracing.URL)
		}
		u.Path = path.Join(u.Path, "/api/traces")

		if errParse != nil {
			log.Errorf("Error parse Jaeger URL fetching Error Traces: %s", err)
			return -1, errParse
		} else {
			q := u.Query()
			q.Set("lookback", "1h")
			queryService := fmt.Sprintf("%s.%s", service, namespace)
			if !config.Get().ExternalServices.Tracing.NamespaceSelector {
				queryService = service
			}
			q.Set("service", queryService)
			t := time.Now().UnixNano() / 1000
			q.Set("start", fmt.Sprintf("%d", t-60*60*1000*1000))
			q.Set("end", fmt.Sprintf("%d", t))
			q.Set("tags", "{\"error\":\"true\"}")

			u.RawQuery = q.Encode()

			body, code, reqError := httputil.HttpGet(u.String(), &auth, time.Second)
			if reqError != nil {
				log.Errorf("Error fetching Jaeger Error Traces (%d): %s", code, reqError)
				return -1, reqError
			} else {
				if code != http.StatusOK {
					return -1, fmt.Errorf("error from Jaeger (%d)", code)
				}
				var traces struct {
					Data []*jaegerModels.Trace `json:"data"`
				}

				if errMarshal := json.Unmarshal([]byte(body), &traces); errMarshal != nil {
					log.Errorf("Error Unmarshal Jaeger Response fetching Error Traces: %s", errMarshal)
					err = errMarshal
					return -1, err
				}
				errorTraces = len(traces.Data)
			}
		}
	}
	return errorTraces, err
}

func getJaegerEndpoint() (u *url.URL, err error) {
	u, err = url.Parse(config.Get().ExternalServices.Tracing.InClusterURL)
	if !config.Get().InCluster {
		u, err = url.Parse(config.Get().ExternalServices.Tracing.URL)
	}
	if err != nil {
		log.Errorf("Error parse Jaeger URL: %s", err)
	}
	return
}

func (in *JaegerService) GetJaegerServices() (services []string, code int, err error) {
	code = 0
	services = []string{}
	u, err := getJaegerEndpoint()
	if err != nil {
		return
	}
	u.Path = path.Join(u.Path, "/api/services")
	resp, code, err := in.makeRequest(u.String(), nil)

	if err != nil {
		log.Errorf("Error request Jaeger URL : %s", err)
		return
	}
	var jaegerResponse struct {
		Data []string `json:"data"`
	}
	if err = json.Unmarshal([]byte(resp), &jaegerResponse); err != nil {
		log.Errorf("Error Unmarshal Jaeger Response fetching Services: %s", err)
		return
	}
	services = jaegerResponse.Data
	code = 200
	return
}

func (in *JaegerService) GetJaegerTraces(namespace string, service string, rawQuery string) (traces []*jaegerModels.Trace, code int, err error) {
	code = 0
	u, err := getJaegerEndpoint()
	if err != nil {
		return
	}

	if config.Get().ExternalServices.Tracing.NamespaceSelector {
		service = service + "." + namespace
	}
	u.Path = path.Join(u.Path, "/api/traces")
	q, _ := url.ParseQuery(rawQuery)
	q.Add("service", service)
	u.RawQuery = q.Encode()
	resp, code, err := in.makeRequest(u.String(), nil)
	if err != nil {
		log.Errorf("Error request Jaeger URL : %s", err)
		return
	}
	var jaegerResponse struct {
		Data []*jaegerModels.Trace `json:"data"`
	}
	if err = json.Unmarshal([]byte(resp), &jaegerResponse); err != nil {
		log.Errorf("Error Unmarshal Jaeger Response fetching Services: %s", err)
		return
	}
	traces = jaegerResponse.Data
	code = 200
	return
}

func (in *JaegerService) GetJaegerTraceDetail(request *http.Request) (trace []*jaegerModels.Trace, code int, err error) {
	code = 0
	u, err := url.Parse(config.Get().ExternalServices.Tracing.InClusterURL)
	if err != nil {
		log.Errorf("Error parse Jaeger URL fetching Services: %s", err)
		return
	}
	params := mux.Vars(request)
	traceID := params["traceID"]
	u.Path = path.Join(u.Path, "/api/traces/"+traceID)

	resp, code, err := in.makeRequest(u.String(), nil)
	if err != nil {
		log.Errorf("Error request Jaeger URL : %s", err)
		return
	}
	var jaegerResponse struct {
		Data []*jaegerModels.Trace `json:"data"`
	}
	if err = json.Unmarshal([]byte(resp), &jaegerResponse); err != nil {
		log.Errorf("Error Unmarshal Jaeger Response fetching Services: %s", err)
		return
	}
	trace = jaegerResponse.Data
	code = 200
	return

}
