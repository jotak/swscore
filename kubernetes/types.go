package kubernetes

import (
	apps_v1 "k8s.io/api/apps/v1"
	autoscaling_v1 "k8s.io/api/autoscaling/v1"
	core_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	// Kubernetes Controllers
	CronJobType               = "CronJob"
	DeploymentType            = "Deployment"
	DeploymentConfigType      = "DeploymentConfig"
	JobType                   = "Job"
	PodType                   = "Pod"
	ReplicationControllerType = "ReplicationController"
	ReplicaSetType            = "ReplicaSet"
	ServiceType               = "Service"
	StatefulSetType           = "StatefulSet"

	// Networking

	DestinationRules        = "destinationrules"
	DestinationRuleType     = "DestinationRule"
	DestinationRuleTypeList = "DestinationRuleList"

	Gateways        = "gateways"
	GatewayType     = "Gateway"
	GatewayTypeList = "GatewayList"

	EnvoyFilters        = "envoyfilters"
	EnvoyFilterType     = "EnvoyFilter"
	EnvoyFilterTypeList = "EnvoyFilterList"

	Sidecars        = "sidecars"
	SidecarType     = "Sidecar"
	SidecarTypeList = "SidecarList"

	ServiceEntries       = "serviceentries"
	ServiceEntryType     = "ServiceEntry"
	ServiceentryTypeList = "ServiceEntryList"

	VirtualServices        = "virtualservices"
	VirtualServiceType     = "VirtualService"
	VirtualServiceTypeList = "VirtualServiceList"

	WorkloadEntries       = "workloadentries"
	WorkloadEntryType     = "WorkloadEntry"
	WorkloadEntryTypeList = "WorkloadEntryList"

	// Quotas

	QuotaSpecs        = "quotaspecs"
	QuotaSpecType     = "QuotaSpec"
	QuotaSpecTypeList = "QuotaSpecList"

	QuotaSpecBindings        = "quotaspecbindings"
	QuotaSpecBindingType     = "QuotaSpecBinding"
	QuotaSpecBindingTypeList = "QuotaSpecBindingList"

	// PeerAuthentications

	Policies       = "policies"
	PolicyType     = "Policy"
	PolicyTypeList = "PolicyList"

	//MeshPeerAuthentications

	MeshPolicies       = "meshpolicies"
	MeshPolicyType     = "MeshPolicy"
	MeshPolicyTypeList = "MeshPolicyList"

	// ServiceMeshPolicies

	ServiceMeshPolicies       = "servicemeshpolicies"
	ServiceMeshPolicyType     = "ServiceMeshPolicy"
	ServiceMeshPolicyTypeList = "ServiceMeshPolicyList"

	// Rbac
	ClusterRbacConfigs        = "clusterrbacconfigs"
	ClusterRbacConfigType     = "ClusterRbacConfig"
	ClusterRbacConfigTypeList = "ClusterRbacConfigList"

	RbacConfigs        = "rbacconfigs"
	RbacConfigType     = "RbacConfig"
	RbacConfigTypeList = "RbacConfigList"

	ServiceRoles        = "serviceroles"
	ServiceRoleType     = "ServiceRole"
	ServiceRoleTypeList = "ServiceRoleList"

	ServiceRoleBindings        = "servicerolebindings"
	ServiceRoleBindingType     = "ServiceRoleBinding"
	ServiceRoleBindingTypeList = "ServiceRoleBindingList"

	ServiceMeshRbacConfigs        = "servicemeshrbacconfigs"
	ServiceMeshRbacConfigType     = "ServiceMeshRbacConfig"
	ServiceMeshRbacConfigTypeList = "ServiceMeshRbacConfigList"

	// Authorization PeerAuthentications
	AuthorizationPolicies         = "authorizationpolicies"
	AuthorizationPoliciesType     = "AuthorizationPolicy"
	AuthorizationPoliciesTypeList = "AuthorizationPolicyList"

	// Peer Authentications
	PeerAuthentications         = "peerauthentications"
	PeerAuthenticationsType     = "PeerAuthentication"
	PeerAuthenticationsTypeList = "PeerAuthenticationList"

	// Request Authentications
	RequestAuthentications         = "requestauthentications"
	RequestAuthenticationsType     = "RequestAuthentication"
	RequestAuthenticationsTypeList = "RequestAuthenticationList"

	// AttributeManifest
	AttributeManifests        = "attributemanifests"
	AttributeManifestType     = "attributemanifest"
	AttributeManifestTypeList = "attributemanifestList"

	// HttpApiSpecBinding
	HttpApiSpecBindings        = "httpapispecbindings"
	HttpApiSpecBindingType     = "HTTPAPISpecBinding"
	HttpApiSpecBindingTypeList = "HTTPAPISpecBindingList"

	// HttpApiSpec
	HttpApiSpecs        = "httpapispecs"
	HttpApiSpecType     = "HTTPAPISpec"
	HttpApiSpecTypeList = "HTTPAPISpecList"

	// Config - Rules

	Rules        = "rules"
	RuleType     = "rule"
	RuleTypeList = "ruleList"

	// Config - Adapters

	Adapters        = "adapters"
	AdapterType     = "adapter"
	AdapterTypeList = "adapterList"

	Handlers        = "handlers"
	HandlerType     = "handler"
	HandlerTypeList = "handlerList"

	// Config - Templates

	Instances        = "instances"
	InstanceType     = "instance"
	InstanceTypeList = "instanceList"

	Templates        = "templates"
	TemplateType     = "template"
	TemplateTypeList = "templateList"

	// Iter8 types

	Iter8Experiments        = "experiments"
	Iter8ExperimentType     = "Experiment"
	Iter8ExperimentTypeList = "ExperimentList"
	Iter8ConfigMap          = "iter8config-metrics"

	// Kiali types

	GraphAdapters        = "graphadapters"
	GraphAdapterType     = "GraphAdapter"
	GraphAdapterTypeList = "GraphAdapterList"
)

