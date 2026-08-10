package kubeResources

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/gknguyen/litelens/internal/dto"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	listerscorev1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
)

// lister helpers

type errorPodLister struct{ err error }

func (e *errorPodLister) List(_ labels.Selector) ([]*corev1.Pod, error) {
	return nil, e.err
}
func (e *errorPodLister) Pods(_ string) listerscorev1.PodNamespaceLister {
	return &errorPodNamespaceLister{e.err}
}

type errorPodNamespaceLister struct{ err error }

func (e *errorPodNamespaceLister) List(_ labels.Selector) ([]*corev1.Pod, error) {
	return nil, e.err
}
func (e *errorPodNamespaceLister) Get(_ string) (*corev1.Pod, error) {
	return nil, e.err
}

func newPodLister(pods ...*corev1.Pod) listerscorev1.PodLister {
	indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
	for _, p := range pods {
		_ = indexer.Add(p)
	}
	return listerscorev1.NewPodLister(indexer)
}

func makePod(name, namespace string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			Namespace:         namespace,
			CreationTimestamp: metav1.Time{Time: fixedTime},
		},
		Spec:   corev1.PodSpec{},
		Status: corev1.PodStatus{},
	}
}

// TestListPods_EmptyNamespace_ReturnsAll verifies namespace="" lists across all namespaces.
func TestListPods_EmptyNamespace_ReturnsAll(t *testing.T) {
	p1 := makePod("pod-a", "ns-a")
	p2 := makePod("pod-b", "ns-b")
	result, err := ListPods(newPodLister(p1, p2), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 results, got %d", len(result))
	}
}

// TestListPods_SpecificNamespace_Filters verifies only pods in the given namespace are returned.
func TestListPods_SpecificNamespace_Filters(t *testing.T) {
	p1 := makePod("pod-a", "ns-a")
	p2 := makePod("pod-b", "ns-b")
	result, err := ListPods(newPodLister(p1, p2), "ns-a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}
	if result[0].Name != "pod-a" {
		t.Errorf("Name = %q; want %q", result[0].Name, "pod-a")
	}
}

// TestListPods_ErrorPropagation_GlobalScope verifies errors from the cluster-scope lister are returned.
func TestListPods_ErrorPropagation_GlobalScope(t *testing.T) {
	sentinel := errors.New("lister unavailable")
	_, err := ListPods(&errorPodLister{err: sentinel}, "")
	if !errors.Is(err, sentinel) {
		t.Errorf("expected sentinel error; got %v", err)
	}
}

// TestGetPodByName_HappyPath verifies a pod is fetched and mapped by name.
func TestGetPodByName_HappyPath(t *testing.T) {
	p := makePod("my-pod", "default")
	lister := newPodLister(p)
	got, err := GetPodByName(lister, "default", "my-pod")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "my-pod" {
		t.Errorf("Name = %q; want %q", got.Name, "my-pod")
	}
	if got.Namespace != "default" {
		t.Errorf("Namespace = %q; want %q", got.Namespace, "default")
	}
}

// TestGetPodByName_ErrorPath verifies a missing pod returns an error.
func TestGetPodByName_ErrorPath(t *testing.T) {
	sentinel := errors.New("not found")
	_, err := GetPodByName(&errorPodLister{err: sentinel}, "default", "missing")
	if !errors.Is(err, sentinel) {
		t.Errorf("expected sentinel error; got %v", err)
	}
}

// TestToPod_StatusTerminating verifies DeletionTimestamp overrides Phase.
func TestToPod_StatusTerminating(t *testing.T) {
	now := metav1.Now()
	p := makePod("p", "default")
	p.DeletionTimestamp = &now
	p.Status.Phase = corev1.PodRunning
	got := toPod(p)
	if got.Status != "Terminating" {
		t.Errorf("Status = %q; want %q", got.Status, "Terminating")
	}
}

// TestToPod_StatusFromPhase verifies Phase is used when DeletionTimestamp is nil.
func TestToPod_StatusFromPhase(t *testing.T) {
	p := makePod("p", "default")
	p.Status.Phase = corev1.PodPending
	got := toPod(p)
	if got.Status != "Pending" {
		t.Errorf("Status = %q; want %q", got.Status, "Pending")
	}
}

// TestToPod_ReadyCount verifies ready count reflects cs.Ready == true entries.
func TestToPod_ReadyCount(t *testing.T) {
	p := makePod("p", "default")
	p.Spec.Containers = []corev1.Container{
		{Name: "c1"}, {Name: "c2"}, {Name: "c3"},
	}
	p.Status.ContainerStatuses = []corev1.ContainerStatus{
		{Name: "c1", Ready: true, RestartCount: 1},
		{Name: "c2", Ready: false, RestartCount: 2},
		{Name: "c3", Ready: true, RestartCount: 0},
	}
	got := toPod(p)
	if got.Ready != "2/3" {
		t.Errorf("Ready = %q; want %q", got.Ready, "2/3")
	}
	if got.Restarts != 3 {
		t.Errorf("Restarts = %d; want 3", got.Restarts)
	}
}

// TestToPod_ControlledBy verifies OwnerReferences[0] sets ControlledBy and ControlledByName.
func TestToPod_ControlledBy(t *testing.T) {
	p := makePod("p", "default")
	p.OwnerReferences = []metav1.OwnerReference{
		{Kind: "ReplicaSet", Name: "my-rs"},
	}
	got := toPod(p)
	if got.ControlledBy != "ReplicaSet" {
		t.Errorf("ControlledBy = %q; want %q", got.ControlledBy, "ReplicaSet")
	}
	if got.ControlledByName != "my-rs" {
		t.Errorf("ControlledByName = %q; want %q", got.ControlledByName, "my-rs")
	}
}

// TestToPod_NoOwner verifies empty ControlledBy when no OwnerReferences.
func TestToPod_NoOwner(t *testing.T) {
	p := makePod("p", "default")
	got := toPod(p)
	if got.ControlledBy != "" {
		t.Errorf("ControlledBy = %q; want empty", got.ControlledBy)
	}
	if got.ControlledByName != "" {
		t.Errorf("ControlledByName = %q; want empty", got.ControlledByName)
	}
}

// TestToPod_ResourceAggregation verifies CPU/mem/disk req+lim are summed across containers.
func TestToPod_ResourceAggregation(t *testing.T) {
	p := makePod("p", "default")
	p.Spec.Containers = []corev1.Container{
		{
			Name: "c1",
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:              resource.MustParse("100m"),
					corev1.ResourceMemory:           resource.MustParse("128Mi"),
					corev1.ResourceEphemeralStorage: resource.MustParse("1Gi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:              resource.MustParse("200m"),
					corev1.ResourceMemory:           resource.MustParse("256Mi"),
					corev1.ResourceEphemeralStorage: resource.MustParse("2Gi"),
				},
			},
		},
		{
			Name: "c2",
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("50m"),
					corev1.ResourceMemory: resource.MustParse("64Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("128Mi"),
				},
			},
		},
	}
	got := toPod(p)
	if got.CPUReqMilli != 150 {
		t.Errorf("CPUReqMilli = %d; want 150", got.CPUReqMilli)
	}
	if got.CPULimMilli != 300 {
		t.Errorf("CPULimMilli = %d; want 300", got.CPULimMilli)
	}
	wantMemReq := int64(192 * 1024 * 1024)
	if got.MemReqBytes != wantMemReq {
		t.Errorf("MemReqBytes = %d; want %d", got.MemReqBytes, wantMemReq)
	}
	wantMemLim := int64(384 * 1024 * 1024)
	if got.MemLimBytes != wantMemLim {
		t.Errorf("MemLimBytes = %d; want %d", got.MemLimBytes, wantMemLim)
	}
	wantDiskReq := int64(1024 * 1024 * 1024)
	if got.DiskReqBytes != wantDiskReq {
		t.Errorf("DiskReqBytes = %d; want %d", got.DiskReqBytes, wantDiskReq)
	}
}

