package kubetest

import "github.com/kiali/kiali/kubernetes"

func (o *K8SClientMock) GetGraphAdapter(namespace, name string) (*kubernetes.GraphAdapter, error) {
	args := o.Called(namespace, name)
	return args.Get(0).(*kubernetes.GraphAdapter), args.Error(1)
}

func (o *K8SClientMock) GetGraphAdapters(namespace string) ([]kubernetes.GraphAdapter, error) {
	args := o.Called(namespace)
	return args.Get(0).([]kubernetes.GraphAdapter), args.Error(1)
}