var (
	ConfigGroupVersion = schema.GroupVersion{
		Group:   "config.istio.io",
		Version: "v1alpha2",
	}
	ApiConfigVersion = ConfigGroupVersion.Group + "/" + ConfigGroupVersion.Version

	NetworkingGroupVersion = schema.GroupVersion{
		Group:   "networking.istio.io",
		Version: "v1alpha3",
	}
	ApiNetworkingVersion = NetworkingGroupVersion.Group + "/" + NetworkingGroupVersion.Version

	AuthenticationGroupVersion = schema.GroupVersion{
		Group:   "authentication.istio.io",
		Version: "v1alpha1",
	}
	ApiAuthenticationVersion = AuthenticationGroupVersion.Group + "/" + AuthenticationGroupVersion.Version

	RbacGroupVersion = schema.GroupVersion{
		Group:   "rbac.istio.io",
		Version: "v1alpha1",
	}
	ApiRbacVersion = RbacGroupVersion.Group + "/" + RbacGroupVersion.Version

	MaistraAuthenticationGroupVersion = schema.GroupVersion{
		Group:   "authentication.maistra.io",
		Version: "v1",
	}
	ApiMaistraAuthenticationVersion = MaistraAuthenticationGroupVersion.Group + "/" + MaistraAuthenticationGroupVersion.Version

	MaistraRbacGroupVersion = schema.GroupVersion{
		Group:   "rbac.maistra.io",
		Version: "v1",
	}
	ApiMaistraRbacVersion = MaistraRbacGroupVersion.Group + "/" + MaistraRbacGroupVersion.Version

	SecurityGroupVersion = schema.GroupVersion{
		Group:   "security.istio.io",
		Version: "v1beta1",
	}
	ApiSecurityVersion = SecurityGroupVersion.Group + "/" + SecurityGroupVersion.Version

	// We will add a new extesion API in a similar way as we added the Kubernetes + Istio APIs
	Iter8GroupVersion = schema.GroupVersion{
		Group:   "iter8.tools",
		Version: "v1alpha1",
	}
	ApiIter8Version = Iter8GroupVersion.Group + "/" + Iter8GroupVersion.Version

	// We will add a new extesion API in a similar way as we added the Kubernetes + Istio APIs
	KialiGroupVersion = schema.GroupVersion{
		Group:   "monitoring.kiali.io",
		Version: "v1alpha1",
	}
	ApiKialiVersion = KialiGroupVersion.Group + "/" + KialiGroupVersion.Version

	networkingTypes = []struct {
		objectKind     string
		collectionKind string
	}{
		{
			objectKind:     GatewayType,
			collectionKind: GatewayTypeList,
		},
		{
			objectKind:     VirtualServiceType,
			collectionKind: VirtualServiceTypeList,
		},
		{
			objectKind:     DestinationRuleType,
			collectionKind: DestinationRuleTypeList,
		},
		{
			objectKind:     ServiceEntryType,
			collectionKind: ServiceentryTypeList,
		},
		{
			objectKind:     SidecarType,
			collectionKind: SidecarTypeList,
		},
		{
			objectKind:     WorkloadEntryType,
			collectionKind: WorkloadEntryTypeList,
		},
		{
			objectKind:     EnvoyFilterType,
			collectionKind: EnvoyFilterTypeList,
		},
	}

	configTypes = []struct {
		objectKind     string
		collectionKind string
	}{
		{
			objectKind:     RuleType,
			collectionKind: RuleTypeList,
		},
		// Quota specs depends on Quota template but are not a "template" object itselft
		{
			objectKind:     QuotaSpecType,
			collectionKind: QuotaSpecTypeList,
		},
		{
			objectKind:     QuotaSpecBindingType,
			collectionKind: QuotaSpecBindingTypeList,
		},
		{
			objectKind:     AttributeManifestType,
			collectionKind: AttributeManifestTypeList,
		},
		{
			objectKind:     HttpApiSpecBindingType,
			collectionKind: HttpApiSpecBindingTypeList,
		},
		{
			objectKind:     HttpApiSpecType,
			collectionKind: HttpApiSpecTypeList,
		},
	}

	authenticationTypes = []struct {
		objectKind     string
		collectionKind string
	}{
		{
			objectKind:     PolicyType,
			collectionKind: PolicyTypeList,
		},
		{
			objectKind:     MeshPolicyType,
			collectionKind: MeshPolicyTypeList,
		},
	}

	maistraAuthenticationTypes = []struct {
		objectKind     string
		collectionKind string
	}{
		{
			objectKind:     ServiceMeshPolicyType,
			collectionKind: ServiceMeshPolicyTypeList,
		},
	}

	securityTypes = []struct {
		objectKind     string
		collectionKind string
	}{
		{
			objectKind:     PeerAuthenticationsType,
			collectionKind: PeerAuthenticationsTypeList,
		},
		{
			objectKind:     AuthorizationPoliciesType,
			collectionKind: AuthorizationPoliciesTypeList,
		},
		{
			objectKind:     RequestAuthenticationsType,
			collectionKind: RequestAuthenticationsTypeList,
		},
	}

	// TODO Adapters and Templates can be loaded from external config for easy maintenance

	adapterTypes = []struct {
		objectKind     string
		collectionKind string
	}{
		{
			objectKind:     AdapterType,
			collectionKind: AdapterTypeList,
		},
		{
			objectKind:     HandlerType,
			collectionKind: HandlerTypeList,
		},
	}

	templateTypes = []struct {
		objectKind     string
		collectionKind string
	}{
		{
			objectKind:     InstanceType,
			collectionKind: InstanceTypeList,
		},
		{
			objectKind:     TemplateType,
			collectionKind: TemplateTypeList,
		},
	}

	rbacTypes = []struct {
		objectKind     string
		collectionKind string
	}{
		{
			objectKind:     ClusterRbacConfigType,
			collectionKind: ClusterRbacConfigTypeList,
		},
		{
			objectKind:     RbacConfigType,
			collectionKind: RbacConfigTypeList,
		},
		{
			objectKind:     ServiceRoleType,
			collectionKind: ServiceRoleTypeList,
		},
		{
			objectKind:     ServiceRoleBindingType,
			collectionKind: ServiceRoleBindingTypeList,
		},
	}

	maistraRbacTypes = []struct {
		objectKind     string
		collectionKind string
	}{
		{
			objectKind:     ServiceMeshRbacConfigType,
			collectionKind: ServiceMeshRbacConfigTypeList,
		},
	}

	iter8Types = []struct {
		objectKind     string
		collectionKind string
	}{
		{
			objectKind:     Iter8ExperimentType,
			collectionKind: Iter8ExperimentTypeList,
		},
	}

	graphAdapterTypes = []struct {
		objectKind     string
		collectionKind string
	}{
		{
			objectKind:     GraphAdapterType,
			collectionKind: GraphAdapterTypeList,
		},
	}

	// A map to get the plural for a Istio type using the singlar type
	PluralType = map[string]string{
		// Networking
		Gateways:         GatewayType,
		VirtualServices:  VirtualServiceType,
		DestinationRules: DestinationRuleType,
		ServiceEntries:   ServiceEntryType,
		Sidecars:         SidecarType,
		WorkloadEntries:  WorkloadEntryType,
		EnvoyFilters:     EnvoyFilterType,

		// Main Config files
		Rules:               RuleType,
		QuotaSpecs:          QuotaSpecType,
		QuotaSpecBindings:   QuotaSpecBindingType,
		AttributeManifests:  AttributeManifestType,
		HttpApiSpecBindings: HttpApiSpecBindingType,
		HttpApiSpecs:        HttpApiSpecType,
		Adapters:            AdapterType,
		Handlers:            HandlerType,
		Instances:           InstanceType,
		Templates:           TemplateType,

		// PeerAuthentications
		Policies:            PolicyType,
		MeshPolicies:        MeshPolicyType,
		ServiceMeshPolicies: ServiceMeshPolicyType,

		// Rbac
		ClusterRbacConfigs:     ClusterRbacConfigType,
		RbacConfigs:            RbacConfigType,
		ServiceRoles:           ServiceRoleType,
		ServiceRoleBindings:    ServiceRoleBindingType,
		ServiceMeshRbacConfigs: ServiceMeshRbacConfigType,

		// Security
		AuthorizationPolicies:  AuthorizationPoliciesType,
		PeerAuthentications:    PeerAuthenticationsType,
		RequestAuthentications: RequestAuthenticationsType,

		// Iter8
		Iter8Experiments: Iter8ExperimentType,

		GraphAdapters: GraphAdapterType,
	}

	ResourceTypesToAPI = map[string]string{
		DestinationRules:       NetworkingGroupVersion.Group,
		VirtualServices:        NetworkingGroupVersion.Group,
		ServiceEntries:         NetworkingGroupVersion.Group,
		Gateways:               NetworkingGroupVersion.Group,
		Sidecars:               NetworkingGroupVersion.Group,
		WorkloadEntries:        NetworkingGroupVersion.Group,
		EnvoyFilters:           NetworkingGroupVersion.Group,
		Adapters:               ConfigGroupVersion.Group,
		Templates:              ConfigGroupVersion.Group,
		Rules:                  ConfigGroupVersion.Group,
		Handlers:               ConfigGroupVersion.Group,
		Instances:              ConfigGroupVersion.Group,
		QuotaSpecs:             ConfigGroupVersion.Group,
		QuotaSpecBindings:      ConfigGroupVersion.Group,
		AttributeManifests:     ConfigGroupVersion.Group,
		HttpApiSpecBindings:    ConfigGroupVersion.Group,
		HttpApiSpecs:           ConfigGroupVersion.Group,
		Policies:               AuthenticationGroupVersion.Group,
		MeshPolicies:           AuthenticationGroupVersion.Group,
		ClusterRbacConfigs:     RbacGroupVersion.Group,
		RbacConfigs:            RbacGroupVersion.Group,
		ServiceRoles:           RbacGroupVersion.Group,
		ServiceRoleBindings:    RbacGroupVersion.Group,
		ServiceMeshPolicies:    MaistraAuthenticationGroupVersion.Group,
		ServiceMeshRbacConfigs: MaistraRbacGroupVersion.Group,
		AuthorizationPolicies:  SecurityGroupVersion.Group,
		PeerAuthentications:    SecurityGroupVersion.Group,
		RequestAuthentications: SecurityGroupVersion.Group,
		// Extensions
		Iter8Experiments: Iter8GroupVersion.Group,
		GraphAdapters:    KialiGroupVersion.Group,
	}

	ApiToVersion = map[string]string{
		NetworkingGroupVersion.Group:            ApiNetworkingVersion,
		ConfigGroupVersion.Group:                ApiConfigVersion,
		AuthenticationGroupVersion.Group:        ApiAuthenticationVersion,
		RbacGroupVersion.Group:                  ApiRbacVersion,
		MaistraAuthenticationGroupVersion.Group: ApiMaistraAuthenticationVersion,
		MaistraRbacGroupVersion.Group:           ApiMaistraRbacVersion,
		SecurityGroupVersion.Group:              ApiSecurityVersion,
		KialiGroupVersion.Group:                 ApiKialiVersion,
	}
)

