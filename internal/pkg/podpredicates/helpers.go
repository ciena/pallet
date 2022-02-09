package podpredicates

import (
	"fmt"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/component-base/featuregate"
	"strings"
)

const (
	// Every feature gate should add method here following this template:
	//
	// // owner: @username
	// // alpha: v1.X
	// MyFeature featuregate.Feature = "MyFeature"

	// owner: @tallclair
	// beta: v1.4
	AppArmor featuregate.Feature = "AppArmor"

	// owner: @mtaufen
	// alpha: v1.4
	// beta: v1.11
	DynamicKubeletConfig featuregate.Feature = "DynamicKubeletConfig"

	// owner: @pweil-
	// alpha: v1.5
	//
	// Default userns=host for containers that are using other host namespaces, host mounts, the pod
	// contains a privileged container, or specific non-namespaced capabilities (MKNOD, SYS_MODULE,
	// SYS_TIME). This should only be enabled if user namespace remapping is enabled in the docker daemon.
	ExperimentalHostUserNamespaceDefaultingGate featuregate.Feature = "ExperimentalHostUserNamespaceDefaulting"

	// owner: @jiayingz
	// beta: v1.10
	//
	// Enables support for Device Plugins
	DevicePlugins featuregate.Feature = "DevicePlugins"

	// owner: @dxist
	// alpha: v1.16
	//
	// Enables support of HPA scaling to zero pods when an object or custom metric is configured.
	HPAScaleToZero featuregate.Feature = "HPAScaleToZero"

	// owner: @Huang-Wei
	// beta: v1.13
	// ga: v1.18
	//
	// Changes the logic behind evicting Pods from not ready Nodes
	// to take advantage of NoExecute Taints and Tolerations.
	TaintBasedEvictions featuregate.Feature = "TaintBasedEvictions"

	// owner: @mikedanese
	// alpha: v1.7
	// beta: v1.12
	//
	// Gets a server certificate for the kubelet from the Certificate Signing
	// Request API instead of generating one self signed and auto rotates the
	// certificate as expiration approaches.
	RotateKubeletServerCertificate featuregate.Feature = "RotateKubeletServerCertificate"

	// owner: @mikedanese
	// beta: v1.8
	//
	// Automatically renews the client certificate used for communicating with
	// the API server as the certificate approaches expiration.
	RotateKubeletClientCertificate featuregate.Feature = "RotateKubeletClientCertificate"

	// owner: @jinxu
	// beta: v1.10
	//
	// New local storage types to support local storage capacity isolation
	LocalStorageCapacityIsolation featuregate.Feature = "LocalStorageCapacityIsolation"

	// owner: @gnufied
	// beta: v1.11
	// Ability to Expand persistent volumes
	ExpandPersistentVolumes featuregate.Feature = "ExpandPersistentVolumes"

	// owner: @mlmhl
	// beta: v1.15
	// Ability to expand persistent volumes' file system without unmounting volumes.
	ExpandInUsePersistentVolumes featuregate.Feature = "ExpandInUsePersistentVolumes"

	// owner: @gnufied
	// alpha: v1.14
	// beta: v1.16
	// Ability to expand CSI volumes
	ExpandCSIVolumes featuregate.Feature = "ExpandCSIVolumes"

	// owner: @verb
	// alpha: v1.16
	//
	// Allows running an ephemeral container in pod namespaces to troubleshoot a running pod.
	EphemeralContainers featuregate.Feature = "EphemeralContainers"

	// owner: @verb
	// alpha: v1.10
	// beta: v1.12
	// GA: v1.17
	//
	// Allows all containers in a pod to share a process namespace.
	PodShareProcessNamespace featuregate.Feature = "PodShareProcessNamespace"

	// owner: @sjenning
	// alpha: v1.11
	//
	// Allows resource reservations at the QoS level preventing pods at lower QoS levels from
	// bursting into resources requested at higher QoS levels (memory only for now)
	QOSReserved featuregate.Feature = "QOSReserved"

	// owner: @ConnorDoyle
	// alpha: v1.8
	// beta: v1.10
	//
	// Alternative container-level CPU affinity policies.
	CPUManager featuregate.Feature = "CPUManager"

	// owner: @szuecs
	// alpha: v1.12
	//
	// Enable nodes to change CPUCFSQuotaPeriod
	CPUCFSQuotaPeriod featuregate.Feature = "CustomCPUCFSQuotaPeriod"

	// owner: @lmdaly
	// alpha: v1.16
	// beta: v1.18
	//
	// Enable resource managers to make NUMA aligned decisions
	TopologyManager featuregate.Feature = "TopologyManager"

	// owner: @sjenning
	// beta: v1.11
	//
	// Enable pods to set sysctls on a pod
	Sysctls featuregate.Feature = "Sysctls"

	// owner @smarterclayton
	// alpha: v1.16
	//
	// Enable legacy behavior to vary cluster functionality on the node-role.kubernetes.io labels. On by default (legacy), will be turned off in 1.18.
	LegacyNodeRoleBehavior featuregate.Feature = "LegacyNodeRoleBehavior"

	// owner @brendandburns
	// alpha: v1.9
	//
	// Enable nodes to exclude themselves from service load balancers
	ServiceNodeExclusion featuregate.Feature = "ServiceNodeExclusion"

	// owner @smarterclayton
	// alpha: v1.16
	//
	// Enable nodes to exclude themselves from network disruption checks
	NodeDisruptionExclusion featuregate.Feature = "NodeDisruptionExclusion"

	// owner: @saad-ali
	// alpha: v1.12
	// beta:  v1.14
	// GA:    v1.18
	// Enable all logic related to the CSIDriver API object in storage.k8s.io
	CSIDriverRegistry featuregate.Feature = "CSIDriverRegistry"

	// owner: @verult
	// alpha: v1.12
	// beta:  v1.14
	// ga:    v1.17
	// Enable all logic related to the CSINode API object in storage.k8s.io
	CSINodeInfo featuregate.Feature = "CSINodeInfo"

	// owner: @screeley44
	// alpha: v1.9
	// beta:  v1.13
	// ga: 	  v1.18
	//
	// Enable Block volume support in containers.
	BlockVolume featuregate.Feature = "BlockVolume"

	// owner: @pospispa
	// GA: v1.11
	//
	// Postpone deletion of a PV or a PVC when they are being used
	StorageObjectInUseProtection featuregate.Feature = "StorageObjectInUseProtection"

	// owner: @aveshagarwal
	// alpha: v1.9
	//
	// Enable resource limits priority function
	ResourceLimitsPriorityFunction featuregate.Feature = "ResourceLimitsPriorityFunction"

	// owner: @m1093782566
	// GA: v1.11
	//
	// Implement IPVS-based in-cluster service load balancing
	SupportIPVSProxyMode featuregate.Feature = "SupportIPVSProxyMode"

	// owner: @dims, @derekwaynecarr
	// alpha: v1.10
	// beta: v1.14
	//
	// Implement support for limiting pids in pods
	SupportPodPidsLimit featuregate.Feature = "SupportPodPidsLimit"

	// owner: @feiskyer
	// alpha: v1.10
	//
	// Enable Hyper-V containers on Windows
	HyperVContainer featuregate.Feature = "HyperVContainer"

	// owner: @mikedanese
	// beta: v1.12
	//
	// Implement TokenRequest endpoint on service account resources.
	TokenRequest featuregate.Feature = "TokenRequest"

	// owner: @mikedanese
	// beta: v1.12
	//
	// Enable ServiceAccountTokenVolumeProjection support in ProjectedVolumes.
	TokenRequestProjection featuregate.Feature = "TokenRequestProjection"

	// owner: @mikedanese
	// alpha: v1.13
	//
	// Migrate ServiceAccount volumes to use a projected volume consisting of a
	// ServiceAccountTokenVolumeProjection. This feature adds new required flags
	// to the API server.
	BoundServiceAccountTokenVolume featuregate.Feature = "BoundServiceAccountTokenVolume"

	// owner: @mtaufen
	// alpha: v1.18
	//
	// Enable OIDC discovery endpoints (issuer and JWKS URLs) for the service
	// account issuer in the API server.
	// Note these endpoints serve minimally-compliant discovery docs that are
	// intended to be used for service account token verification.
	ServiceAccountIssuerDiscovery featuregate.Feature = "ServiceAccountIssuerDiscovery"

	// owner: @Random-Liu
	// beta: v1.11
	//
	// Enable container log rotation for cri container runtime
	CRIContainerLogRotation featuregate.Feature = "CRIContainerLogRotation"

	// owner: @krmayankk
	// beta: v1.14
	//
	// Enables control over the primary group ID of containers' init processes.
	RunAsGroup featuregate.Feature = "RunAsGroup"

	// owner: @saad-ali
	// ga
	//
	// Allow mounting a subpath of a volume in a container
	// Do not remove this feature gate even though it's GA
	VolumeSubpath featuregate.Feature = "VolumeSubpath"

	// owner: @gnufied
	// beta : v1.12
	// GA   : v1.17

	//
	// Add support for volume plugins to report node specific
	// volume limits
	AttachVolumeLimit featuregate.Feature = "AttachVolumeLimit"

	// owner: @ravig
	// alpha: v1.11
	//
	// Include volume count on node to be considered for balanced resource allocation while scheduling.
	// A node which has closer cpu,memory utilization and volume count is favoured by scheduler
	// while making decisions.
	BalanceAttachedNodeVolumes featuregate.Feature = "BalanceAttachedNodeVolumes"

	// owner: @kevtaylor
	// alpha: v1.14
	// beta: v1.15
	// ga: v1.17
	//
	// Allow subpath environment variable substitution
	// Only applicable if the VolumeSubpath feature is also enabled
	VolumeSubpathEnvExpansion featuregate.Feature = "VolumeSubpathEnvExpansion"

	// owner: @vladimirvivien
	// alpha: v1.11
	// beta:  v1.14
	// ga: 	  v1.18
	//
	// Enables CSI to use raw block storage volumes
	CSIBlockVolume featuregate.Feature = "CSIBlockVolume"

	// owner: @vladimirvivien
	// alpha: v1.14
	// beta: v1.16
	//
	// Enables CSI Inline volumes support for pods
	CSIInlineVolume featuregate.Feature = "CSIInlineVolume"

	// owner: @tallclair
	// alpha: v1.12
	// beta:  v1.14
	//
	// Enables RuntimeClass, for selecting between multiple runtimes to run a pod.
	RuntimeClass featuregate.Feature = "RuntimeClass"

	// owner: @mtaufen
	// alpha: v1.12
	// beta:  v1.14
	// GA: v1.17
	//
	// Kubelet uses the new Lease API to report node heartbeats,
	// (Kube) Node Lifecycle Controller uses these heartbeats as a node health signal.
	NodeLease featuregate.Feature = "NodeLease"

	// owner: @janosi
	// alpha: v1.12
	//
	// Enables SCTP as new protocol for Service ports, NetworkPolicy, and ContainerPort in Pod/Containers definition
	SCTPSupport featuregate.Feature = "SCTPSupport"

	// owner: @xing-yang
	// alpha: v1.12
	// beta: v1.17
	//
	// Enable volume snapshot data source support.
	VolumeSnapshotDataSource featuregate.Feature = "VolumeSnapshotDataSource"

	// owner: @jessfraz
	// alpha: v1.12
	//
	// Enables control over ProcMountType for containers.
	ProcMountType featuregate.Feature = "ProcMountType"

	// owner: @janetkuo
	// alpha: v1.12
	//
	// Allow TTL controller to clean up Pods and Jobs after they finish.
	TTLAfterFinished featuregate.Feature = "TTLAfterFinished"

	// owner: @dashpole
	// alpha: v1.13
	// beta: v1.15
	//
	// Enables the kubelet's pod resources grpc endpoint
	KubeletPodResources featuregate.Feature = "KubeletPodResources"

	// owner: @davidz627
	// alpha: v1.14
	// beta: v1.17
	//
	// Enables the in-tree storage to CSI Plugin migration feature.
	CSIMigration featuregate.Feature = "CSIMigration"

	// owner: @davidz627
	// alpha: v1.14
	// beta: v1.17
	//
	// Enables the GCE PD in-tree driver to GCE CSI Driver migration feature.
	CSIMigrationGCE featuregate.Feature = "CSIMigrationGCE"

	// owner: @davidz627
	// alpha: v1.17
	//
	// Disables the GCE PD in-tree driver.
	// Expects GCE PD CSI Driver to be installed and configured on all nodes.
	CSIMigrationGCEComplete featuregate.Feature = "CSIMigrationGCEComplete"

	// owner: @leakingtapan
	// alpha: v1.14
	// beta: v1.17
	//
	// Enables the AWS EBS in-tree driver to AWS EBS CSI Driver migration feature.
	CSIMigrationAWS featuregate.Feature = "CSIMigrationAWS"

	// owner: @leakingtapan
	// alpha: v1.17
	//
	// Disables the AWS EBS in-tree driver.
	// Expects AWS EBS CSI Driver to be installed and configured on all nodes.
	CSIMigrationAWSComplete featuregate.Feature = "CSIMigrationAWSComplete"

	// owner: @andyzhangx
	// alpha: v1.15
	//
	// Enables the Azure Disk in-tree driver to Azure Disk Driver migration feature.
	CSIMigrationAzureDisk featuregate.Feature = "CSIMigrationAzureDisk"

	// owner: @andyzhangx
	// alpha: v1.17
	//
	// Disables the Azure Disk in-tree driver.
	// Expects Azure Disk CSI Driver to be installed and configured on all nodes.
	CSIMigrationAzureDiskComplete featuregate.Feature = "CSIMigrationAzureDiskComplete"

	// owner: @andyzhangx
	// alpha: v1.15
	//
	// Enables the Azure File in-tree driver to Azure File Driver migration feature.
	CSIMigrationAzureFile featuregate.Feature = "CSIMigrationAzureFile"

	// owner: @andyzhangx
	// alpha: v1.17
	//
	// Disables the Azure File in-tree driver.
	// Expects Azure File CSI Driver to be installed and configured on all nodes.
	CSIMigrationAzureFileComplete featuregate.Feature = "CSIMigrationAzureFileComplete"

	// owner: @gnufied
	// alpha: v1.18
	// Allows user to configure volume permission change policy for fsGroups when mounting
	// a volume in a Pod.
	ConfigurableFSGroupPolicy featuregate.Feature = "ConfigurableFSGroupPolicy"

	// owner: @RobertKrawitz
	// beta: v1.15
	//
	// Implement support for limiting pids in nodes
	SupportNodePidsLimit featuregate.Feature = "SupportNodePidsLimit"

	// owner: @wk8
	// alpha: v1.14
	// beta: v1.16
	//
	// Enables GMSA support for Windows workloads.
	WindowsGMSA featuregate.Feature = "WindowsGMSA"

	// owner: @bclau
	// alpha: v1.16
	// beta: v1.17
	// GA: v1.18
	//
	// Enables support for running container entrypoints as different usernames than their default ones.
	WindowsRunAsUserName featuregate.Feature = "WindowsRunAsUserName"

	// owner: @adisky
	// alpha: v1.14
	// beta: v1.18
	//
	// Enables the OpenStack Cinder in-tree driver to OpenStack Cinder CSI Driver migration feature.
	CSIMigrationOpenStack featuregate.Feature = "CSIMigrationOpenStack"

	// owner: @adisky
	// alpha: v1.17
	//
	// Disables the OpenStack Cinder in-tree driver.
	// Expects the OpenStack Cinder CSI Driver to be installed and configured on all nodes.
	CSIMigrationOpenStackComplete featuregate.Feature = "CSIMigrationOpenStackComplete"

	// owner: @MrHohn
	// alpha: v1.15
	// beta:  v1.16
	// GA: v1.17
	//
	// Enables Finalizer Protection for Service LoadBalancers.
	ServiceLoadBalancerFinalizer featuregate.Feature = "ServiceLoadBalancerFinalizer"

	// owner: @RobertKrawitz
	// alpha: v1.15
	//
	// Allow use of filesystems for ephemeral storage monitoring.
	// Only applies if LocalStorageCapacityIsolation is set.
	LocalStorageCapacityIsolationFSQuotaMonitoring featuregate.Feature = "LocalStorageCapacityIsolationFSQuotaMonitoring"

	// owner: @denkensk
	// alpha: v1.15
	//
	// Enables NonPreempting option for priorityClass and pod.
	NonPreemptingPriority featuregate.Feature = "NonPreemptingPriority"

	// owner: @j-griffith
	// alpha: v1.15
	// beta: v1.16
	// GA: v1.18
	//
	// Enable support for specifying an existing PVC as a DataSource
	VolumePVCDataSource featuregate.Feature = "VolumePVCDataSource"

	// owner: @egernst
	// alpha: v1.16
	// beta: v1.18
	//
	// Enables PodOverhead, for accounting pod overheads which are specific to a given RuntimeClass
	PodOverhead featuregate.Feature = "PodOverhead"

	// owner: @khenidak
	// alpha: v1.15
	//
	// Enables ipv6 dual stack
	IPv6DualStack featuregate.Feature = "IPv6DualStack"

	// owner: @robscott @freehan
	// alpha: v1.16
	//
	// Enable Endpoint Slices for more scalable Service endpoints.
	EndpointSlice featuregate.Feature = "EndpointSlice"

	// owner: @robscott @freehan
	// alpha: v1.18
	//
	// Enable Endpoint Slice consumption by kube-proxy for improved scalability.
	EndpointSliceProxying featuregate.Feature = "EndpointSliceProxying"

	// owner: @Huang-Wei
	// beta: v1.18
	//
	// Schedule pods evenly across available topology domains.
	EvenPodsSpread featuregate.Feature = "EvenPodsSpread"

	// owner: @matthyx
	// alpha: v1.16
	// beta: v1.18
	//
	// Enables the startupProbe in kubelet worker.
	StartupProbe featuregate.Feature = "StartupProbe"

	// owner: @deads2k
	// beta: v1.17
	//
	// Enables the users to skip TLS verification of kubelets on pod logs requests
	AllowInsecureBackendProxy featuregate.Feature = "AllowInsecureBackendProxy"

	// owner: @mortent
	// alpha: v1.3
	// beta:  v1.5
	//
	// Enable all logic related to the PodDisruptionBudget API object in policy
	PodDisruptionBudget featuregate.Feature = "PodDisruptionBudget"

	// owner: @m1093782566
	// alpha: v1.17
	//
	// Enables topology aware service routing
	ServiceTopology featuregate.Feature = "ServiceTopology"

	// owner: @robscott
	// alpha: v1.18
	//
	// Enables AppProtocol field for Services and Endpoints.
	ServiceAppProtocol featuregate.Feature = "ServiceAppProtocol"

	// owner: @wojtek-t
	// alpha: v1.18
	//
	// Enables a feature to make secrets and configmaps data immutable.
	ImmutableEphemeralVolumes featuregate.Feature = "ImmutableEphemeralVolumes"

	// owner: @robscott
	// beta: v1.18
	//
	// Enables DefaultIngressClass admission controller.
	DefaultIngressClass featuregate.Feature = "DefaultIngressClass"

	// owner: @bart0sh
	// alpha: v1.18
	//
	// Enables usage of HugePages-<size> in a volume medium,
	// e.g. emptyDir:
	//        medium: HugePages-1Gi
	HugePageStorageMediumSize featuregate.Feature = "HugePageStorageMediumSize"

	// owner: @freehan
	// GA: v1.18
	//
	// Enable ExternalTrafficPolicy for Service ExternalIPs.
	// This is for bug fix #69811
	ExternalPolicyForExternalIP featuregate.Feature = "ExternalPolicyForExternalIP"

	// owner: @bswartz
	// alpha: v1.18
	//
	// Enables usage of any object for volume data source in PVCs
	AnyVolumeDataSource featuregate.Feature = "AnyVolumeDataSource"
)

