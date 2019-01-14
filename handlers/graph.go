package handlers

// Graph.go provides handlers for graph request endpoints.   The handlers return configuration
// for a specified vendor (default cytoscape).  The configuration format is vendor-specific, typically
// JSON, and provides what is necessary to allow the vendor's graphing tool to render the graph.
//
// The algorithm is three-pass:
//   First Pass: Query Prometheus (istio-requests-total metric) to retrieve the source-destination
//               dependencies. Build a traffic map to provide a full representation of nodes and edges.
//
//   Second Pass: Apply any requested appenders to alter or append to the graph.
//
//   Third Pass: Supply the traffic map to a vendor-specific config generator that
//               constructs the vendor-specific output.
//
// The current Handlers:
//   GraphNamespace:  Generate a graph for all services in a namespace (whether source or destination)
//   GraphNode:       Generate a graph centered on a specified node, limited to requesting and requested nodes.
//
// The handlers accept the following query parameters (some handlers may ignore some parameters):
//   appenders:      Comma-separated list of appenders to run from [circuit_breaker, unused_service...] (default all)
//                   Note, appenders may support appender-specific query parameters
//   duration:       time.Duration indicating desired query range duration, (default 10m)
//   graphType:      Determines how to present the telemetry data. app | service | versionedApp | workload (default workload)
//   groupBy:        If supported by vendor, visually group by a specified node attribute (default version)
//   includeIstio:   Include istio-system (infra) services (default false)
//   namespaces:     Comma-separated list of namespace names to use in the graph. Will override namespace path param
//   queryTime:      Unix time (seconds) for query such that range is queryTime-duration..queryTime (default now)
//   vendor:         cytoscape (default cytoscape)
//
// * Error% is the percentage of requests with response code != 2XX
// * See the vendor-specific config generators for more details about the specific vendor.
//
import (
	"context"
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"

	"github.com/kiali/kiali/config"
	"github.com/kiali/kiali/graph"
	"github.com/kiali/kiali/graph/appender"
	"github.com/kiali/kiali/graph/cytoscape"
	"github.com/kiali/kiali/graph/options"
	"github.com/kiali/kiali/log"
	"github.com/kiali/kiali/prometheus"
	"github.com/kiali/kiali/prometheus/internalmetrics"
)

// GraphNamespaces is a REST http.HandlerFunc handling graph generation for 1 or more namespaces
func GraphNamespaces(w http.ResponseWriter, r *http.Request) {
	defer handlePanic(w)

	client, err := prometheus.NewClient()
	graph.CheckError(err)

	graphNamespaces(w, r, client)
}

// graphNamespaces provides a testing hook that can supply a mock client
func graphNamespaces(w http.ResponseWriter, r *http.Request, client *prometheus.Client) {
	o := options.NewOptions(r)

	// time how long it takes to generate this graph
	promtimer := internalmetrics.GetGraphGenerationTimePrometheusTimer(o.GetGraphKind(), o.GraphType, o.InjectServiceNodes)
	defer promtimer.ObserveDuration()

	trafficMap := buildNamespacesTrafficMap(o, client)
	generateGraph(trafficMap, w, o)

	// update metrics
	internalmetrics.SetGraphNodes(o.GetGraphKind(), o.GraphType, o.InjectServiceNodes, len(trafficMap))
}

func buildNamespacesTrafficMap(o options.Options, client *prometheus.Client) graph.TrafficMap {
	switch o.Vendor {
	case "cytoscape":
	default:
		graph.Error(fmt.Sprintf("Vendor [%s] not supported", o.Vendor))
	}

	log.Debugf("Build [%s] graph for [%v] namespaces [%s]", o.GraphType, len(o.Namespaces), o.Namespaces)

	trafficMap := graph.NewTrafficMap()

	globalInfo := appender.NewGlobalInfo()
	for _, namespace := range o.Namespaces {
		log.Debugf("Build traffic map for namespace [%s]", namespace)
		namespaceTrafficMap := buildNamespaceTrafficMap(namespace.Name, o, client)
		namespaceInfo := appender.NewNamespaceInfo(namespace.Name)
		for _, a := range o.Appenders {
			appenderTimer := internalmetrics.GetGraphAppenderTimePrometheusTimer(a.Name())
			a.AppendGraph(namespaceTrafficMap, globalInfo, namespaceInfo)
			appenderTimer.ObserveDuration()
		}
		mergeTrafficMaps(trafficMap, namespace.Name, namespaceTrafficMap)
	}

	// The appenders can add/remove/alter nodes. After the manipulations are complete
	// we can make some final adjustments:
	// - mark the outsiders (i.e. nodes not in the requested namespaces)
	// - mark the insider traffic generators (i.e. inside the namespace and only outgoing edges)
	markOutsideOrInaccessible(trafficMap, o)
	markTrafficGenerators(trafficMap)

	if graph.GraphTypeService == o.GraphType {
		trafficMap = reduceToServiceGraph(trafficMap)
	}

	return trafficMap
}