// TestToPod_IPFields verifies HostIPs and PodIPs are populated correctly.
func TestToPod_IPFields(t *testing.T) {
	p := makePod("p", "default")
	p.Status.HostIPs = []corev1.HostIP{{IP: "192.168.1.1"}, {IP: "192.168.1.2"}}
	p.Status.PodIPs = []corev1.PodIP{{IP: "10.0.0.1"}}
	got := toPod(p)
	if len(got.HostIPs) != 2 || got.HostIPs[0] != "192.168.1.1" {
		t.Errorf("HostIPs = %v; want [192.168.1.1 192.168.1.2]", got.HostIPs)
	}
	if len(got.PodIPs) != 1 || got.PodIPs[0] != "10.0.0.1" {
		t.Errorf("PodIPs = %v; want [10.0.0.1]", got.PodIPs)
	}
}

// TestToPod_IPFieldsFallback verifies legacy single-IP fields are used when slice is empty.
func TestToPod_IPFieldsFallback(t *testing.T) {
	p := makePod("p", "default")
	p.Status.HostIP = "10.1.1.1"
	p.Status.PodIP = "172.16.0.5"
	got := toPod(p)
	if len(got.HostIPs) != 1 || got.HostIPs[0] != "10.1.1.1" {
		t.Errorf("HostIPs fallback = %v; want [10.1.1.1]", got.HostIPs)
	}
	if len(got.PodIPs) != 1 || got.PodIPs[0] != "172.16.0.5" {
		t.Errorf("PodIPs fallback = %v; want [172.16.0.5]", got.PodIPs)
	}
}

// TestToPod_Tolerations verifies toleration count and detail mapping.
func TestToPod_Tolerations(t *testing.T) {
	secs := int64(30)
	p := makePod("p", "default")
	p.Spec.Tolerations = []corev1.Toleration{
		{Key: "node-role", Operator: corev1.TolerationOpEqual, Value: "master", Effect: corev1.TaintEffectNoSchedule},
		{Key: "spot", Operator: corev1.TolerationOpExists, TolerationSeconds: &secs},
	}
	got := toPod(p)
	if got.Tolerations != 2 {
		t.Errorf("Tolerations = %d; want 2", got.Tolerations)
	}
	if len(got.TolerationDetails) != 2 {
		t.Fatalf("TolerationDetails len = %d; want 2", len(got.TolerationDetails))
	}
	if got.TolerationDetails[0].Key != "node-role" {
		t.Errorf("TolerationDetails[0].Key = %q; want %q", got.TolerationDetails[0].Key, "node-role")
	}
	if got.TolerationDetails[1].Seconds != &secs && *got.TolerationDetails[1].Seconds != secs {
		t.Errorf("TolerationDetails[1].Seconds = %v; want %d", got.TolerationDetails[1].Seconds, secs)
	}
}

// TestToPod_AffinityCount verifies each non-nil affinity type increments the count.
func TestToPod_AffinityCount(t *testing.T) {
	p := makePod("p", "default")
	p.Spec.Affinity = &corev1.Affinity{
		NodeAffinity:    &corev1.NodeAffinity{},
		PodAntiAffinity: &corev1.PodAntiAffinity{},
	}
	got := toPod(p)
	if got.AffinityCount != 2 {
		t.Errorf("AffinityCount = %d; want 2", got.AffinityCount)
	}
}

