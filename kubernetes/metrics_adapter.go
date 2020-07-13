package kubernetes

import (
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// GetGraphAdapter returns a GraphAdapter for the given name
func (in *K8SClient) GetGraphAdapter(namespace, name string) (*GraphAdapter, error) {
	result := GraphAdapter{}
	err := in.graphAdapterAPI.Get().Namespace(namespace).Resource(GraphAdapters).SubResource(name).Do().Into(&result)
	if err != nil {
		return nil, err
	}
	return &result, err
}

// GetGraphAdapters returns all GraphAdapters from the given namespace
func (in *K8SClient) GetGraphAdapters(namespace string) ([]GraphAdapter, error) {
	result := GraphAdaptersList{}
	err := in.graphAdapterAPI.Get().Namespace(namespace).Resource(GraphAdapters).Do().Into(&result)
	if err != nil {
		return nil, err
	}
	return result.Items, nil
}

type GraphAdapter struct {
	meta_v1.TypeMeta   `json:",inline"`
	meta_v1.ObjectMeta `json:"metadata"`
	Spec               GraphAdapterSpec `json:"spec"`
}

type GraphAdaptersList struct {
	meta_v1.TypeMeta `json:",inline"`
	meta_v1.ListMeta `json:"metadata"`
	Items            []GraphAdapter `json:"items"`
}

type GraphAdapterSpec struct {
	Title             string                         `json:"title"`             // Shows as Graph Provider
	Metrics           []GraphAdapterMetric           `json:"metrics"`           // List all  metrics to fetch, either for graph generation, or edge annotations, or both
	Aggregations      []GraphAdapterAggregation      `json:"aggregations"`      // Here are defined different levels of aggregations (~ Graph Type)
	IntermediateNodes []GraphAdapterIntermediateNode `json:"intermediateNodes"` // Intermediate nodes can be determined out of other prometheus labels, and display can be turned on/off
}

type GraphAdapterMetric struct {
	Name            string                      `json:"name"`            // This name will appear in toolbar for edge label mode selection
	Query           string                      `json:"query"`           // Query to run (will be transformed based on desired level of aggregation and the "function" to apply)
	Filters         string                      `json:"filters"`         // Prometheus filters that apply on Query
	Function        string                      `json:"function"`        // Function to apply: "raw" (no transformation, typical for gauges), "rate" (typical for counters), "p50", "p95", "p99" or "avg" (for histograms)
	Unit            string                      `json:"unit"`            // Unit of stored values
	UnitScale       float64                     `json:"unitScale"`       // Multiplier to apply on values for display
	GeneratesGraph  bool                        `json:"generatesGraph"`  // Tells if this metric must be used for graph generation (nodes & edges)
	EdgeLabels      bool                        `json:"edgeLabels"`      // Tells if this metric must be used for edges annotations (edge labels)
	ErrorEvaluation GraphAdapterErrorEvaluation `json:"errorEvaluation"` // Optional: how to evaluate error rates
}

type GraphAdapterErrorEvaluation struct {
	FromLabel GraphAdapterLabelEvaluation `json:"fromLabel"` // Evaluate error rates based on label...
	FromQuery string                      `json:"fromQuery"` // Evaluate error rates based on an alternative metrics (e.g. error counter)
}

type GraphAdapterLabelEvaluation struct {
	Name  string `json:"name"`  // Name of the label in prometheus
	Regex string `json:"regex"` // Regex that tells if it's an error
}

type GraphAdapterAggregation struct {
	Name                 string   `json:"name"`                 // Name to show in Aggregation (Graph Type) selector in the UI
	SourceLabels         []string `json:"sourceLabels"`         // Which prom label, or combination of labels, is used to identify sources of that type
	DestLabels           []string `json:"destLabels"`           // Which prom label, or combination of labels, is used to identify destinations of that type
	SourceNamespaceLabel string   `json:"sourceNamespaceLabel"` // Label identifying source namespace
	DestNamespaceLabel   string   `json:"destNamespaceLabel"`   // Label identifying destination namespace
	Shape                string   `json:"shape"`                // Shape for graph display
}

type GraphAdapterIntermediateNode struct {
	Name   string   `json:"name"`   // Name of intermediate node to show in UI for on/off switch
	Labels []string `json:"labels"` // Prometheus label that identifies such nodes
	Shape  string   `json:"shape"`  // Shape for display in graph
}

// TODO: auto-generate the following deepcopy methods!

func (in *GraphAdapter) DeepCopyInto(out *GraphAdapter) {
	*out = *in
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
}

func (in *GraphAdapter) DeepCopy() *GraphAdapter {
	if in == nil {
		return nil
	}
	out := new(GraphAdapter)
	in.DeepCopyInto(out)
	return out
}

func (in *GraphAdapter) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

func (in *GraphAdaptersList) DeepCopyInto(out *GraphAdaptersList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]GraphAdapter, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

func (in *GraphAdaptersList) DeepCopy() *GraphAdaptersList {
	if in == nil {
		return nil
	}
	out := new(GraphAdaptersList)
	in.DeepCopyInto(out)
	return out
}

func (in *GraphAdaptersList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}