func IsExtendedResourceName(name v1.ResourceName) bool {
	if IsNativeResource(name) || strings.HasPrefix(string(name), v1.DefaultResourceRequestsPrefix) {
		return false
	}
	// Ensure it satisfies the rules in IsQualifiedName() after converted into quota resource name
	nameForQuota := fmt.Sprintf("%s%s", v1.DefaultResourceRequestsPrefix, string(name))
	if errs := validation.IsQualifiedName(string(nameForQuota)); len(errs) != 0 {
		return false
	}
	return true
}

func IsAttachableVolumeResourceName(name v1.ResourceName) bool {
	return strings.HasPrefix(string(name), v1.ResourceAttachableVolumesPrefix)
}

// Extended and Hugepages resources
func IsScalarResourceName(name v1.ResourceName) bool {
	return IsExtendedResourceName(name) || IsHugePageResourceName(name) ||
		IsPrefixedNativeResource(name) || IsAttachableVolumeResourceName(name)
}

// IsPrefixedNativeResource returns true if the resource name is in the
// *kubernetes.io/ namespace.
func IsPrefixedNativeResource(name v1.ResourceName) bool {
	return strings.Contains(string(name), v1.ResourceDefaultNamespacePrefix)
}

// IsNativeResource returns true if the resource name is in the
// *kubernetes.io/ namespace. Partially-qualified (unprefixed) names are
// implicitly in the kubernetes.io/ namespace.
func IsNativeResource(name v1.ResourceName) bool {
	return !strings.Contains(string(name), "/") ||
		IsPrefixedNativeResource(name)
}

// IsHugePageResourceName returns true if the resource name has the huge page
// resource prefix.
func IsHugePageResourceName(name v1.ResourceName) bool {
	return strings.HasPrefix(string(name), v1.ResourceHugePagesPrefix)
}

// HugePageResourceName returns a ResourceName with the canonical hugepage
// prefix prepended for the specified page size.  The page size is converted
// to its canonical representation.
func HugePageResourceName(pageSize resource.Quantity) v1.ResourceName {
	return v1.ResourceName(fmt.Sprintf("%s%s", v1.ResourceHugePagesPrefix, pageSize.String()))
}

// HugePageSizeFromResourceName returns the page size for the specified huge page
// resource name.  If the specified input is not a valid huge page resource name
// an error is returned.
func HugePageSizeFromResourceName(name v1.ResourceName) (resource.Quantity, error) {
	if !IsHugePageResourceName(name) {
		return resource.Quantity{}, fmt.Errorf("resource name: %s is an invalid hugepage name", name)
	}
	pageSize := strings.TrimPrefix(string(name), v1.ResourceHugePagesPrefix)
	return resource.ParseQuantity(pageSize)
}