// TestToPod_Conditions verifies conditions slice is mapped correctly.
func TestToPod_Conditions(t *testing.T) {
	p := makePod("p", "default")
	p.Status.Conditions = []corev1.PodCondition{
		{Type: corev1.PodReady, Status: corev1.ConditionTrue, Message: "ready"},
		{Type: corev1.PodScheduled, Status: corev1.ConditionFalse, Message: "unschedulable"},
	}
	got := toPod(p)
	if len(got.Conditions) != 2 {
		t.Fatalf("Conditions len = %d; want 2", len(got.Conditions))
	}
	if got.Conditions[0].Type != "Ready" || got.Conditions[0].Status != "True" {
		t.Errorf("Conditions[0] = %+v; want Type=Ready Status=True", got.Conditions[0])
	}
}

// TestToPod_ContainerDetails_StateStrings verifies state strings for running/waiting/terminated.
func TestToPod_ContainerDetails_StateStrings(t *testing.T) {
	p := makePod("p", "default")
	p.Spec.Containers = []corev1.Container{
		{Name: "c-running"},
		{Name: "c-waiting"},
		{Name: "c-terminated"},
		{Name: "c-unknown"},
	}
	p.Status.ContainerStatuses = []corev1.ContainerStatus{
		{Name: "c-running", State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}},
		{Name: "c-waiting", State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{}}},
		{Name: "c-terminated", State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{}}},
		// c-unknown has no state set
	}
	got := toPod(p)
	if len(got.ContainerDetails) != 4 {
		t.Fatalf("ContainerDetails len = %d; want 4", len(got.ContainerDetails))
	}
	stateMap := map[string]string{}
	for _, cd := range got.ContainerDetails {
		stateMap[cd.Name] = cd.Status
	}
	cases := map[string]string{
		"c-running":    "running",
		"c-waiting":    "waiting",
		"c-terminated": "terminated",
		"c-unknown":    "unknown",
	}
	for name, want := range cases {
		if stateMap[name] != want {
			t.Errorf("container %q Status = %q; want %q", name, stateMap[name], want)
		}
	}
}

// TestToPod_ContainerDetails_LastStatus verifies LastTerminationState is mapped.
func TestToPod_ContainerDetails_LastStatus(t *testing.T) {
	started := metav1.Time{Time: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	finished := metav1.Time{Time: time.Date(2024, 1, 1, 1, 0, 0, 0, time.UTC)}
	p := makePod("p", "default")
	p.Spec.Containers = []corev1.Container{{Name: "c"}}
	p.Status.ContainerStatuses = []corev1.ContainerStatus{
		{
			Name: "c",
			LastTerminationState: corev1.ContainerState{
				Terminated: &corev1.ContainerStateTerminated{
					Reason:     "OOMKilled",
					ExitCode:   137,
					StartedAt:  started,
					FinishedAt: finished,
				},
			},
		},
	}
	got := toPod(p)
	if len(got.ContainerDetails) != 1 {
		t.Fatalf("ContainerDetails len = %d; want 1", len(got.ContainerDetails))
	}
	ls := got.ContainerDetails[0].LastStatus
	if ls == nil {
		t.Fatal("LastStatus is nil; want non-nil")
	}
	if ls.Reason != "OOMKilled" {
		t.Errorf("LastStatus.Reason = %q; want OOMKilled", ls.Reason)
	}
	if ls.ExitCode != 137 {
		t.Errorf("LastStatus.ExitCode = %d; want 137", ls.ExitCode)
	}
	if ls.Started == "" || ls.Finished == "" {
		t.Error("LastStatus Started/Finished must not be empty")
	}
}

// TestToPod_ContainerDetails_EnvVars verifies env vars: plain values and refs.
func TestToPod_ContainerDetails_EnvVars(t *testing.T) {
	p := makePod("p", "default")
	p.Spec.Containers = []corev1.Container{
		{
			Name: "c",
			Env: []corev1.EnvVar{
				{Name: "PLAIN", Value: "value1"},
				{Name: "SECRET_KEY", ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{},
				}},
			},
		},
	}
	got := toPod(p)
	if len(got.ContainerDetails) != 1 {
		t.Fatalf("ContainerDetails len = %d; want 1", len(got.ContainerDetails))
	}
	envs := got.ContainerDetails[0].EnvVars
	if len(envs) != 2 {
		t.Fatalf("EnvVars len = %d; want 2", len(envs))
	}
	if envs[0] != "PLAIN=value1" {
		t.Errorf("EnvVars[0] = %q; want %q", envs[0], "PLAIN=value1")
	}
	if envs[1] != "SECRET_KEY=<ref>" {
		t.Errorf("EnvVars[1] = %q; want %q", envs[1], "SECRET_KEY=<ref>")
	}
}

// TestToPod_ContainerDetails_Ports verifies port mapping.
func TestToPod_ContainerDetails_Ports(t *testing.T) {
	p := makePod("p", "default")
	p.Spec.Containers = []corev1.Container{
		{
			Name: "c",
			Ports: []corev1.ContainerPort{
				{Name: "http", ContainerPort: 8080, Protocol: corev1.ProtocolTCP},
			},
		},
	}
	got := toPod(p)
	if len(got.ContainerDetails[0].Ports) != 1 {
		t.Fatalf("Ports len = %d; want 1", len(got.ContainerDetails[0].Ports))
	}
	port := got.ContainerDetails[0].Ports[0]
	if port.Name != "http" || port.ContainerPort != 8080 || port.Protocol != "TCP" {
		t.Errorf("Port = %+v; want name=http containerPort=8080 protocol=TCP", port)
	}
}

