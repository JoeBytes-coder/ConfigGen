package render

import (
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	sigsyaml "sigs.k8s.io/yaml"

	"configgen/internal/domain"
)

func allKindsSelection() domain.Selection {
	kinds := make([]string, 0, len(domain.Catalog))
	for _, spec := range domain.Catalog {
		// HPA 与多个可伸缩工作负载同时存在属于歧义，单独在 TestRenderHPA 中覆盖。
		if spec.Kind == "HorizontalPodAutoscaler" {
			continue
		}
		kinds = append(kinds, spec.Kind)
	}
	return domain.Selection{
		Type: domain.TypeK8s,
		App: domain.AppContext{
			Name: "demo", Namespace: "prod", Image: "nginx", Tag: "1.25",
			Replicas:  2,
			Ports:     []domain.Port{{Name: "http", Container: 8080, Service: 80}},
			Env:       map[string]string{"FOO": "bar"},
			SecretEnv: map[string]string{"TOKEN": "s3cr3t"},
			Resources: domain.ResourceRequirements{
				Requests: map[string]string{"cpu": "100m", "memory": "128Mi"},
				Limits:   map[string]string{"cpu": "500m", "memory": "512Mi"},
			},
			Probe: domain.ProbeConfig{Enabled: true, Path: "/healthz"},
		},
		Resources: kinds,
	}
}

func artifactsByKind(files []domain.Artifact) map[string]string {
	out := make(map[string]string, len(files))
	for _, file := range files {
		out[file.Kind] = file.Content
	}
	return out
}

func TestRenderAllKindsProducesValidYAML(t *testing.T) {
	plan, err := domain.Resolve(allKindsSelection())
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	files, bundle, err := Render(plan)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	if len(files) != len(plan.Nodes) {
		t.Fatalf("expected %d files, got %d", len(plan.Nodes), len(files))
	}
	if !strings.Contains(bundle, "kind: Deployment") || !strings.Contains(bundle, "kind: Service") {
		t.Error("bundle missing core resources")
	}
	for _, file := range files {
		var obj map[string]any
		if err := sigsyaml.Unmarshal([]byte(file.Content), &obj); err != nil {
			t.Fatalf("%s invalid yaml: %v", file.Path, err)
		}
		if obj["kind"] == nil {
			t.Errorf("%s missing kind", file.Path)
		}
	}
}

func TestRenderWorkloadReferences(t *testing.T) {
	selection := domain.Selection{
		Type: domain.TypeK8s,
		App: domain.AppContext{
			Name: "demo", Image: "nginx", Tag: "1.25", Replicas: 2,
			Ports: []domain.Port{{Name: "http", Container: 8080, Service: 80}},
			Env:   map[string]string{"FOO": "bar"},
		},
		Resources: []string{"Deployment", "Service", "ConfigMap", "Secret", "PersistentVolumeClaim", "ServiceAccount"},
	}
	plan, err := domain.Resolve(selection)
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	files, _, err := Render(plan)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	contents := artifactsByKind(files)

	var deployment appsv1.Deployment
	if err := sigsyaml.Unmarshal([]byte(contents["Deployment"]), &deployment); err != nil {
		t.Fatalf("decode deployment: %v", err)
	}
	pod := deployment.Spec.Template.Spec
	if pod.ServiceAccountName != "demo-sa" {
		t.Errorf("expected serviceAccountName demo-sa, got %q", pod.ServiceAccountName)
	}
	if len(pod.Containers) != 1 {
		t.Fatalf("expected 1 container, got %d", len(pod.Containers))
	}
	container := pod.Containers[0]
	if container.Image != "nginx:1.25" {
		t.Errorf("unexpected image %q", container.Image)
	}
	if len(container.Ports) != 1 || container.Ports[0].ContainerPort != 8080 {
		t.Errorf("unexpected container ports: %+v", container.Ports)
	}
	if len(container.EnvFrom) != 1 || container.EnvFrom[0].ConfigMapRef == nil || container.EnvFrom[0].ConfigMapRef.Name != "demo-configmap" {
		t.Errorf("expected envFrom configmap demo-configmap, got %+v", container.EnvFrom)
	}
	secretVolume, pvcVolume := false, false
	for _, volume := range pod.Volumes {
		if volume.Secret != nil && volume.Secret.SecretName == "demo-secret" {
			secretVolume = true
		}
		if volume.PersistentVolumeClaim != nil && volume.PersistentVolumeClaim.ClaimName == "demo-pvc" {
			pvcVolume = true
		}
	}
	if !secretVolume {
		t.Error("expected secret volume demo-secret")
	}
	if !pvcVolume {
		t.Error("expected pvc volume demo-pvc")
	}

	var service corev1.Service
	if err := sigsyaml.Unmarshal([]byte(contents["Service"]), &service); err != nil {
		t.Fatalf("decode service: %v", err)
	}
	if service.Spec.Selector["app"] != "demo" {
		t.Errorf("unexpected service selector: %v", service.Spec.Selector)
	}
	if len(service.Spec.Ports) != 1 || int(service.Spec.Ports[0].Port) != 80 || service.Spec.Ports[0].TargetPort.IntValue() != 8080 {
		t.Errorf("unexpected service ports: %+v", service.Spec.Ports)
	}
	if service.Spec.ClusterIP == corev1.ClusterIPNone {
		t.Error("service should not be headless for Deployment")
	}
}