// mergeTrafficMaps ensures that we only have unique nodes by removing duplicate
// nodes and merging their edges.  When removing a duplicate prefer an instance
// from the namespace being merged-in because it is guaranteed to have all appender
// information applied. We also need to avoid duplicate edges, it can
// happen when an terminal node of one namespace is a root node of another:
//   ns1 graph: unknown -> ns1:A -> ns2:B
//   ns2 graph:   ns1:A -> ns2:B -> ns2:C
func mergeTrafficMaps(trafficMap graph.TrafficMap, ns string, nsTrafficMap graph.TrafficMap) {
	for nsId, nsNode := range nsTrafficMap {
		if node, isDup := trafficMap[nsId]; isDup {
			if nsNode.Namespace == ns {
				// prefer nsNode (see above comment), so do a swap
				trafficMap[nsId] = nsNode
				temp := node
				node = nsNode
				nsNode = temp
			}
			for _, nsEdge := range nsNode.Edges {
				isDupEdge := false
				for _, e := range node.Edges {
					if nsEdge.Dest.ID == e.Dest.ID && nsEdge.Metadata["protocol"] == e.Metadata["protocol"] {
						isDupEdge = true
						break
					}
				}
				if !isDupEdge {
					node.Edges = append(node.Edges, nsEdge)
				}
			}
		} else {
			trafficMap[nsId] = nsNode
		}
	}
}

func markOutsideOrInaccessible(trafficMap graph.TrafficMap, o options.Options) {
	for _, n := range trafficMap {
		switch n.NodeType {
		case graph.NodeTypeUnknown:
			n.Metadata["isInaccessible"] = true
		case graph.NodeTypeService:
			if _, ok := n.Metadata["isServiceEntry"]; ok {
				n.Metadata["isInaccessible"] = true
			} else {
				if isOutside(n, o.Namespaces) {
					n.Metadata["isOutside"] = true
				}
			}
		default:
			if isOutside(n, o.Namespaces) {
				n.Metadata["isOutside"] = true
			}
		}
		if isOutsider, ok := n.Metadata["isOutside"]; ok && isOutsider.(bool) {
			if _, ok2 := n.Metadata["isInaccessible"]; !ok2 {
				if isInaccessible(n, o.AccessibleNamespaces) {
					n.Metadata["isInaccessible"] = true
				}
			}
		}
	}
}

func isOutside(n *graph.Node, namespaces map[string]graph.NamespaceInfo) bool {
	if n.Namespace == graph.Unknown {
		return false
	}
	for _, ns := range namespaces {
		if n.Namespace == ns.Name {
			return false
		}
	}
	return true
}

func isInaccessible(n *graph.Node, accessibleNamespaces map[string]time.Time) bool {
	if _, found := accessibleNamespaces[n.Namespace]; !found {
		return true
	} else {
		return false
	}
}

func markTrafficGenerators(trafficMap graph.TrafficMap) {
	destMap := make(map[string]*graph.Node)
	for _, n := range trafficMap {
		for _, e := range n.Edges {
			destMap[e.Dest.ID] = e.Dest
		}
	}
	for _, n := range trafficMap {
		if len(n.Edges) == 0 {
			continue
		}
		if _, isDest := destMap[n.ID]; !isDest {
			n.Metadata["isRoot"] = true
		}
	}
}

// reduceToServicGraph compresses a [service-injected workload] graph by removing
// the workload nodes such that, with exception of non-service root nodes, the resulting
// graph has edges only from and to service nodes.
func reduceToServiceGraph(trafficMap graph.TrafficMap) graph.TrafficMap {
	reducedTrafficMap := graph.NewTrafficMap()

	for id, n := range trafficMap {
		if n.NodeType != graph.NodeTypeService {
			// if node isRoot then keep it to better understand traffic flow
			if val, ok := n.Metadata["isRoot"]; ok && val.(bool) {
				reducedTrafficMap[id] = n
			}
			continue
		}

		// handle service node, add to reduced traffic map and generate new edges
		reducedTrafficMap[id] = n
		workloadEdges := n.Edges
		n.Edges = []*graph.Edge{}
		for _, workloadEdge := range workloadEdges {
			workload := workloadEdge.Dest
			checkNodeType(graph.NodeTypeWorkload, workload)
			for _, serviceEdge := range workload.Edges {
				childService := serviceEdge.Dest
				checkNodeType(graph.NodeTypeService, childService)
				var edge *graph.Edge
				for _, e := range n.Edges {
					if childService.ID == e.Dest.ID && serviceEdge.Metadata["protocol"] == e.Metadata["protocol"] {
						edge = e
						break
					}
				}
				if nil == edge {
					n.Edges = append(n.Edges, serviceEdge)
				} else {
					addServiceGraphTraffic(edge, serviceEdge)
				}
			}
		}
	}

	return reducedTrafficMap
}

func addServiceGraphTraffic(target, source *graph.Edge) {
	protocol := target.Metadata["protocol"]
	switch protocol {
	case "http":
		addToMetadataValue(target.Metadata, "http", source.Metadata["http"].(float64))
		if val, ok := source.Metadata["http3xx"]; ok {
			addToMetadataValue(target.Metadata, "http3xx", val.(float64))
		}
		if val, ok := source.Metadata["http4xx"]; ok {
			addToMetadataValue(target.Metadata, "http4xx", val.(float64))
		}
		if val, ok := source.Metadata["http5xx"]; ok {
			addToMetadataValue(target.Metadata, "http5xx", val.(float64))
		}
	case "tcp":
		addToMetadataValue(target.Metadata, "tcp", source.Metadata["tcp"].(float64))
	default:
		graph.Error(fmt.Sprintf("Unexpected edge protocol [%v] for edge [%+v]", protocol, target))
	}
	// hande any appender-based edge data
	// - responseTime is not a counter, set an average, not a total
	if responseTime, ok := source.Metadata["responseTime"]; ok {
		averageMetadataValue(target.Metadata, "responseTime", responseTime.(float64))
	}
}

func checkNodeType(expected string, n *graph.Node) {
	if expected != n.NodeType {
		graph.Error(fmt.Sprintf("Expected nodeType [%s] for node [%+v]", expected, n))
	}
}