// TestToPod_Volumes_KindDetection verifies volume kind is detected for each type.
func TestToPod_Volumes_KindDetection(t *testing.T) {
	hostPathType := corev1.HostPathDirectory
	p := makePod("p", "default")
	p.Spec.Volumes = []corev1.Volume{
		{Name: "hp", VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/tmp", Type: &hostPathType}}},
		{Name: "ed", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{Medium: corev1.StorageMediumMemory}}},
		{Name: "cm", VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{}}},
		{Name: "sec", VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{}}},
		{Name: "pvc", VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{}}},
		{Name: "nfs", VolumeSource: corev1.VolumeSource{NFS: &corev1.NFSVolumeSource{}}},
		{Name: "dapi", VolumeSource: corev1.VolumeSource{DownwardAPI: &corev1.DownwardAPIVolumeSource{}}},
	}
	got := toPod(p)
	if len(got.Volumes) != 7 {
		t.Fatalf("Volumes len = %d; want 7", len(got.Volumes))
	}
	kindMap := map[string]string{}
	for _, v := range got.Volumes {
		kindMap[v.Name] = v.Kind
	}
	expected := map[string]string{
		"hp":   "hostPath",
		"ed":   "emptyDir",
		"cm":   "configMap",
		"sec":  "secret",
		"pvc":  "persistentVolumeClaim",
		"nfs":  "nfs",
		"dapi": "downwardAPI",
	}
	for name, want := range expected {
		if kindMap[name] != want {
			t.Errorf("volume %q Kind = %q; want %q", name, kindMap[name], want)
		}
	}
	// hostPath extras
	for _, v := range got.Volumes {
		if v.Name == "hp" {
			if v.HostPath != "/tmp" {
				t.Errorf("hostPath.HostPath = %q; want /tmp", v.HostPath)
			}
			if v.CheckBehavior != "Directory" {
				t.Errorf("hostPath.CheckBehavior = %q; want Directory", v.CheckBehavior)
			}
		}
		if v.Name == "ed" && v.Medium != "Memory" {
			t.Errorf("emptyDir.Medium = %q; want Memory", v.Medium)
		}
	}
}

