package builders

import (
	"fmt"
	"sort"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"configgen/internal/domain"
)

// Builder 把一个 Plan 节点渲染为具体的 K8s 对象。
type Builder func(app domain.AppContext, n domain.Node) (any, error)

var registry = map[string]Builder{}

func register(kind string, b Builder) { registry[kind] = b }

func Build(app domain.AppContext, n domain.Node) (any, error) {
	b, ok := registry[n.Kind]
	if !ok {
		return nil, fmt.Errorf("no builder registered for %s", n.Kind)
	}
	return b(app, n)
}

func labels(app domain.AppContext) map[string]string {
	return map[string]string{"app": app.Name, "app.kubernetes.io/managed-by": "configgen"}
}

func selectorLabels(app domain.AppContext) map[string]string {
	return map[string]string{"app": app.Name}
}

func objectMeta(app domain.AppContext, n domain.Node) metav1.ObjectMeta {
	return metav1.ObjectMeta{Name: n.Name, Namespace: n.Namespace, Labels: labels(app)}
}

func int32Ptr(v int32) *int32 { return &v }

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func envVars(env map[string]string) []corev1.EnvVar {
	out := make([]corev1.EnvVar, 0, len(env))
	for _, k := range sortedKeys(env) {
		out = append(out, corev1.EnvVar{Name: k, Value: env[k]})
	}
	return out
}

func resourceList(in map[string]string) corev1.ResourceList {
	out := corev1.ResourceList{}
	for _, k := range sortedKeys(in) {
		if q, err := resource.ParseQuantity(in[k]); err == nil {
			out[corev1.ResourceName(k)] = q
		}
	}
	return out
}

func containerPorts(app domain.AppContext) []corev1.ContainerPort {
	out := make([]corev1.ContainerPort, 0, len(app.Ports))
	for _, p := range app.Ports {
		out = append(out, corev1.ContainerPort{Name: p.Name, ContainerPort: int32(p.Container), Protocol: corev1.ProtocolTCP})
	}
	return out
}

func httpProbe(path, portName string) *corev1.Probe {
	if path == "" {
		path = "/"
	}
	return &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{Path: path, Port: intstr.FromString(portName)},
		},
		InitialDelaySeconds: 5,
		PeriodSeconds:       10,
		TimeoutSeconds:      2,
		FailureThreshold:    3,
	}
}

// podSpec 是工作负载共用容器模板：环境变量、挂载、探针、资源配额。
func podSpec(app domain.AppContext, n domain.Node, withPorts bool) corev1.PodSpec {
	container := corev1.Container{Name: app.Name, Image: app.ImageRef()}
	if withPorts {
		container.Ports = containerPorts(app)
	}
	if len(app.Resources.Requests) > 0 || len(app.Resources.Limits) > 0 {
		container.Resources = corev1.ResourceRequirements{
			Requests: resourceList(app.Resources.Requests),
			Limits:   resourceList(app.Resources.Limits),
		}
	}
	if ref, ok := n.Refs["configmap"]; ok {
		container.EnvFrom = append(container.EnvFrom, corev1.EnvFromSource{
			ConfigMapRef: &corev1.ConfigMapEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: ref.Name}},
		})
	} else if len(app.Env) > 0 {
		container.Env = envVars(app.Env)
	}
	if ref, ok := n.Refs["secret"]; ok && len(app.SecretEnv) > 0 {
		container.EnvFrom = append(container.EnvFrom, corev1.EnvFromSource{
			SecretRef: &corev1.SecretEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: ref.Name}},
		})
	}
	if withPorts && len(app.Ports) > 0 && app.Probe.Enabled {
		portName := app.Ports[0].Name
		container.LivenessProbe = httpProbe(app.Probe.Path, portName)
		readiness := app.Probe.ReadinessPath
		if readiness == "" {
			readiness = app.Probe.Path
		}
		container.ReadinessProbe = httpProbe(readiness, portName)
	}

	spec := corev1.PodSpec{Containers: []corev1.Container{container}}
	if ref, ok := n.Refs["sa"]; ok {
		spec.ServiceAccountName = ref.Name
	}
	if ref, ok := n.Refs["secret"]; ok && len(app.SecretEnv) == 0 {
		container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{Name: "secret-volume", MountPath: "/etc/secret", ReadOnly: true})
		spec.Volumes = append(spec.Volumes, corev1.Volume{
			Name:         "secret-volume",
			VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{SecretName: ref.Name}},
		})
	}
	if ref, ok := n.Refs["pvc"]; ok {
		container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{Name: "data", MountPath: "/data"})
		spec.Volumes = append(spec.Volumes, corev1.Volume{
			Name:         "data",
			VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: ref.Name}},
		})
	}
	spec.Containers = []corev1.Container{container}
	return spec
}