// buildNamespaceTrafficMap returns a map of all namespace nodes (key=id).  All
// nodes either directly send and/or receive requests from a node in the namespace.
func buildNamespaceTrafficMap(namespace string, o options.Options, client *prometheus.Client) graph.TrafficMap {
	httpMetric := "request_total"
	duration := o.Namespaces[namespace].Duration

	// // query prometheus for request traffic in three queries:
	// // 1) query for traffic originating from "unknown" (i.e. the internet).
	groupBy := "namespace,deployment,dst_namespace,dst_service,dst_deployment"
	// query := fmt.Sprintf(`sum(rate(%s{reporter="destination",source_workload="unknown",destination_service_namespace="%s",response_code=~"%s"} [%vs])) by (%s)`,
	// 	httpMetric,
	// 	namespace,
	// 	"[2345][0-9][0-9]",      // regex for valid response_codes
	// 	int(duration.Seconds()), // range duration for the query
	// 	groupBy)
	// unkVector := promQuery(query, time.Unix(o.QueryTime, 0), client.API())

	// // 2) query for traffic originating from a workload outside of the namespace.  Exclude any "unknown" source telemetry (an unusual corner case)
	// query = fmt.Sprintf(`sum(rate(%s{reporter="source",source_workload_namespace!="%s",source_workload!="unknown",destination_service_namespace="%s",response_code=~"%s"} [%vs])) by (%s)`,
	// 	httpMetric,
	// 	namespace,
	// 	namespace,
	// 	"[2345][0-9][0-9]",      // regex for valid response_codes
	// 	int(duration.Seconds()), // range duration for the query
	// 	groupBy)

	// // fetch the externally originating request traffic time-series
	// extVector := promQuery(query, time.Unix(o.QueryTime, 0), client.API())

	// 3) query for traffic originating from a workload inside of the namespace
	// query = fmt.Sprintf(`sum(rate(%s{reporter="source",source_workload_namespace="%s",response_code=~"%s"} [%vs])) by (%s)`,
	query := fmt.Sprintf(`sum(rate(%s{namespace="%s"} [%vs])) by (%s)`,
		httpMetric,
		namespace,
		int(duration.Seconds()), // range duration for the query
		groupBy)

	// fetch the internally originating request traffic time-series
	intVector := promQuery(query, time.Unix(o.QueryTime, 0), client.API())

	// create map to aggregate traffic by response code
	trafficMap := graph.NewTrafficMap()
	populateTrafficMapHttp(trafficMap, &intVector, o)

	// istio component telemetry is only reported destination-side, so we must perform additional queries
	if o.IncludeIstio {
		istioNamespace := config.Get().IstioNamespace

		// 4) if the target namespace is istioNamespace re-query for traffic originating from a workload outside of the namespace
		if namespace == istioNamespace {
			query = fmt.Sprintf(`sum(rate(%s{reporter="destination",source_workload_namespace!="%s",destination_service_namespace="%s",response_code=~"%s"} [%vs])) by (%s)`,
				httpMetric,
				namespace,
				namespace,
				"[2345][0-9][0-9]",      // regex for valid response_codes
				int(duration.Seconds()), // range duration for the query
				groupBy)

			// fetch the externally originating request traffic time-series
			extIstioVector := promQuery(query, time.Unix(o.QueryTime, 0), client.API())
			populateTrafficMapHttp(trafficMap, &extIstioVector, o)
		}

		// 5) supplemental query for traffic originating from a workload inside of the namespace with istioSystem destination
		query = fmt.Sprintf(`sum(rate(%s{reporter="destination",source_workload_namespace="%s",destination_service_namespace="%s",response_code=~"%s"} [%vs])) by (%s)`,
			httpMetric,
			namespace,
			istioNamespace,
			"[2345][0-9][0-9]",      // regex for valid response_codes
			int(duration.Seconds()), // range duration for the query
			groupBy)

		// fetch the internally originating request traffic time-series
		intIstioVector := promQuery(query, time.Unix(o.QueryTime, 0), client.API())
		populateTrafficMapHttp(trafficMap, &intIstioVector, o)
	}

	// Section for TCP services
	tcpMetric := "istio_tcp_sent_bytes_total"

	// 1) query for traffic originating from "unknown" (i.e. the internet)
	tcpGroupBy := "source_workload_namespace,source_workload,source_app,source_version,destination_workload_namespace,destination_service_name,destination_workload,destination_app,destination_version"
	query = fmt.Sprintf(`sum(rate(%s{reporter="destination",source_workload="unknown",destination_workload_namespace="%s"} [%vs])) by (%s)`,
		tcpMetric,
		namespace,
		int(duration.Seconds()), // range duration for the query
		tcpGroupBy)
	tcpUnkVector := promQuery(query, time.Unix(o.QueryTime, 0), client.API())

	// 2) query for traffic originating from a workload outside of the namespace. Exclude any "unknown" source telemetry (an unusual corner case)
	tcpGroupBy = "source_workload_namespace,source_workload,source_app,source_version,destination_service_namespace,destination_service_name,destination_workload,destination_app,destination_version"
	query = fmt.Sprintf(`sum(rate(%s{reporter="source",source_workload_namespace!="%s",source_workload!="unknown",destination_service_namespace="%s"} [%vs])) by (%s)`,
		tcpMetric,
		namespace,
		namespace,
		int(duration.Seconds()), // range duration for the query
		tcpGroupBy)
	tcpExtVector := promQuery(query, time.Unix(o.QueryTime, 0), client.API())

	// 3) query for traffic originating from a workload inside of the namespace
	query = fmt.Sprintf(`sum(rate(%s{reporter="source",source_workload_namespace="%s"} [%vs])) by (%s)`,
		tcpMetric,
		namespace,
		int(duration.Seconds()), // range duration for the query
		tcpGroupBy)
	tcpInVector := promQuery(query, time.Unix(o.QueryTime, 0), client.API())

	populateTrafficMapTcp(trafficMap, &tcpUnkVector, o)
	populateTrafficMapTcp(trafficMap, &tcpExtVector, o)
	populateTrafficMapTcp(trafficMap, &tcpInVector, o)

	return trafficMap
}

