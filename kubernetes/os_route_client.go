package kubernetes

import (
	"encoding/json"
	"errors"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
)

// OSRouteClient is the client struct for OpenShift Routes API over Kubernetes
// It hides the way it queries each API
type OSRouteClient struct {
	client *rest.RESTClient
}

// NewOSRouteClient creates a new client able to fetch OpenShift Routes API.
func NewOSRouteClient() (*OSRouteClient, error) {
	config, err := ConfigClient()
	if err != nil {
		return nil, err
	}

	types := runtime.NewScheme()
	schemeBuilder := runtime.NewSchemeBuilder(
		func(scheme *runtime.Scheme) error {
			return nil
		})

	err = schemeBuilder.AddToScheme(types)
	if err != nil {
		return nil, err
	}

	client, err := newClientForAPI(config, routeGroupVersion, types)
	return &OSRouteClient{
		client: client,
	}, err
}

// GetRoute returns an OpenShift route URL for the given name
func (in *OSRouteClient) GetRoute(namespace string, routeName string) (string, error) {
	result, err := in.client.Get().Namespace(namespace).Resource("routes").SubResource(routeName).Do().Raw()
	if err != nil {
		return "", err
	}
	var obj interface{}
	err = json.Unmarshal(result, &obj)
	if err != nil {
		return "", err
	}
	spec, ok := obj.(map[string]interface{})["spec"]
	if !ok {
		return "", errors.New("Missing spec in Route")
	}
	host, ok := spec.(map[string]interface{})["host"].(string)
	if !ok {
		return "", errors.New("Missing host in Route spec")
	}
	protocol := "http"
	tls, ok := spec.(map[string]interface{})["tls"]
	if ok {
		tlsTermination, ok := tls.(map[string]interface{})["termination"].(string)
		if ok && len(tlsTermination) > 0 {
			protocol = "https"
		}
	}
	return protocol + "://" + host, nil
}