// TestToPod_Volumes_Projected verifies projected volume sources are mapped.
func TestToPod_Volumes_Projected(t *testing.T) {
	exp := int64(3600)
	dm := int32(0o644)
	p := makePod("p", "default")
	p.Spec.Volumes = []corev1.Volume{
		{
			Name: "proj",
			VolumeSource: corev1.VolumeSource{
				Projected: &corev1.ProjectedVolumeSource{
					DefaultMode: &dm,
					Sources: []corev1.VolumeProjection{
						{ServiceAccountToken: &corev1.ServiceAccountTokenProjection{
							Path:              "token",
							ExpirationSeconds: &exp,
						}},
						{ConfigMap: &corev1.ConfigMapProjection{
							LocalObjectReference: corev1.LocalObjectReference{Name: "my-cm"},
							Items:                []corev1.KeyToPath{{Key: "k", Path: "p"}},
						}},
						{Secret: &corev1.SecretProjection{
							LocalObjectReference: corev1.LocalObjectReference{Name: "my-secret"},
							Items:                []corev1.KeyToPath{{Key: "sk", Path: "sp"}},
						}},
						{DownwardAPI: &corev1.DownwardAPIProjection{
							Items: []corev1.DownwardAPIVolumeFile{
								{Path: "labels", FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.labels"}},
							},
						}},
					},
				},
			},
		},
	}
	got := toPod(p)
	if len(got.Volumes) != 1 {
		t.Fatalf("Volumes len = %d; want 1", len(got.Volumes))
	}
	vol := got.Volumes[0]
	if vol.Kind != "projected" {
		t.Errorf("Kind = %q; want projected", vol.Kind)
	}
	if vol.DefaultMode != "0o644" {
		t.Errorf("DefaultMode = %q; want 0o644", vol.DefaultMode)
	}
	if len(vol.Sources) != 4 {
		t.Fatalf("Sources len = %d; want 4", len(vol.Sources))
	}
	srcTypes := map[string]bool{}
	for _, s := range vol.Sources {
		srcTypes[s.Type] = true
	}
	for _, want := range []string{"ServiceAccountToken", "ConfigMap", "Secret", "DownwardAPI"} {
		if !srcTypes[want] {
			t.Errorf("missing projected source type %q", want)
		}
	}
	for _, s := range vol.Sources {
		if s.Type == "ServiceAccountToken" && s.Expiration != "3600s" {
			t.Errorf("ServiceAccountToken.Expiration = %q; want 3600s", s.Expiration)
		}
		if s.Type == "ConfigMap" {
			if s.Name != "my-cm" {
				t.Errorf("ConfigMap.Name = %q; want my-cm", s.Name)
			}
			if len(s.Items) != 1 || s.Items[0] != "k→p" {
				t.Errorf("ConfigMap.Items = %v; want [k→p]", s.Items)
			}
		}
		if s.Type == "DownwardAPI" {
			if len(s.Items) != 1 || s.Items[0] != "metadata.labels" {
				t.Errorf("DownwardAPI.Items = %v; want [metadata.labels]", s.Items)
			}
		}
	}
}

// TestToPod_Labels verifies pod labels are passed through to the DTO.
func TestToPod_Labels(t *testing.T) {
	p := makePod("p", "default")
	p.Labels = map[string]string{"app": "web"}
	got := toPod(p)
	if got.Labels["app"] != "web" {
		t.Errorf("Labels[app] = %q; want web", got.Labels["app"])
	}
}

// TestApplyPodMetrics_UpdatesCPUAndMemory verifies metrics labels and percents are applied.
func TestApplyPodMetrics_UpdatesCPUAndMemory(t *testing.T) {
	pods := []dto.Pod{
		{
			Name:        "pod-a",
			Namespace:   "default",
			CPUReqMilli: 100,
			CPULimMilli: 200,
			MemReqBytes: 128 * 1024 * 1024,
			MemLimBytes: 256 * 1024 * 1024,
		},
	}
	usage := map[string]dto.PodUsage{
		"default/pod-a": {CPUMilliCores: 50, MemoryBytes: 64 * 1024 * 1024},
	}
	result := ApplyPodMetrics(pods, usage)
	if len(result) != 1 {
		t.Fatalf("result len = %d; want 1", len(result))
	}
	if result[0].CPUPercent != 25 {
		t.Errorf("CPUPercent = %d; want 25 (50/200*100)", result[0].CPUPercent)
	}
	if result[0].MemPercent != 25 {
		t.Errorf("MemPercent = %d; want 25 (64/256*100)", result[0].MemPercent)
	}
	if result[0].CPU == "" {
		t.Error("CPU label must not be empty after ApplyPodMetrics")
	}
	if result[0].Memory == "" {
		t.Error("Memory label must not be empty after ApplyPodMetrics")
	}
}

// TestApplyPodMetrics_MissingUsage verifies pod without usage entry is unchanged.
func TestApplyPodMetrics_MissingUsage(t *testing.T) {
	pods := []dto.Pod{
		{Name: "pod-x", Namespace: "default", CPUPercent: 0, MemPercent: 0},
	}
	result := ApplyPodMetrics(pods, map[string]dto.PodUsage{})
	if result[0].CPUPercent != 0 || result[0].MemPercent != 0 {
		t.Errorf("unexpected metrics update for pod without usage entry")
	}
}

// TestFormatCPUMilli verifies formatting edge cases.
func TestFormatCPUMilli(t *testing.T) {
	cases := []struct {
		input int64
		want  string
	}{
		{0, "—"},
		{-1, "—"},
		{100, "100m"},
		{1500, "1500m"},
	}
	for _, c := range cases {
		got := formatCPUMilli(c.input)
		if got != c.want {
			t.Errorf("formatCPUMilli(%d) = %q; want %q", c.input, got, c.want)
		}
	}
}

// TestFormatMemBytes verifies MiB and GiB formatting.
func TestFormatMemBytes(t *testing.T) {
	cases := []struct {
		input int64
		want  string
	}{
		{0, "—"},
		{-1, "—"},
		{128 * 1024 * 1024, "128Mi"},
		{2 * 1024 * 1024 * 1024, "2.0Gi"},
	}
	for _, c := range cases {
		got := formatMemBytes(c.input)
		if got != c.want {
			t.Errorf("formatMemBytes(%d) = %q; want %q", c.input, got, c.want)
		}
	}
}

// TestResourcePercent verifies percent clamping and fallback from limit to req.
func TestResourcePercent(t *testing.T) {
	if got := resourcePercent(0, 100, 200); got != 0 {
		t.Errorf("got %d; want 0 when usage=0", got)
	}
	if got := resourcePercent(50, 200, 100); got != 25 {
		t.Errorf("got %d; want 25 (limit=200 used)", got)
	}
	if got := resourcePercent(50, 0, 100); got != 50 {
		t.Errorf("got %d; want 50 (req fallback)", got)
	}
	if got := resourcePercent(500, 100, 0); got != 100 {
		t.Errorf("got %d; want 100 (clamped)", got)
	}
	if got := resourcePercent(10, 0, 0); got != 0 {
		t.Errorf("got %d; want 0 when denom=0", got)
	}
}

// TestProbeString_Variants verifies handler type formatting.
func TestProbeString_Variants(t *testing.T) {
	if got := probeString(nil); got != "" {
		t.Errorf("probeString(nil) = %q; want empty", got)
	}

	httpProbe := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Host:   "localhost",
				Port:   intstr.FromInt32(8080),
				Path:   "/health",
				Scheme: corev1.URISchemeHTTP,
			},
		},
	}
	got := probeString(httpProbe)
	if got == "" {
		t.Error("probeString(httpGet) must not be empty")
	}
	if !strings.Contains(got, "http-get") {
		t.Errorf("probeString(httpGet) = %q; want prefix http-get", got)
	}

	execProbe := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			Exec: &corev1.ExecAction{Command: []string{"cat", "/tmp/healthy"}},
		},
	}
	got = probeString(execProbe)
	if !strings.Contains(got, "exec cat /tmp/healthy") {
		t.Errorf("probeString(exec) = %q; want exec prefix", got)
	}

	tcpProbe := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			TCPSocket: &corev1.TCPSocketAction{Host: "localhost", Port: intstr.FromInt32(9090)},
		},
	}
	got = probeString(tcpProbe)
	if !strings.Contains(got, "tcp-socket") {
		t.Errorf("probeString(tcp) = %q; want tcp-socket prefix", got)
	}

	grpcProbe := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			GRPC: &corev1.GRPCAction{Port: 50051},
		},
	}
	got = probeString(grpcProbe)
	if !strings.Contains(got, "grpc :50051") {
		t.Errorf("probeString(grpc) = %q; want grpc :50051", got)
	}
}

// TestToPod_TerminationGracePeriod verifies TerminationGracePeriod is formatted as "<N>s".
func TestToPod_TerminationGracePeriod(t *testing.T) {
	p := makePod("p", "default")
	grace := int64(30)
	p.Spec.TerminationGracePeriodSeconds = &grace
	got := toPod(p)
	if got.TerminationGracePeriod != "30s" {
		t.Errorf("TerminationGracePeriod = %q; want 30s", got.TerminationGracePeriod)
	}

	p2 := makePod("p2", "default")
	got2 := toPod(p2)
	if got2.TerminationGracePeriod != "" {
		t.Errorf("TerminationGracePeriod = %q; want empty when nil", got2.TerminationGracePeriod)
	}
}

// ---------------------------------------------------------------------------
// Gap-fill tests — coverage not present in the original test file
// ---------------------------------------------------------------------------