func populateTrafficMapHttp(trafficMap graph.TrafficMap, vector *model.Vector, o options.Options) {
	for _, s := range *vector {
		m := s.Metric
		lSourceWlNs, sourceWlNsOk := m["namespace"]
		lSourceWl, sourceWlOk := m["deployment"]
		lDestSvcNs, destSvcNsOk := m["dst_namespace"]
		lDestSvcName, destSvcNameOk := m["dst_service"]
		lDestWl, destWlOk := m["dst_deployment"]

		if !sourceWlNsOk || !sourceWlOk || !destSvcNsOk || !destSvcNameOk || !destWlOk {
			log.Warningf("Skipping %s, missing expected TS labels", m.String())
			continue
		}

		sourceWlNs := string(lSourceWlNs)
		sourceWl := string(lSourceWl)
		sourceApp := "Unknown"
		sourceVer := "Unknown"
		destSvcNs := string(lDestSvcNs)
		destSvcName := string(lDestSvcName)
		destWl := string(lDestWl)
		destApp := "Unknown"
		destVer := "Unknown"
		code := "200"

		val := float64(s.Value)

		if o.InjectServiceNodes {
			// don't inject a service node if the dest node is already a service node.  Also, we can't inject if destSvcName is not set.
			_, destNodeType := graph.Id(destSvcNs, destWl, destApp, destVer, destSvcName, o.GraphType)
			if destSvcNameOk && destNodeType != graph.NodeTypeService {
				addHttpTraffic(trafficMap, val, code, sourceWlNs, sourceWl, sourceApp, sourceVer, "", destSvcNs, "", "", "", destSvcName, o)
				addHttpTraffic(trafficMap, val, code, destSvcNs, "", "", "", destSvcName, destSvcNs, destWl, destApp, destVer, destSvcName, o)
			} else {
				addHttpTraffic(trafficMap, val, code, sourceWlNs, sourceWl, sourceApp, sourceVer, "", destSvcNs, destWl, destApp, destVer, destSvcName, o)
			}
		} else {
			addHttpTraffic(trafficMap, val, code, sourceWlNs, sourceWl, sourceApp, sourceVer, "", destSvcNs, destWl, destApp, destVer, destSvcName, o)
		}
	}
}

func addHttpTraffic(trafficMap graph.TrafficMap, val float64, code, sourceWlNs, sourceWl, sourceApp, sourceVer, sourceSvcName, destSvcNs, destWl, destApp, destVer, destSvcName string, o options.Options) (source, dest *graph.Node) {

	source, sourceFound := addNode(trafficMap, sourceWlNs, sourceWl, sourceApp, sourceVer, sourceSvcName, o)
	dest, destFound := addNode(trafficMap, destSvcNs, destWl, destApp, destVer, destSvcName, o)

	addToDestServices(dest.Metadata, destSvcName)

	var edge *graph.Edge
	for _, e := range source.Edges {
		if dest.ID == e.Dest.ID && e.Metadata["protocol"] == "http" {
			edge = e
			break
		}
	}
	if nil == edge {
		edge = source.AddEdge(dest)
		edge.Metadata["protocol"] = "http"
	}

	// A workload may mistakenly have multiple app and or version label values.
	// This is a misconfiguration we need to handle. See Kiali-1309.
	if sourceFound {
		handleMisconfiguredLabels(source, sourceApp, sourceVer, val, o)
	}
	if destFound {
		handleMisconfiguredLabels(dest, destApp, destVer, val, o)
	}

	addToMetadataValue(source.Metadata, "httpOut", val)
	addToMetadataValue(dest.Metadata, "httpIn", val)
	addToMetadataValue(edge.Metadata, "http", val)

	// note, we don't track 2xx because it's not used downstream and can be easily
	// calculated: 2xx = (rate - 3xx - 4xx - 5xx)
	switch {
	case strings.HasPrefix(string(code), "3"):
		addToMetadataValue(dest.Metadata, "httpIn3xx", val)
		addToMetadataValue(edge.Metadata, "http3xx", val)
	case strings.HasPrefix(string(code), "4"):
		addToMetadataValue(dest.Metadata, "httpIn4xx", val)
		addToMetadataValue(edge.Metadata, "http4xx", val)
	case strings.HasPrefix(string(code), "5"):
		addToMetadataValue(dest.Metadata, "httpIn5xx", val)
		addToMetadataValue(edge.Metadata, "http5xx", val)
	}

	return source, dest
}

