package prometheus

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/api"
	"github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"

	"github.com/kiali/kiali/config"
	"github.com/kiali/kiali/log"
	"github.com/kiali/kiali/prometheus/internalmetrics"
	"github.com/kiali/kiali/util"
)

// ClientInterface for mocks (only mocked function are necessary here)
type ClientInterface interface {
	GetServiceHealth(namespace, servicename string, ports []int32) (EnvoyServiceHealth, error)
	GetAllRequestRates(namespace, ratesInterval string, queryTime time.Time) (model.Vector, error)
	GetNamespaceServicesRequestRates(namespace, ratesInterval string, queryTime time.Time) (model.Vector, error)
	GetServiceRequestRates(namespace, service, ratesInterval string, queryTime time.Time) (model.Vector, error)
	GetAppRequestRates(namespace, app, ratesInterval string, queryTime time.Time) (model.Vector, model.Vector, error)
	GetWorkloadRequestRates(namespace, workload, ratesInterval string, queryTime time.Time) (model.Vector, model.Vector, error)
	GetSourceWorkloads(namespace string, namespaceCreationTime time.Time, servicename string) (map[string][]Workload, error)
	FetchRateRange(metricName, labels, rateFunc, rateInterval, grouping string, bounds v1.Range) *Metric
	FetchHistogramRange(metricName, labels, rateInterval, grouping string, bounds v1.Range, avg bool, quantiles []string) Histogram
}

// Client for Prometheus API.
// It hides the way we query Prometheus offering a layer with a high level defined API.
type Client struct {
	ClientInterface
	p8s api.Client
	api v1.API
}

// Workload describes a workload with contextual information
type Workload struct {
	Namespace string
	App       string
	Workload  string
	Version   string
}

// NewClient creates a new client to the Prometheus API.
// It returns an error on any problem.
func NewClient() (*Client, error) {
	if config.Get() == nil {
		return nil, errors.New("config.Get() must be not null")
	}
	p8s, err := api.NewClient(api.Config{Address: config.Get().ExternalServices.PrometheusServiceURL})
	if err != nil {
		return nil, err
	}
	client := Client{p8s: p8s, api: v1.NewAPI(p8s)}
	return &client, nil
}

// Inject allows for replacing the API with a mock For testing
func (in *Client) Inject(api v1.API) {
	in.api = api
}

// GetSourceWorkloads returns a map of list of source workloads for a given service
// identified by its namespace and service name.
// Returned map has a destination version as a key and a list of workloads as values.
// It returns an error on any problem.
func (in *Client) GetSourceWorkloads(namespace string, namespaceCreationTime time.Time, servicename string) (map[string][]Workload, error) {
	reporter := "source"
	if config.Get().IstioNamespace == namespace {
		reporter = "destination"
	}
	// The query needs a lower bound to make sure that no outdated data is fetched
	// So, a range is set and an "easy" function (delta) is applied to return an instant-vector,
	// since only labels are needed.
	queryTime := util.Clock.Now()
	queryInterval := queryTime.Sub(namespaceCreationTime)
	query := fmt.Sprintf("delta(istio_requests_total{reporter=\"%s\",destination_service_name=\"%s\",destination_service_namespace=\"%s\"}[%vs])",
		reporter, servicename, namespace, int(queryInterval.Seconds()))
	log.Debugf("GetSourceWorkloads query: %s", query)
	promtimer := internalmetrics.GetPrometheusProcessingTimePrometheusTimer("GetSourceWorkloads")
	result, err := in.api.Query(context.Background(), query, queryTime)
	if err != nil {
		return nil, err
	}
	promtimer.ObserveDuration() // notice we only collect metrics for successful prom queries
	routes := make(map[string][]Workload)
	switch result.Type() {
	case model.ValVector:
		matrix := result.(model.Vector)
		for _, sample := range matrix {
			metric := sample.Metric
			index := string(metric["destination_version"])
			source := Workload{
				Namespace: string(metric["source_workload_namespace"]),
				App:       string(metric["source_app"]),
				Workload:  string(metric["source_workload"]),
				Version:   string(metric["source_version"]),
			}
			if arr, ok := routes[index]; ok {
				found := false
				for _, s := range arr {
					if s.Workload == source.Workload {
						found = true
						break
					}
				}
				if !found {
					routes[index] = append(arr, source)
				}
			} else {
				routes[index] = []Workload{source}
			}
		}
	}
	return routes, nil
}

// GetMetrics returns the Metrics related to the provided query options.
func (in *Client) GetMetrics(query *MetricsQuery) Metrics {
	return getMetrics(in.api, query)
}

// GetServiceHealth returns the Health related to the provided service identified by its namespace and service name.
// It reads Envoy metrics, inbound and outbound
// When the health is unavailable, total number of members will be 0.
func (in *Client) GetServiceHealth(namespace, servicename string, ports []int32) (EnvoyServiceHealth, error) {
	return getServiceHealth(in.api, namespace, servicename, ports)
}

// GetAllRequestRates queries Prometheus to fetch request counter rates, over a time interval, for requests
// into, internal to, or out of the namespace.
// Returns (rates, error)
func (in *Client) GetAllRequestRates(namespace string, ratesInterval string, queryTime time.Time) (model.Vector, error) {
	return getAllRequestRates(in.api, namespace, queryTime, ratesInterval)
}

// GetNamespaceServicesRequestRates queries Prometheus to fetch request counter rates, over a time interval, limited to
// requests for services in the namespace.
// Returns (rates, error)
func (in *Client) GetNamespaceServicesRequestRates(namespace string, ratesInterval string, queryTime time.Time) (model.Vector, error) {
	return getNamespaceServicesRequestRates(in.api, namespace, queryTime, ratesInterval)
}

// GetServiceRequestRates queries Prometheus to fetch request counters rates over a time interval
// for a given service (hence only inbound).
// Returns (in, error)
func (in *Client) GetServiceRequestRates(namespace, service, ratesInterval string, queryTime time.Time) (model.Vector, error) {
	return getServiceRequestRates(in.api, namespace, service, queryTime, ratesInterval)
}

// GetAppRequestRates queries Prometheus to fetch request counters rates over a time interval
// for a given app, both in and out.
// Returns (in, out, error)
func (in *Client) GetAppRequestRates(namespace, app, ratesInterval string, queryTime time.Time) (model.Vector, model.Vector, error) {
	return getItemRequestRates(in.api, namespace, app, "app", queryTime, ratesInterval)
}

// GetWorkloadRequestRates queries Prometheus to fetch request counters rates over a time interval
// for a given workload, both in and out.
// Returns (in, out, error)
func (in *Client) GetWorkloadRequestRates(namespace, workload, ratesInterval string, queryTime time.Time) (model.Vector, model.Vector, error) {
	return getItemRequestRates(in.api, namespace, workload, "workload", queryTime, ratesInterval)
}

func (in *Client) FetchRateRange(metricName, labels, rateFunc, rateInterval, grouping string, bounds v1.Range) *Metric {
	return fetchRateRange(in.api, metricName, labels, rateFunc, rateInterval, grouping, bounds)
}

func (in *Client) FetchHistogramRange(metricName, labels, rateInterval, grouping string, bounds v1.Range, avg bool, quantiles []string) Histogram {
	return fetchHistogramRange(in.api, metricName, labels, rateInterval, grouping, bounds, avg, quantiles)
}

// API returns the Prometheus V1 HTTP API for performing calls not supported natively by this client
func (in *Client) API() v1.API {
	return in.api
}

// Address return the configured Prometheus service URL
func (in *Client) Address() string {
	return config.Get().ExternalServices.PrometheusServiceURL
}