// IstioObject is a k8s wrapper interface for config objects.
// Taken from istio.io
type IstioObject interface {
	runtime.Object
	GetSpec() map[string]interface{}
	SetSpec(map[string]interface{})
	GetTypeMeta() meta_v1.TypeMeta
	SetTypeMeta(meta_v1.TypeMeta)
	GetObjectMeta() meta_v1.ObjectMeta
	SetObjectMeta(meta_v1.ObjectMeta)
	DeepCopyIstioObject() IstioObject
	HasWorkloadSelectorLabels() bool
	HasMatchLabelsSelector() bool
}

// IstioObjectList is a k8s wrapper interface for list config objects.
// Taken from istio.io
type IstioObjectList interface {
	runtime.Object
	GetItems() []IstioObject
}

type IstioMeshConfig struct {
	DisableMixerHttpReports bool  `yaml:"disableMixerHttpReports,omitempty"`
	EnableAutoMtls          *bool `yaml:"enableAutoMtls,omitempty"`
}

// ServiceList holds list of services, pods and deployments
type ServiceList struct {
	Services    *core_v1.ServiceList
	Pods        *core_v1.PodList
	Deployments *apps_v1.DeploymentList
}

// ServiceDetails is a wrapper to group full Service description, Endpoints and Pods.
// Used to fetch all details in a single operation instead to invoke individual APIs per each group.
type ServiceDetails struct {
	Service     *core_v1.Service                            `json:"service"`
	Endpoints   *core_v1.Endpoints                          `json:"endpoints"`
	Deployments *apps_v1.DeploymentList                     `json:"deployments"`
	Autoscalers *autoscaling_v1.HorizontalPodAutoscalerList `json:"autoscalers"`
	Pods        []core_v1.Pod                               `json:"pods"`
}