func populateTrafficMapTcp(trafficMap graph.TrafficMap, vector *model.Vector, o options.Options) {
	for _, s := range *vector {
		m := s.Metric
		lSourceWlNs, sourceWlNsOk := m["source_workload_namespace"]
		lSourceWl, sourceWlOk := m["source_workload"]
		lSourceApp, sourceAppOk := m["source_app"]
		lSourceVer, sourceVerOk := m["source_version"]
		lDestSvcNs, destSvcNsOk := m["destination_service_namespace"]
		lDestSvcName, destSvcNameOk := m["destination_service_name"]
		lDestWl, destWlOk := m["destination_workload"]
		lDestApp, destAppOk := m["destination_app"]
		lDestVer, destVerOk := m["destination_version"]

		// TCP queries doesn't use destination_service_namespace for the unknown node.
		// Check if this is the case and use destination_workload_namespace
		if !destSvcNsOk {
			lDestSvcNs, destSvcNsOk = m["destination_workload_namespace"]
		}

		if !sourceWlNsOk || !sourceWlOk || !sourceAppOk || !sourceVerOk || !destSvcNsOk || !destSvcNameOk || !destWlOk || !destAppOk || !destVerOk {
			log.Warningf("Skipping %s, missing expected TS labels", m.String())
			continue
		}

		sourceWlNs := string(lSourceWlNs)
		sourceWl := string(lSourceWl)
		sourceApp := string(lSourceApp)
		sourceVer := string(lSourceVer)
		destSvcNs := string(lDestSvcNs)
		destSvcName := string(lDestSvcName)
		destWl := string(lDestWl)
		destApp := string(lDestApp)
		destVer := string(lDestVer)

		val := float64(s.Value)

		if o.InjectServiceNodes {
			// don't inject a service node if the dest node is already a service node.  Also, we can't inject if destSvcName is not set.
			destSvcNameOk = graph.IsOK(destSvcName)
			_, destNodeType := graph.Id(destSvcNs, destWl, destApp, destVer, destSvcName, o.GraphType)
			if destSvcNameOk && destNodeType != graph.NodeTypeService {
				addTcpTraffic(trafficMap, val, sourceWlNs, sourceWl, sourceApp, sourceVer, "", destSvcNs, "", "", "", destSvcName, o)
				addTcpTraffic(trafficMap, val, destSvcNs, "", "", "", destSvcName, destSvcNs, destWl, destApp, destVer, destSvcName, o)
			} else {
				addTcpTraffic(trafficMap, val, sourceWlNs, sourceWl, sourceApp, sourceVer, "", destSvcNs, destWl, destApp, destVer, destSvcName, o)
			}
		} else {
			addTcpTraffic(trafficMap, val, sourceWlNs, sourceWl, sourceApp, sourceVer, "", destSvcNs, destWl, destApp, destVer, destSvcName, o)
		}
	}
}

func addTcpTraffic(trafficMap graph.TrafficMap, val float64, sourceWlNs, sourceWl, sourceApp, sourceVer, sourceSvcName, destSvcNs, destWl, destApp, destVer, destSvcName string, o options.Options) (source, dest *graph.Node) {

	source, sourceFound := addNode(trafficMap, sourceWlNs, sourceWl, sourceApp, sourceVer, sourceSvcName, o)
	dest, destFound := addNode(trafficMap, destSvcNs, destWl, destApp, destVer, destSvcName, o)

	addToDestServices(dest.Metadata, destSvcName)

	var edge *graph.Edge
	for _, e := range source.Edges {
		if dest.ID == e.Dest.ID && e.Metadata["procotol"] == "tcp" {
			edge = e
			break
		}
	}
	if nil == edge {
		edge = source.AddEdge(dest)
		edge.Metadata["protocol"] = "tcp"
	}

	// A workload may mistakenly have multiple app and or version label values.
	// This is a misconfiguration we need to handle. See Kiali-1309.
	if sourceFound {
		handleMisconfiguredLabels(source, sourceApp, sourceVer, val, o)
	}
	if destFound {
		handleMisconfiguredLabels(dest, destApp, destVer, val, o)
	}

	addToMetadataValue(source.Metadata, "tcpOut", val)
	addToMetadataValue(dest.Metadata, "tcpIn", val)
	addToMetadataValue(edge.Metadata, "tcp", val)

	return source, dest
}

func addToMetadataValue(md map[string]interface{}, k string, v float64) {
	if curr, ok := md[k]; ok {
		md[k] = curr.(float64) + v
	} else {
		md[k] = v
	}
}

func averageMetadataValue(md map[string]interface{}, k string, v float64) {
	total := v
	count := 1.0
	kTotal := k + "_total"
	kCount := k + "_count"
	if prevTotal, ok := md[kTotal]; ok {
		total += prevTotal.(float64)
	}
	if prevCount, ok := md[kCount]; ok {
		count += prevCount.(float64)
	}
	md[kTotal] = total
	md[kCount] = count
	md[k] = total / count
}

func addToDestServices(md map[string]interface{}, destService string) {
	destServices, ok := md["destServices"]
	if !ok {
		destServices = make(map[string]bool)
		md["destServices"] = destServices
	}
	destServices.(map[string]bool)[destService] = true
}

func handleMisconfiguredLabels(node *graph.Node, app, version string, rate float64, o options.Options) {
	isVersionedAppGraph := o.VendorOptions.GraphType == graph.GraphTypeVersionedApp
	isWorkloadNode := node.NodeType == graph.NodeTypeWorkload
	isVersionedAppNode := node.NodeType == graph.NodeTypeApp && isVersionedAppGraph
	if isWorkloadNode || isVersionedAppNode {
		labels := []string{}
		if node.App != app {
			labels = append(labels, "app")
		}
		if node.Version != version {
			labels = append(labels, "version")
		}
		// prefer the labels of an active time series as often the other labels are inactive
		if len(labels) > 0 {
			node.Metadata["isMisconfigured"] = fmt.Sprintf("labels=%v", labels)
			if rate > 0.0 {
				node.App = app
				node.Version = version
			}
		}
	}
}