// TestListPods_EmptyLister_ReturnsNonNilEmptySlice verifies empty lister yields a non-nil slice.
func TestListPods_EmptyLister_ReturnsNonNilEmptySlice(t *testing.T) {
	result, err := ListPods(newPodLister(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Error("expected non-nil slice; got nil")
	}
	if len(result) != 0 {
		t.Errorf("expected empty result; got %d items", len(result))
	}
}

// TestListPods_ErrorPropagation_NamespacedScope verifies namespaced lister errors propagate.
func TestListPods_ErrorPropagation_NamespacedScope(t *testing.T) {
	sentinel := errors.New("namespace store unavailable")
	_, err := ListPods(&errorPodLister{err: sentinel}, "default")
	if !errors.Is(err, sentinel) {
		t.Errorf("expected sentinel error; got %v", err)
	}
}

// TestToPod_ZeroContainers_ReadyString verifies Ready is "0/0" when there are no containers.
func TestToPod_ZeroContainers_ReadyString(t *testing.T) {
	p := makePod("p", "default")
	// No Spec.Containers, no ContainerStatuses
	got := toPod(p)
	if got.Ready != "0/0" {
		t.Errorf("Ready = %q; want \"0/0\"", got.Ready)
	}
	if got.Containers != 0 {
		t.Errorf("Containers = %d; want 0", got.Containers)
	}
}

// TestToPod_NilLabels_ReturnsEmptyMap verifies nil pod.Labels yields map[string]string{} not nil.
func TestToPod_NilLabels_ReturnsEmptyMap(t *testing.T) {
	p := makePod("p", "default")
	p.Labels = nil
	got := toPod(p)
	if got.Labels == nil {
		t.Error("Labels must not be nil when pod.Labels is nil; want empty map")
	}
	if len(got.Labels) != 0 {
		t.Errorf("Labels len = %d; want 0", len(got.Labels))
	}
}

// TestToPod_NilAnnotations_ReturnsEmptyMap verifies nil pod.Annotations yields map[string]string{} not nil.
func TestToPod_NilAnnotations_ReturnsEmptyMap(t *testing.T) {
	p := makePod("p", "default")
	p.Annotations = nil
	got := toPod(p)
	if got.Annotations == nil {
		t.Error("Annotations must not be nil when pod.Annotations is nil; want empty map")
	}
	if len(got.Annotations) != 0 {
		t.Errorf("Annotations len = %d; want 0", len(got.Annotations))
	}
}

// TestToPod_ManagedFields_EmptyInput_NonNilSlice verifies ManagedFields is always a non-nil slice.
func TestToPod_ManagedFields_EmptyInput_NonNilSlice(t *testing.T) {
	p := makePod("p", "default")
	got := toPod(p)
	if got.ManagedFields == nil {
		t.Error("ManagedFields must not be nil when pod has no managed fields; want empty slice")
	}
	if len(got.ManagedFields) != 0 {
		t.Errorf("ManagedFields len = %d; want 0", len(got.ManagedFields))
	}
}

// TestToPod_AllAffinities_CountThree verifies all three affinity types each increment the count.
func TestToPod_AllAffinities_CountThree(t *testing.T) {
	p := makePod("p", "default")
	p.Spec.Affinity = &corev1.Affinity{
		NodeAffinity:    &corev1.NodeAffinity{},
		PodAffinity:     &corev1.PodAffinity{},
		PodAntiAffinity: &corev1.PodAntiAffinity{},
	}
	got := toPod(p)
	if got.AffinityCount != 3 {
		t.Errorf("AffinityCount = %d; want 3", got.AffinityCount)
	}
}

// TestToPod_NilDeletionTimestamp_NotTerminating verifies nil DeletionTimestamp keeps Phase as Status.
func TestToPod_NilDeletionTimestamp_NotTerminating(t *testing.T) {
	p := makePod("p", "default")
	p.DeletionTimestamp = nil
	p.Status.Phase = corev1.PodRunning
	got := toPod(p)
	if got.Status == "Terminating" {
		t.Errorf("Status = %q; want non-Terminating when DeletionTimestamp is nil", got.Status)
	}
	if got.Status != "Running" {
		t.Errorf("Status = %q; want \"Running\"", got.Status)
	}
}

// TestProbeString_HTTPGet_EmptyScheme verifies empty scheme defaults to "http://" in probe output.
func TestProbeString_HTTPGet_EmptyScheme(t *testing.T) {
	p := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				// Scheme intentionally empty — must default to "http"
				Host: "myhost",
				Port: intstr.FromInt32(8080),
				Path: "/healthz",
			},
		},
	}
	got := probeString(p)
	if !strings.Contains(got, "http://") {
		t.Errorf("probeString with empty scheme = %q; want to contain \"http://\"", got)
	}
}

// TestApplyPodMetrics_DiskPercent_StaysZero verifies ApplyPodMetrics never touches DiskPercent.
func TestApplyPodMetrics_DiskPercent_StaysZero(t *testing.T) {
	pods := []dto.Pod{
		{
			Name:         "p",
			Namespace:    "default",
			DiskReqBytes: 1024 * 1024 * 1024,
			DiskLimBytes: 2 * 1024 * 1024 * 1024,
			DiskPercent:  0,
		},
	}
	usage := map[string]dto.PodUsage{
		"default/p": {CPUMilliCores: 100, MemoryBytes: 128 * 1024 * 1024},
	}
	result := ApplyPodMetrics(pods, usage)
	if result[0].DiskPercent != 0 {
		t.Errorf("DiskPercent = %d; want 0 (ApplyPodMetrics does not update disk)", result[0].DiskPercent)
	}
}

// TestToPod_Volume_Unknown verifies an empty VolumeSource produces kind "unknown".
func TestToPod_Volume_Unknown(t *testing.T) {
	p := makePod("p", "default")
	p.Spec.Volumes = []corev1.Volume{
		{Name: "mystery", VolumeSource: corev1.VolumeSource{}},
	}
	got := toPod(p)
	if len(got.Volumes) != 1 {
		t.Fatalf("Volumes len = %d; want 1", len(got.Volumes))
	}
	if got.Volumes[0].Kind != "unknown" {
		t.Errorf("Kind = %q; want \"unknown\"", got.Volumes[0].Kind)
	}
}