// IstioDetails is a wrapper to group all Istio objects related to a Service.
// Used to fetch all Istio information in a single operation instead to invoke individual APIs per each group.
type IstioDetails struct {
	VirtualServices  []IstioObject `json:"virtualservices"`
	DestinationRules []IstioObject `json:"destinationrules"`
	ServiceEntries   []IstioObject `json:"serviceentries"`
	Gateways         []IstioObject `json:"gateways"`
	Sidecars         []IstioObject `json:"sidecars"`
}

// MTLSDetails is a wrapper to group all Istio objects related to non-local mTLS configurations
type MTLSDetails struct {
	DestinationRules        []IstioObject `json:"destinationrules"`
	MeshPeerAuthentications []IstioObject `json:"meshpeerauthentications"`
	ServiceMeshPolicies     []IstioObject `json:"servicemeshpolicies"`
	PeerAuthentications     []IstioObject `json:"peerauthentications"`
	EnabledAutoMtls         bool          `json:"enabledautomtls"`
}

// RBACDetails is a wrapper for objects related to Istio RBAC (Role Based Access Control)
type RBACDetails struct {
	ClusterRbacConfigs     []IstioObject `json:"clusterrbacconfigs"`
	ServiceMeshRbacConfigs []IstioObject `json:"servicemeshrbacconfigs"`
	ServiceRoles           []IstioObject `json:"serviceroles"`
	ServiceRoleBindings    []IstioObject `json:"servicerolebindings"`
	AuthorizationPolicies  []IstioObject `json:"authorizationpolicies"`
}