func addNode(trafficMap graph.TrafficMap, namespace, workload, app, version, service string, o options.Options) (*graph.Node, bool) {
	id, nodeType := graph.Id(namespace, workload, app, version, service, o.GraphType)
	node, found := trafficMap[id]
	if !found {
		newNode := graph.NewNodeExplicit(id, namespace, workload, app, version, service, nodeType, o.GraphType)
		node = &newNode
		trafficMap[id] = node
	}
	return node, found
}

// GraphNode is a REST http.HandlerFunc handling node-detail graph
// config generation.
func GraphNode(w http.ResponseWriter, r *http.Request) {
	defer handlePanic(w)

	client, err := prometheus.NewClient()
	graph.CheckError(err)

	graphNode(w, r, client)
}

// graphNode provides a testing hook that can supply a mock client
func graphNode(w http.ResponseWriter, r *http.Request, client *prometheus.Client) {
	o := options.NewOptions(r)
	switch o.Vendor {
	case "cytoscape":
	default:
		graph.Error(fmt.Sprintf("Vendor [%s] not supported", o.Vendor))
	}
	if len(o.Namespaces) != 1 {
		graph.Error(fmt.Sprintf("Node graph does not support the 'namespaces' query parameter or the 'all' namespace"))
	}

	// time how long it takes to generate this graph
	promtimer := internalmetrics.GetGraphGenerationTimePrometheusTimer(o.GetGraphKind(), o.GraphType, o.InjectServiceNodes)
	defer promtimer.ObserveDuration()

	n := graph.NewNode(o.NodeOptions.Namespace, o.NodeOptions.Workload, o.NodeOptions.App, o.NodeOptions.Version, o.NodeOptions.Service, o.GraphType)

	log.Debugf("Build graph for node [%+v]", n)

	trafficMap := buildNodeTrafficMap(o.NodeOptions.Namespace, n, o, client)

	globalInfo := appender.NewGlobalInfo()
	namespaceInfo := appender.NewNamespaceInfo(o.NodeOptions.Namespace)

	for _, a := range o.Appenders {
		appenderTimer := internalmetrics.GetGraphAppenderTimePrometheusTimer(a.Name())
		a.AppendGraph(trafficMap, globalInfo, namespaceInfo)
		appenderTimer.ObserveDuration()
	}

	// The appenders can add/remove/alter nodes. After the manipulations are complete
	// we can make some final adjustments:
	// - mark the outsiders (i.e. nodes not in the requested namespaces)
	// - mark the traffic generators
	markOutsideOrInaccessible(trafficMap, o)
	markTrafficGenerators(trafficMap)

	// Note that this is where we would call reduceToServiceGraph for graphTypeService but
	// the current decision is to not reduce the node graph to provide more detail.  This may be
	// confusing to users, we'll see...

	generateGraph(trafficMap, w, o)

	// update metrics
	internalmetrics.SetGraphNodes(o.GetGraphKind(), o.GraphType, o.InjectServiceNodes, len(trafficMap))
}

