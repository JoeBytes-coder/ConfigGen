package domain

import (
	"errors"
	"strings"
	"testing"
)

func sel(resources ...string) Selection {
	return Selection{
		Type: TypeK8s,
		App: AppContext{
			Name: "myapp", Image: "nginx", Tag: "1.25", Replicas: 2,
			Ports: []Port{{Name: "http", Container: 8080, Service: 80}},
			Env:   map[string]string{"FOO": "bar"},
		},
		Resources: resources,
	}
}

func TestResolveAutoAddsDependencies(t *testing.T) {
	plan, err := Resolve(sel("Ingress"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	kinds := map[string]bool{}
	for _, n := range plan.Nodes {
		kinds[n.Kind] = true
		if !n.AutoAdded && n.Kind != "Ingress" {
			t.Errorf("%s should be auto-added", n.Kind)
		}
	}
	for _, want := range []string{"Ingress", "Service", "Deployment"} {
		if !kinds[want] {
			t.Errorf("missing auto-added %s", want)
		}
	}
}

func TestResolveValidatesAppName(t *testing.T) {
	s := sel("Deployment")
	s.App.Name = "My App"
	plan, err := Resolve(s)
	if err == nil {
		t.Fatal("expected validation error")
	}
	var appErr *AppError
	if !errors.As(err, &appErr) || appErr.Kind != ErrBadRequest {
		t.Fatalf("expected bad request AppError, got %v", err)
	}
	if plan.Valid {
		t.Error("plan must be invalid")
	}
	if len(plan.Diagnostics) == 0 {
		t.Error("expected diagnostics")
	}
}

func TestResolveHPAAmbiguity(t *testing.T) {
	_, err := Resolve(sel("Deployment", "StatefulSet", "HorizontalPodAutoscaler"))
	var appErr *AppError
	if !errors.As(err, &appErr) || appErr.Kind != ErrUnprocessable {
		t.Fatalf("expected unprocessable error, got %v", err)
	}
}

func TestResolveHPAUnsupportedTarget(t *testing.T) {
	_, err := Resolve(sel("DaemonSet", "HorizontalPodAutoscaler"))
	var appErr *AppError
	if !errors.As(err, &appErr) || appErr.Kind != ErrUnprocessable {
		t.Fatalf("expected unprocessable error, got %v", err)
	}
}

func TestResolveStatefulSetAddsHeadlessService(t *testing.T) {
	plan, err := Resolve(sel("StatefulSet"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var stateful *Node
	for i := range plan.Nodes {
		if plan.Nodes[i].Kind == "StatefulSet" {
			stateful = &plan.Nodes[i]
		}
	}
	if stateful == nil {
		t.Fatal("missing StatefulSet node")
	}
	if stateful.Refs["service"].Name != "myapp-service" {
		t.Errorf("statefulset should reference service, got %+v", stateful.Refs)
	}
}

func TestResolveNamespace(t *testing.T) {
	s := sel("Deployment", "Namespace")
	plan, err := Resolve(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan.App.Namespace != "myapp-namespace" {
		t.Errorf("expected generated namespace, got %q", plan.App.Namespace)
	}
	for _, n := range plan.Nodes {
		if n.Kind != "Namespace" && n.Namespace != "myapp-namespace" {
			t.Errorf("%s should carry namespace, got %q", n.Kind, n.Namespace)
		}
	}
}

func TestResolveOrderingAndDedup(t *testing.T) {
	plan, err := Resolve(sel("Service", "Deployment", "Service", "ConfigMap"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plan.Nodes) != 3 {
		t.Fatalf("expected 3 nodes after dedupe, got %d", len(plan.Nodes))
	}
	levels := []int{}
	for _, n := range plan.Nodes {
		levels = append(levels, n.Level)
	}
	for i := 1; i < len(levels); i++ {
		if levels[i] < levels[i-1] {
			t.Errorf("nodes not sorted by level: %v", levels)
		}
	}
}

func TestResolveNoResources(t *testing.T) {
	_, err := Resolve(sel())
	var appErr *AppError
	if !errors.As(err, &appErr) || appErr.Kind != ErrBadRequest {
		t.Fatalf("expected bad request, got %v", err)
	}
}

func TestResolveWorkloadOnlyHasNoPorts(t *testing.T) {
	s := sel("Deployment")
	s.App.Ports = nil
	plan, err := Resolve(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plan.App.Ports) != 0 {
		t.Errorf("workload-only selection should not invent ports, got %+v", plan.App.Ports)
	}
}

func TestResolveServiceGetsPortDefaults(t *testing.T) {
	s := sel("Service")
	s.App.Ports = nil
	plan, err := Resolve(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plan.App.Ports) != 1 || plan.App.Ports[0].Service != 80 {
		t.Errorf("expected default port 80, got %+v", plan.App.Ports)
	}
}

func TestResolveWhitespaceResources(t *testing.T) {
	_, err := Resolve(Selection{Type: TypeK8s, App: AppContext{Name: "demo"}, Resources: []string{" ", ""}})
	var appErr *AppError
	if !errors.As(err, &appErr) || appErr.Kind != ErrBadRequest {
		t.Fatalf("expected bad request for whitespace resources, got %v", err)
	}
}

func hasDiagnostic(diags []Diagnostic, code string) bool {
	for _, d := range diags {
		if d.Code == code {
			return true
		}
	}
	return false
}

func TestResolveSecretEnvAutoAddsSecret(t *testing.T) {
	s := sel("Deployment")
	s.App.SecretEnv = map[string]string{"TOKEN": "x"}
	plan, err := Resolve(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, node := range plan.Nodes {
		if node.Kind == "Secret" {
			found = true
			if !node.AutoAdded {
				t.Error("secret should be auto-added")
			}
		}
	}
	if !found {
		t.Fatal("expected auto-added Secret when secret_env is set")
	}
}

func TestResolveInvalidResourceQuantity(t *testing.T) {
	s := sel("Deployment")
	s.App.Resources = ResourceRequirements{Requests: map[string]string{"cpu": "not-a-quantity"}}
	_, err := Resolve(s)
	var appErr *AppError
	if !errors.As(err, &appErr) || appErr.Kind != ErrBadRequest {
		t.Fatalf("expected bad request for invalid quantity, got %v", err)
	}
	if !hasDiagnostic(appErr.Diagnostics, "invalid_resource_quantity") {
		t.Error("expected invalid_resource_quantity diagnostic")
	}
}

func TestResolveIgnoredInputWarnings(t *testing.T) {
	s := sel("Deployment")
	s.App.Ports = nil
	s.App.Host = "app.example.com"
	s.App.Probe = ProbeConfig{Enabled: true, Path: "/healthz"}
	plan, err := Resolve(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, code := range []string{"host_ignored", "probe_ignored_no_ports"} {
		if !hasDiagnostic(plan.Diagnostics, code) {
			t.Errorf("expected diagnostic %s", code)
		}
	}
}

func TestResolveFixedValueWarnings(t *testing.T) {
	plan, err := Resolve(sel("CronJob", "PersistentVolumeClaim", "EndpointSlice"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, code := range []string{"cronjob_default_schedule", "pvc_default_size", "endpointslice_empty"} {
		if !hasDiagnostic(plan.Diagnostics, code) {
			t.Errorf("expected diagnostic %s", code)
		}
	}
}

func TestResolveMultipleWorkloadsWarning(t *testing.T) {
	plan, err := Resolve(sel("Deployment", "StatefulSet", "Service"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, code := range []string{"multiple_workloads", "statefulset_service_not_headless"} {
		if !hasDiagnostic(plan.Diagnostics, code) {
			t.Errorf("expected diagnostic %s", code)
		}
	}
}

func TestResolveDiagnosticsAreInformative(t *testing.T) {
	plan, err := Resolve(sel("Deployment", "NetworkPolicy", "StorageClass"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	joined := ""
	for _, d := range plan.Diagnostics {
		joined += d.Code + " "
	}
	if !strings.Contains(joined, "netpol_default_deny") {
		t.Error("expected network policy warning")
	}
}