// GenericIstioObject is a type to test Istio types defined by Istio as a Kubernetes extension.
type GenericIstioObject struct {
	meta_v1.TypeMeta   `json:",inline" yaml:",inline"`
	meta_v1.ObjectMeta `json:"metadata" yaml:"metadata"`
	Spec               map[string]interface{} `json:"spec"`
}

// GenericIstioObjectList is the generic Kubernetes API list wrapper
type GenericIstioObjectList struct {
	meta_v1.TypeMeta `json:",inline"`
	meta_v1.ListMeta `json:"metadata"`
	Items            []GenericIstioObject `json:"items"`
}

// GetSpec from a wrapper
func (in *GenericIstioObject) GetSpec() map[string]interface{} {
	return in.Spec
}

// SetSpec for a wrapper
func (in *GenericIstioObject) SetSpec(spec map[string]interface{}) {
	in.Spec = spec
}

// GetTypeMeta from a wrapper
func (in *GenericIstioObject) GetTypeMeta() meta_v1.TypeMeta {
	return in.TypeMeta
}

// SetObjectMeta for a wrapper
func (in *GenericIstioObject) SetTypeMeta(typemeta meta_v1.TypeMeta) {
	in.TypeMeta = typemeta
}

// GetObjectMeta from a wrapper
func (in *GenericIstioObject) GetObjectMeta() meta_v1.ObjectMeta {
	return in.ObjectMeta
}