// buildNodeTrafficMap returns a map of all nodes requesting or requested by the target node (key=id).
func buildNodeTrafficMap(namespace string, n graph.Node, o options.Options, client *prometheus.Client) graph.TrafficMap {
	httpMetric := "istio_requests_total"
	interval := o.Namespaces[namespace].Duration

	// create map to aggregate traffic by response code
	trafficMap := graph.NewTrafficMap()

	// query prometheus for request traffic in two queries:
	// 1) query for incoming traffic
	var query string
	groupBy := "source_workload_namespace,source_workload,source_app,source_version,destination_service_namespace,destination_service_name,destination_workload,destination_app,destination_version,response_code"
	switch n.NodeType {
	case graph.NodeTypeWorkload:
		query = fmt.Sprintf(`sum(rate(%s{reporter="destination",destination_workload_namespace="%s",destination_workload="%s",response_code=~"%s"} [%vs])) by (%s)`,
			httpMetric,
			namespace,
			n.Workload,
			"[2345][0-9][0-9]",      // regex for valid response_codes
			int(interval.Seconds()), // range duration for the query
			groupBy)
	case graph.NodeTypeApp:
		if graph.IsOK(n.Version) {
			query = fmt.Sprintf(`sum(rate(%s{reporter="destination",destination_service_namespace="%s",destination_app="%s",destination_version="%s",response_code=~"%s"} [%vs])) by (%s)`,
				httpMetric,
				namespace,
				n.App,
				n.Version,
				"[2345][0-9][0-9]",      // regex for valid response_codes
				int(interval.Seconds()), // range duration for the query
				groupBy)
		} else {
			query = fmt.Sprintf(`sum(rate(%s{reporter="destination",destination_service_namespace="%s",destination_app="%s",response_code=~"%s"} [%vs])) by (%s)`,
				httpMetric,
				namespace,
				n.App,
				"[2345][0-9][0-9]",      // regex for valid response_codes
				int(interval.Seconds()), // range duration for the query
				groupBy)
		}
	case graph.NodeTypeService:
		// for service requests we want source reporting to capture source-reported errors.  But unknown only generates destination telemetry.  So
		// perform a special query just to capture [successful] request telemetry from unknown.
		query = fmt.Sprintf(`sum(rate(%s{reporter="destination",source_workload="unknown",destination_service_namespace="%s",destination_service_name="%s",response_code=~"%s"} [%vs])) by (%s)`,
			httpMetric,
			namespace,
			n.Service,
			"[2345][0-9][0-9]",      // regex for valid response_codes
			int(interval.Seconds()), // range duration for the query
			groupBy)
		vector := promQuery(query, time.Unix(o.QueryTime, 0), client.API())
		populateTrafficMapHttp(trafficMap, &vector, o)

		query = fmt.Sprintf(`sum(rate(%s{reporter="source",destination_service_namespace="%s",destination_service_name="%s",response_code=~"%s"} [%vs])) by (%s)`,
			httpMetric,
			namespace,
			n.Service,
			"[2345][0-9][0-9]",      // regex for valid response_codes
			int(interval.Seconds()), // range duration for the query
			groupBy)
	default:
		graph.Error(fmt.Sprintf("NodeType [%s] not supported", n.NodeType))
	}
	inVector := promQuery(query, time.Unix(o.QueryTime, 0), client.API())

	// 2) query for outbound traffic
	switch n.NodeType {
	case graph.NodeTypeWorkload:
		query = fmt.Sprintf(`sum(rate(%s{reporter="source",source_workload_namespace="%s",source_workload="%s",response_code=~"%s"} [%vs])) by (%s)`,
			httpMetric,
			namespace,
			n.Workload,
			"[2345][0-9][0-9]",      // regex for valid response_codes
			int(interval.Seconds()), // range duration for the query
			groupBy)
	case graph.NodeTypeApp:
		if graph.IsOK(n.Version) {
			query = fmt.Sprintf(`sum(rate(%s{reporter="source",source_workload_namespace="%s",source_app="%s",source_version="%s",response_code=~"%s"} [%vs])) by (%s)`,
				httpMetric,
				namespace,
				n.App,
				n.Version,
				"[2345][0-9][0-9]",      // regex for valid response_codes
				int(interval.Seconds()), // range duration for the query
				groupBy)
		} else {
			query = fmt.Sprintf(`sum(rate(%s{reporter="source",source_workload_namespace="%s",source_app="%s",response_code=~"%s"} [%vs])) by (%s)`,
				httpMetric,
				namespace,
				n.App,
				"[2345][0-9][0-9]",      // regex for valid response_codes
				int(interval.Seconds()), // range duration for the query
				groupBy)
		}
	case graph.NodeTypeService:
		query = ""
	default:
		graph.Error(fmt.Sprintf("NodeType [%s] not supported", n.NodeType))
	}
	outVector := promQuery(query, time.Unix(o.QueryTime, 0), client.API())

	populateTrafficMapHttp(trafficMap, &inVector, o)
	populateTrafficMapHttp(trafficMap, &outVector, o)

	// istio component telemetry is only reported destination-side, so we must perform additional queries
	if o.IncludeIstio {
		istioNamespace := config.Get().IstioNamespace

		// 3) supplemental query for outbound traffic to the istio namespace
		switch n.NodeType {
		case graph.NodeTypeWorkload:
			query = fmt.Sprintf(`sum(rate(%s{reporter="destination",source_workload_namespace="%s",source_workload="%s",destination_service_namespace="%s",response_code=~"%s"} [%vs])) by (%s)`,
				httpMetric,
				namespace,
				n.Workload,
				istioNamespace,
				"[2345][0-9][0-9]",      // regex for valid response_codes
				int(interval.Seconds()), // range duration for the query
				groupBy)
		case graph.NodeTypeApp:
			if graph.IsOK(n.Version) {
				query = fmt.Sprintf(`sum(rate(%s{reporter="destination",source_workload_namespace="%s",source_app="%s",source_version="%s",destination_service_namespace="%s",response_code=~"%s"} [%vs])) by (%s)`,
					httpMetric,
					namespace,
					n.App,
					n.Version,
					istioNamespace,
					"[2345][0-9][0-9]",      // regex for valid response_codes
					int(interval.Seconds()), // range duration for the query
					groupBy)
			} else {
				query = fmt.Sprintf(`sum(rate(%s{reporter="destination",source_workload_namespace="%s",source_app="%s",destination_service_namespace="%s",response_code=~"%s"} [%vs])) by (%s)`,
					httpMetric,
					namespace,
					n.App,
					istioNamespace,
					"[2345][0-9][0-9]",      // regex for valid response_codes
					int(interval.Seconds()), // range duration for the query
					groupBy)
			}
		case graph.NodeTypeService:
			query = fmt.Sprintf(`sum(rate(%s{reporter="destination",destination_service_namespace="%s",destination_service_name="%s",response_code=~"%s"} [%vs])) by (%s)`,
				httpMetric,
				istioNamespace,
				n.Service,
				"[2345][0-9][0-9]",      // regex for valid response_codes
				int(interval.Seconds()), // range duration for the query
				groupBy)
		default:
			graph.Error(fmt.Sprintf("NodeType [%s] not supported", n.NodeType))
		}
		outIstioVector := promQuery(query, time.Unix(o.QueryTime, 0), client.API())
		populateTrafficMapHttp(trafficMap, &outIstioVector, o)
	}

	// Section for TCP services
	tcpMetric := "istio_tcp_sent_bytes_total"

	tcpGroupBy := "source_workload_namespace,source_workload,source_app,source_version,destination_service_namespace,destination_service_name,destination_workload,destination_app,destination_version"
	switch n.NodeType {
	case graph.NodeTypeWorkload:
		query = fmt.Sprintf(`sum(rate(%s{reporter="source",destination_workload_namespace="%s",destination_workload="%s"} [%vs])) by (%s)`,
			tcpMetric,
			namespace,
			n.Workload,
			int(interval.Seconds()), // range duration for the query
			tcpGroupBy)
	case graph.NodeTypeApp:
		if graph.IsOK(n.Version) {
			query = fmt.Sprintf(`sum(rate(%s{reporter="source",destination_service_namespace="%s",destination_app="%s",destination_version="%s"} [%vs])) by (%s)`,
				tcpMetric,
				namespace,
				n.App,
				n.Version,
				int(interval.Seconds()), // range duration for the query
				tcpGroupBy)
		} else {
			query = fmt.Sprintf(`sum(rate(%s{reporter="source",destination_service_namespace="%s",destination_app="%s"} [%vs])) by (%s)`,
				tcpMetric,
				namespace,
				n.App,
				int(interval.Seconds()), // range duration for the query
				tcpGroupBy)
		}
	case graph.NodeTypeService:
		// TODO: Do we need to handle requests from unknown in a special way (like in HTTP above)? Not sure how tcp is reported from unknown.
		query = fmt.Sprintf(`sum(rate(%s{reporter="source",destination_service_namespace="%s",destination_service_name="%s"} [%vs])) by (%s)`,
			tcpMetric,
			namespace,
			n.Service,
			int(interval.Seconds()), // range duration for the query
			tcpGroupBy)
	default:
		graph.Error(fmt.Sprintf("NodeType [%s] not supported", n.NodeType))
	}
	tcpInVector := promQuery(query, time.Unix(o.QueryTime, 0), client.API())

	// 2) query for outbound traffic
	switch n.NodeType {
	case graph.NodeTypeWorkload:
		query = fmt.Sprintf(`sum(rate(%s{reporter="source",source_workload_namespace="%s",source_workload="%s"} [%vs])) by (%s)`,
			tcpMetric,
			namespace,
			n.Workload,
			int(interval.Seconds()), // range duration for the query
			tcpGroupBy)
	case graph.NodeTypeApp:
		if graph.IsOK(n.Version) {
			query = fmt.Sprintf(`sum(rate(%s{reporter="source",source_workload_namespace="%s",source_app="%s",source_version="%s"} [%vs])) by (%s)`,
				tcpMetric,
				namespace,
				n.App,
				n.Version,
				int(interval.Seconds()), // range duration for the query
				tcpGroupBy)
		} else {
			query = fmt.Sprintf(`sum(rate(%s{reporter="source",source_workload_namespace="%s",source_app="%s"} [%vs])) by (%s)`,
				tcpMetric,
				namespace,
				n.App,
				int(interval.Seconds()), // range duration for the query
				tcpGroupBy)
		}
	case graph.NodeTypeService:
		query = ""
	default:
		graph.Error(fmt.Sprintf("NodeType [%s] not supported", n.NodeType))
	}
	tcpOutVector := promQuery(query, time.Unix(o.QueryTime, 0), client.API())

	populateTrafficMapTcp(trafficMap, &tcpInVector, o)
	populateTrafficMapTcp(trafficMap, &tcpOutVector, o)

	return trafficMap
}