// TestToPod_ReadyDenominator_FromContainerStatuses documents that the Ready denominator
// comes from ContainerStatuses length, not Spec.Containers length.
func TestToPod_ReadyDenominator_FromContainerStatuses(t *testing.T) {
	p := makePod("p", "default")
	p.Spec.Containers = []corev1.Container{
		{Name: "c1"}, {Name: "c2"}, {Name: "c3"},
	}
	// Only 2 of 3 containers have reported status yet.
	p.Status.ContainerStatuses = []corev1.ContainerStatus{
		{Name: "c1", Ready: true},
		{Name: "c2", Ready: true},
	}
	got := toPod(p)
	// Denominator is ContainerStatuses count (2), not Spec.Containers count (3).
	if got.Ready != "2/2" {
		t.Errorf("Ready = %q; want \"2/2\" (denominator from ContainerStatuses len)", got.Ready)
	}
	// Spec.Containers count is tracked separately.
	if got.Containers != 3 {
		t.Errorf("Containers = %d; want 3", got.Containers)
	}
}

// TestToPod_Volumes_Projected_NilDefaultMode verifies nil DefaultMode yields an empty string.
func TestToPod_Volumes_Projected_NilDefaultMode(t *testing.T) {
	p := makePod("p", "default")
	p.Spec.Volumes = []corev1.Volume{
		{
			Name: "proj",
			VolumeSource: corev1.VolumeSource{
				Projected: &corev1.ProjectedVolumeSource{
					DefaultMode: nil,
				},
			},
		},
	}
	got := toPod(p)
	if len(got.Volumes) != 1 {
		t.Fatalf("Volumes len = %d; want 1", len(got.Volumes))
	}
	if got.Volumes[0].DefaultMode != "" {
		t.Errorf("DefaultMode = %q; want empty string when nil", got.Volumes[0].DefaultMode)
	}
}

// TestContainerStatusMessage tests

func TestContainerStatusMessage_PlainState(t *testing.T) {
	got := containerStatusMessage("running", false, nil)
	if got != "running" {
		t.Errorf("got %q; want \"running\"", got)
	}
}

func TestContainerStatusMessage_StateWithReady(t *testing.T) {
	got := containerStatusMessage("running", true, nil)
	if got != "running, ready" {
		t.Errorf("got %q; want \"running, ready\"", got)
	}
}

func TestContainerStatusMessage_StateWithTerminated(t *testing.T) {
	terminated := &corev1.ContainerStateTerminated{
		Reason:   "OOMKilled",
		ExitCode: 137,
	}
	got := containerStatusMessage("terminated", false, terminated)
	if got != "terminated - OOMKilled (exit code: 137)" {
		t.Errorf("got %q; want \"terminated - OOMKilled (exit code: 137)\"", got)
	}
}

func TestContainerStatusMessage_ReadyAndTerminated(t *testing.T) {
	terminated := &corev1.ContainerStateTerminated{
		Reason:   "Completed",
		ExitCode: 0,
	}
	got := containerStatusMessage("running", true, terminated)
	if got != "running, ready - Completed (exit code: 0)" {
		t.Errorf("got %q; want \"running, ready - Completed (exit code: 0)\"", got)
	}
}

func TestContainerStatusMessage_WaitingState(t *testing.T) {
	got := containerStatusMessage("waiting", false, nil)
	if got != "waiting" {
		t.Errorf("got %q; want \"waiting\"", got)
	}
}

func TestContainerStatusMessage_UnknownState(t *testing.T) {
	got := containerStatusMessage("unknown", false, nil)
	if got != "unknown" {
		t.Errorf("got %q; want \"unknown\"", got)
	}
}

// TestToPod_Volumes_Projected_DownwardAPI_ResourceFieldRef verifies the ResourceFieldRef branch.
func TestToPod_Volumes_Projected_DownwardAPI_ResourceFieldRef(t *testing.T) {
	p := makePod("p", "default")
	p.Spec.Volumes = []corev1.Volume{
		{
			Name: "proj",
			VolumeSource: corev1.VolumeSource{
				Projected: &corev1.ProjectedVolumeSource{
					Sources: []corev1.VolumeProjection{
						{DownwardAPI: &corev1.DownwardAPIProjection{
							Items: []corev1.DownwardAPIVolumeFile{
								{Path: "cpu", ResourceFieldRef: &corev1.ResourceFieldSelector{Resource: "requests.cpu"}},
							},
						}},
					},
				},
			},
		},
	}
	got := toPod(p)
	if len(got.Volumes) != 1 {
		t.Fatalf("Volumes len = %d; want 1", len(got.Volumes))
	}
	src := got.Volumes[0].Sources
	if len(src) != 1 || src[0].Type != "DownwardAPI" {
		t.Fatalf("Sources = %v; want 1 DownwardAPI entry", src)
	}
	if len(src[0].Items) != 1 || src[0].Items[0] != "requests.cpu" {
		t.Errorf("DownwardAPI ResourceFieldRef Items = %v; want [requests.cpu]", src[0].Items)
	}
}

// TestToPod_EmptyIPFields_ReturnsNil verifies nil is returned when both slice and legacy fields are empty.
func TestToPod_EmptyIPFields_ReturnsNil(t *testing.T) {
	p := makePod("p", "default")
	// HostIPs, HostIP, PodIPs, PodIP all at zero values.
	got := toPod(p)
	if got.HostIPs != nil {
		t.Errorf("HostIPs = %v; want nil when no IP info present", got.HostIPs)
	}
	if got.PodIPs != nil {
		t.Errorf("PodIPs = %v; want nil when no IP info present", got.PodIPs)
	}
}

// — containerStatusMessage edge cases —

func TestContainerStatusMessage_EmptyStateString_IncludesReadyFlag(t *testing.T) {
	msg := containerStatusMessage("", true, nil)
	if !strings.Contains(msg, "ready") {
		t.Errorf("msg = %q; want to contain 'ready'", msg)
	}
}