// SetObjectMeta for a wrapper
func (in *GenericIstioObject) SetObjectMeta(metadata meta_v1.ObjectMeta) {
	in.ObjectMeta = metadata
}

func (in *GenericIstioObject) HasWorkloadSelectorLabels() bool {
	hwsl := false

	if ws, found := in.GetSpec()["workloadSelector"]; found {
		if wsCasted, ok := ws.(map[string]interface{}); ok {
			if _, found := wsCasted["labels"]; found {
				hwsl = true
			}
		}
	}

	return hwsl
}

func (in *GenericIstioObject) HasMatchLabelsSelector() bool {
	hwsl := false

	if s, found := in.GetSpec()["selector"]; found {
		if sCasted, ok := s.(map[string]interface{}); ok {
			if ml, found := sCasted["matchLabels"]; found {
				if mlCasted, ok := ml.(map[string]interface{}); ok {
					if len(mlCasted) > 0 {
						hwsl = true
					}
				}
			}
		}
	}

	return hwsl
}

// GetItems from a wrapper
func (in *GenericIstioObjectList) GetItems() []IstioObject {
	out := make([]IstioObject, len(in.Items))
	for i := range in.Items {
		out[i] = &in.Items[i]
	}
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GenericIstioObject) DeepCopyInto(out *GenericIstioObject) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GenericIstioObject.
func (in *GenericIstioObject) DeepCopy() *GenericIstioObject {
	if in == nil {
		return nil
	}
	out := new(GenericIstioObject)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *GenericIstioObject) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyIstioObject is an autogenerated deepcopy function, copying the receiver, creating a new IstioObject.
func (in *GenericIstioObject) DeepCopyIstioObject() IstioObject {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GenericIstioObjectList) DeepCopyInto(out *GenericIstioObjectList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]GenericIstioObject, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GenericIstioObjectList.
func (in *GenericIstioObjectList) DeepCopy() *GenericIstioObjectList {
	if in == nil {
		return nil
	}
	out := new(GenericIstioObjectList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *GenericIstioObjectList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

func (imc IstioMeshConfig) GetEnableAutoMtls() bool {
	if imc.EnableAutoMtls == nil {
		return true
	}
	return *imc.EnableAutoMtls
}