func generateGraph(trafficMap graph.TrafficMap, w http.ResponseWriter, o options.Options) {
	log.Debugf("Generating config for [%s] service graph...", o.Vendor)

	promtimer := internalmetrics.GetGraphMarshalTimePrometheusTimer(o.GetGraphKind(), o.GraphType, o.InjectServiceNodes)
	defer promtimer.ObserveDuration()

	var vendorConfig interface{}
	switch o.Vendor {
	case "cytoscape":
		vendorConfig = cytoscape.NewConfig(trafficMap, o.VendorOptions)
	default:
		graph.Error(fmt.Sprintf("Vendor [%s] not supported", o.Vendor))
	}

	log.Debugf("Done generating config for [%s] service graph.", o.Vendor)
	RespondWithJSONIndent(w, http.StatusOK, vendorConfig)
}

func promQuery(query string, queryTime time.Time, api v1.API) model.Vector {
	if "" == query {
		return model.Vector{}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// wrap with a round() to be in line with metrics api
	query = fmt.Sprintf("round(%s,0.001)", query)
	log.Debugf("Graph query:\n%s@time=%v (now=%v, %v)\n", query, queryTime.Format(graph.TF), time.Now().Format(graph.TF), queryTime.Unix())

	promtimer := internalmetrics.GetPrometheusProcessingTimePrometheusTimer("Graph-Generation")
	value, err := api.Query(ctx, query, queryTime)
	graph.CheckError(err)
	promtimer.ObserveDuration() // notice we only collect metrics for successful prom queries

	switch t := value.Type(); t {
	case model.ValVector: // Instant Vector
		return value.(model.Vector)
	default:
		graph.Error(fmt.Sprintf("No handling for type %v!\n", t))
	}

	return nil
}

func handlePanic(w http.ResponseWriter) {
	code := http.StatusInternalServerError
	if r := recover(); r != nil {
		var message string
		switch r.(type) {
		case string:
			message = r.(string)
		case error:
			message = r.(error).Error()
		case func() string:
			message = r.(func() string)()
		case graph.Response:
			message = r.(graph.Response).Message
			code = r.(graph.Response).Code
		default:
			message = fmt.Sprintf("%v", r)
		}
		if code == http.StatusInternalServerError {
			log.Errorf("%s: %s", message, debug.Stack())
		}
		RespondWithError(w, code, message)
	}
}

// some debugging utils
//func ids(r *[]graph.Node) []string {
//	s := []string{}
//	for _, r := range *r {
//		s = append(s, r.ID)
//	}
//	return s
//}

//func keys(m map[string]*graph.Node) []string {
//	s := []string{}
//	for k := range m {
//		s = append(s, k)
//	}
//	return s
//}