func TestContainerStatusMessage_ReadyFalseNoTerminated_OnlyState(t *testing.T) {
	msg := containerStatusMessage("running", false, nil)
	if msg != "running" {
		t.Errorf("msg = %q; want %q", msg, "running")
	}
}

func TestContainerStatusMessage_TerminatedWithExitCode_IncludesReasonAndCode(t *testing.T) {
	terminated := &corev1.ContainerStateTerminated{
		Reason:   "OOMKilled",
		ExitCode: 137,
	}
	msg := containerStatusMessage("terminated", false, terminated)
	if !strings.Contains(msg, "OOMKilled") {
		t.Errorf("msg = %q; want to contain OOMKilled", msg)
	}
	if !strings.Contains(msg, "137") {
		t.Errorf("msg = %q; want to contain exit code 137", msg)
	}
}

func TestContainerStatusMessage_TerminatedWithZeroExitCode_IncludesZero(t *testing.T) {
	terminated := &corev1.ContainerStateTerminated{
		Reason:   "Completed",
		ExitCode: 0,
	}
	msg := containerStatusMessage("terminated", false, terminated)
	if !strings.Contains(msg, "0") {
		t.Errorf("msg = %q; want to contain exit code 0", msg)
	}
}

func TestContainerStatusMessage_TerminatedWithNegativeExitCode_IncludesNegative(t *testing.T) {
	terminated := &corev1.ContainerStateTerminated{
		Reason:   "Signal",
		ExitCode: -1,
	}
	msg := containerStatusMessage("terminated", false, terminated)
	if !strings.Contains(msg, "-1") {
		t.Errorf("msg = %q; want to contain exit code -1", msg)
	}
}

func TestContainerStatusMessage_TerminatedEmptyReason_StillIncludesExitCode(t *testing.T) {
	terminated := &corev1.ContainerStateTerminated{
		Reason:   "",
		ExitCode: 1,
	}
	msg := containerStatusMessage("terminated", false, terminated)
	if !strings.Contains(msg, "1") {
		t.Errorf("msg = %q; want to contain exit code 1", msg)
	}
}

// — buildContainerDetails edge cases —

func TestBuildContainerDetails_EmptyContainers_ReturnsEmptySlice(t *testing.T) {
	result := buildContainerDetails([]corev1.Container{}, []corev1.ContainerStatus{})
	if result == nil {
		t.Error("result must not be nil")
	}
	if len(result) != 0 {
		t.Errorf("length = %d; want 0", len(result))
	}
}

func TestBuildContainerDetails_ContainerNoMatchingStatus_DefaultsToUnknown(t *testing.T) {
	containers := []corev1.Container{
		{Name: "app", Image: "app:1.0"},
	}
	statuses := []corev1.ContainerStatus{
		{Name: "other", Ready: true},
	}
	result := buildContainerDetails(containers, statuses)
	if len(result) != 1 {
		t.Fatalf("length = %d; want 1", len(result))
	}
	if result[0].Status != "unknown" {
		t.Errorf("Status = %q; want unknown", result[0].Status)
	}
	if result[0].Name != "app" {
		t.Errorf("Name = %q; want app", result[0].Name)
	}
}

func TestBuildContainerDetails_MultipleContainers_AllMatched(t *testing.T) {
	containers := []corev1.Container{
		{Name: "app", Image: "app:1.0"},
		{Name: "sidecar", Image: "sidecar:1.0"},
	}
	statuses := []corev1.ContainerStatus{
		{Name: "app", Ready: true, State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}},
		{Name: "sidecar", Ready: false, State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "Pending"}}},
	}
	result := buildContainerDetails(containers, statuses)
	if len(result) != 2 {
		t.Fatalf("length = %d; want 2", len(result))
	}
	if result[0].Status != "running" {
		t.Errorf("result[0].Status = %q; want running", result[0].Status)
	}
	if result[1].Status != "waiting" {
		t.Errorf("result[1].Status = %q; want waiting", result[1].Status)
	}
}

func TestBuildContainerDetails_TerminatedContainerZeroExitCode_Captured(t *testing.T) {
	containers := []corev1.Container{
		{Name: "app", Image: "app:1.0"},
	}
	statuses := []corev1.ContainerStatus{
		{
			Name: "app",
			State: corev1.ContainerState{
				Terminated: &corev1.ContainerStateTerminated{
					ExitCode: 0,
					Reason:   "Completed",
				},
			},
		},
	}
	result := buildContainerDetails(containers, statuses)
	if len(result) != 1 {
		t.Fatalf("length = %d; want 1", len(result))
	}
	if result[0].Status != "terminated" {
		t.Errorf("Status = %q; want terminated", result[0].Status)
	}
	if result[0].ExitCode == nil {
		t.Error("ExitCode must not be nil")
	}
	if *result[0].ExitCode != 0 {
		t.Errorf("ExitCode = %d; want 0", *result[0].ExitCode)
	}
}

func TestBuildContainerDetails_TerminatedContainerNegativeExitCode_Captured(t *testing.T) {
	containers := []corev1.Container{
		{Name: "app", Image: "app:1.0"},
	}
	statuses := []corev1.ContainerStatus{
		{
			Name: "app",
			State: corev1.ContainerState{
				Terminated: &corev1.ContainerStateTerminated{
					ExitCode: -1,
					Reason:   "Signal",
				},
			},
		},
	}
	result := buildContainerDetails(containers, statuses)
	if len(result) != 1 {
		t.Fatalf("length = %d; want 1", len(result))
	}
	if result[0].ExitCode == nil {
		t.Error("ExitCode must not be nil")
	}
	if *result[0].ExitCode != -1 {
		t.Errorf("ExitCode = %d; want -1", *result[0].ExitCode)
	}
}