func TestRenderHPA(t *testing.T) {
	plan, err := domain.Resolve(domain.Selection{
		Type: domain.TypeK8s,
		App: domain.AppContext{
			Name: "demo", Image: "nginx", Tag: "1.25", Replicas: 2,
			Ports: []domain.Port{{Name: "http", Container: 80, Service: 80}},
		},
		Resources: []string{"Deployment", "HorizontalPodAutoscaler"},
	})
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	files, _, err := Render(plan)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	var hpa autoscalingv2.HorizontalPodAutoscaler
	if err := sigsyaml.Unmarshal([]byte(artifactsByKind(files)["HorizontalPodAutoscaler"]), &hpa); err != nil {
		t.Fatalf("decode hpa: %v", err)
	}
	if hpa.Spec.ScaleTargetRef.Kind != "Deployment" || hpa.Spec.ScaleTargetRef.Name != "demo-deployment" {
		t.Errorf("unexpected scale target: %+v", hpa.Spec.ScaleTargetRef)
	}
	if hpa.Spec.MinReplicas == nil || *hpa.Spec.MinReplicas != 2 {
		t.Errorf("unexpected minReplicas: %v", hpa.Spec.MinReplicas)
	}
	if hpa.Spec.MaxReplicas != 5 {
		t.Errorf("expected maxReplicas 5, got %d", hpa.Spec.MaxReplicas)
	}
	if len(hpa.Spec.Metrics) != 1 || hpa.Spec.Metrics[0].Resource == nil || hpa.Spec.Metrics[0].Resource.Target.AverageUtilization == nil {
		t.Fatalf("unexpected metrics: %+v", hpa.Spec.Metrics)
	}
	if *hpa.Spec.Metrics[0].Resource.Target.AverageUtilization != 70 {
		t.Errorf("expected CPU utilization 70, got %d", *hpa.Spec.Metrics[0].Resource.Target.AverageUtilization)
	}
}

func TestRenderBatchJobs(t *testing.T) {
	plan, err := domain.Resolve(domain.Selection{
		Type: domain.TypeK8s,
		App: domain.AppContext{
			Name: "batch", Image: "busybox", Tag: "1.36", Ports: []domain.Port{{Name: "http", Container: 8080, Service: 8080}},
			Probe: domain.ProbeConfig{Enabled: true, Path: "/healthz"},
		},
		Resources: []string{"Job", "CronJob"},
	})
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	files, _, err := Render(plan)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	contents := artifactsByKind(files)

	var job batchv1.Job
	if err := sigsyaml.Unmarshal([]byte(contents["Job"]), &job); err != nil {
		t.Fatalf("decode job: %v", err)
	}
	if job.Spec.Template.Spec.RestartPolicy != corev1.RestartPolicyNever {
		t.Errorf("job restartPolicy must be Never, got %q", job.Spec.Template.Spec.RestartPolicy)
	}
	if len(job.Spec.Template.Spec.Containers) != 1 {
		t.Fatalf("expected 1 job container")
	}
	jobContainer := job.Spec.Template.Spec.Containers[0]
	if len(jobContainer.Ports) != 0 {
		t.Errorf("job must not declare container ports, got %+v", jobContainer.Ports)
	}
	if jobContainer.LivenessProbe != nil || jobContainer.ReadinessProbe != nil {
		t.Error("job must not declare probes when container ports are omitted")
	}

	var cronJob batchv1.CronJob
	if err := sigsyaml.Unmarshal([]byte(contents["CronJob"]), &cronJob); err != nil {
		t.Fatalf("decode cronjob: %v", err)
	}
	if cronJob.Spec.JobTemplate.Spec.Template.Spec.RestartPolicy != corev1.RestartPolicyNever {
		t.Errorf("cronjob restartPolicy must be Never, got %q", cronJob.Spec.JobTemplate.Spec.Template.Spec.RestartPolicy)
	}
}
