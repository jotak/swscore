package business

import (
	"fmt"
	"strings"

	"github.com/kiali/k-charted/kubernetes/v1alpha1"
	"github.com/prometheus/common/log"
	pmod "github.com/prometheus/common/model"

	"github.com/kiali/kiali/config"
	"github.com/kiali/kiali/kubernetes"
	"github.com/kiali/kiali/models"
	"github.com/kiali/kiali/prometheus"
)

// GenericGraphService ...
type GenericGraphService struct {
	prom prometheus.ClientInterface
	k8s  kubernetes.ClientInterface
}

func (in *GenericGraphService) loadGraphAdapter(namespace, name string) (*kubernetes.GraphAdapter, error) {
	adapter, err := in.k8s.GetGraphAdapter(namespace, name)
	globalNamespace := config.Get().IstioNamespace
	if err != nil && globalNamespace != "" {
		adapter, err = in.k8s.GetGraphAdapter(globalNamespace, name)
		if err != nil {
			return nil, err
		}
	}
	return adapter, err
}

func (in *GenericGraphService) loadAllGraphAdapters(namespace string) ([]kubernetes.GraphAdapter, error) {
	// From specific namespace
	adapters, err := in.k8s.GetGraphAdapters(namespace)
	if err != nil {
		return nil, err
	}

	// From global namespace
	globalNamespace := config.Get().IstioNamespace
	if globalNamespace != "" && namespace != globalNamespace {
		globals, err := in.k8s.GetGraphAdapters(globalNamespace)
		if err != nil {
			return nil, err
		}
		for _, a1 := range globals {
			duplicate := false
			for _, a2 := range adapters {
				if a1.Name == a2.Name {
					duplicate = true
					break
				}
			}
			if !duplicate {
				adapters = append(adapters, a1)
			}
		}
	}

	return adapters, nil
}

// GetGraphAdapters fetches and returns the names of all adapters for given namespace
func (in *GenericGraphService) GetGraphAdapters(q models.GraphQuery) (*models.AdaptersInfo, error) {
	all, err := in.loadAllGraphAdapters(q.Namespace)
	if err != nil {
		return nil, err
	}
	if len(all) == 0 {
		return &models.AdaptersInfo{List: []models.TitleAndName{}}, nil
	}
	list := []models.TitleAndName{}
	for _, adapter := range all {
		list = append(list, models.TitleAndName{Title: adapter.Spec.Title, Name: adapter.Name})
	}

	first := all[0].Spec
	if len(first.Aggregations) == 0 {
		return &models.AdaptersInfo{
			List:  list,
			First: &models.GraphResponse{Adapter: first},
		}, nil
	}
	firstAgg := first.Aggregations[0]
	g, err := in.generateGraph(q, first, firstAgg)
	if err != nil {
		return nil, err
	}

	return &models.AdaptersInfo{List: list, First: g}, nil
}

func (in *GenericGraphService) GetGraph(q models.GraphQuery) (*models.GraphResponse, error) {
	adapter, err := in.loadGraphAdapter(q.Namespace, q.GraphAdapter)
	if err != nil {
		return nil, err
	}

	if len(adapter.Spec.Aggregations) == 0 {
		return &models.GraphResponse{
			Adapter: adapter.Spec,
		}, nil
	}

	for _, agg := range adapter.Spec.Aggregations {
		if agg.Name == q.AggregationLevel {
			return in.generateGraph(q, adapter.Spec, agg)
		}
	}
	// Aggregation not found
	log.Warnf("Aggregation %s was not found in adapter %s", q.AggregationLevel, q.GraphAdapter)
	return &models.GraphResponse{Adapter: adapter.Spec}, nil
}

func (in *GenericGraphService) generateGraph(q models.GraphQuery, adapter kubernetes.GraphAdapterSpec, agg kubernetes.GraphAdapterAggregation) (*models.GraphResponse, error) {
	allEdges := make(map[string]models.Edge)
	// Note: allEdgeLabels will be merged into allEdges ultimately, but needs to be kept separate for edges that require labelling but not used in graph generation
	allEdgeLabels := make(map[string][]models.EdgeLabel)

	log.Infof("Aggregation: %v", agg)
	filterFromInside := agg.SourceNamespaceLabel + "=\"" + q.Namespace + "\""
	filterFromOutside := agg.SourceNamespaceLabel + "!=\"" + q.Namespace + "\"," + agg.DestNamespaceLabel + "=\"" + q.Namespace + "\""
	allLabels := append(agg.SourceLabels, agg.DestLabels...)
	allLabels = append(allLabels, agg.SourceNamespaceLabel)
	allLabels = append(allLabels, agg.DestNamespaceLabel)
	groupBy := strings.Join(allLabels, ",")
	// TODO: add grouping by intermediate nodes

	for _, metric := range adapter.Metrics {
		// TODO: add other Functions (p99 etc.)
		if metric.Function == v1alpha1.Rate {
			fromInside, errInside := in.prom.FetchRatePoint(metric.Query, concatFilters(filterFromInside, metric.Filters), groupBy, q.Time, q.Duration)
			if errInside != nil {
				return nil, fmt.Errorf("could not generate graph, fetching 'from inside': %v", errInside)
			}
			in.processPromResult(fromInside, metric, agg, allEdges, allEdgeLabels)
			fromOutside, errOutside := in.prom.FetchRatePoint(metric.Query, concatFilters(filterFromOutside, metric.Filters), groupBy, q.Time, q.Duration)
			if errOutside != nil {
				return nil, fmt.Errorf("could not generate graph, fetching 'from outside': %v", errOutside)
			}
			in.processPromResult(fromOutside, metric, agg, allEdges, allEdgeLabels)
		}
	}

	// Build final list
	edges := []models.Edge{}
	for id, edge := range allEdges {
		if labels, ok := allEdgeLabels[id]; ok {
			edge.Labels = labels
		} else {
			edge.Labels = []models.EdgeLabel{}
		}
		edges = append(edges, edge)
	}

	return &models.GraphResponse{
		Adapter: adapter,
		Edges:   edges,
	}, nil
}

func concatFilters(first, second string) string {
	if second == "" {
		return "{" + first + "}"
	}
	return "{" + first + "," + second + "}"
}

func (in *GenericGraphService) processPromResult(v pmod.Vector, metric kubernetes.GraphAdapterMetric, agg kubernetes.GraphAdapterAggregation, allEdges map[string]models.Edge, allEdgeLabels map[string][]models.EdgeLabel) {
	scale := 1.0
	if metric.UnitScale != 0.0 {
		scale = metric.UnitScale
	}

	for _, item := range v {
		sourceID := buildID(item, append(agg.SourceLabels, agg.SourceNamespaceLabel))
		destID := buildID(item, append(agg.DestLabels, agg.DestNamespaceLabel))
		edgeID := sourceID + "~" + destID
		if metric.GeneratesGraph {
			allEdges[edgeID] = models.Edge{SourceID: sourceID, DestID: destID}
		}
		if metric.EdgeLabels {
			val := float64(item.Value) * scale
			allEdgeLabels[edgeID] = append(allEdgeLabels[edgeID], models.EdgeLabel{
				Name:  metric.Name,
				Unit:  metric.Unit,
				Value: val,
			})
		}
	}
}

func buildID(sample *pmod.Sample, labels []string) string {
	ids := []string{}
	for _, label := range labels {
		if val, ok := sample.Metric[pmod.LabelName(label)]; ok {
			ids = append(ids, string(val))
		} else {
			ids = append(ids, "")
		}
	}
	return strings.Join(ids, ",")
}
