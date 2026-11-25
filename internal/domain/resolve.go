package domain

import (
	"fmt"
	"sort"
	"strings"
)

type resolver struct {
	plan     Plan
	present  map[string]bool
	auto     map[string]bool
	diags    []Diagnostic
	conflict bool
}

// Resolve 把一个 Selection 解析为可渲染的 Plan：
// 校验 -> 依赖闭包 -> 工作负载规则 -> 默认值 -> 命名 -> 引用绑定 -> 排序。
func Resolve(sel Selection) (Plan, error) {
	if !sel.Type.Valid() {
		return Plan{}, NewBadRequest("不支持的配置类型: " + string(sel.Type))
	}

	r := &resolver{
		plan: Plan{
			Type: TypeK8s, App: normalizeApp(sel.App),
			Nodes: []Node{}, Edges: []Edge{}, Diagnostics: []Diagnostic{},
			CatalogVersion: CatalogVersion,
		},
		present: map[string]bool{},
		auto:    map[string]bool{},
	}

	r.resolveK8s(sel)

	r.plan.Diagnostics = r.diags
	r.plan.Valid = !r.hasError()
	if r.hasError() {
		kind := ErrBadRequest
		msg := "配置校验失败"
		if r.conflict {
			kind = ErrUnprocessable
			msg = "资源依赖无法满足"
		}
		return r.plan, &AppError{Kind: kind, Message: msg, Diagnostics: r.diags}
	}
	return r.plan, nil
}

func normalizeApp(a AppContext) AppContext {
	a.Name = strings.TrimSpace(a.Name)
	a.Namespace = strings.TrimSpace(a.Namespace)
	a.Image = strings.TrimSpace(a.Image)
	a.Tag = strings.TrimSpace(a.Tag)
	a.Host = strings.TrimSpace(a.Host)
	a.TLSSecret = strings.TrimSpace(a.TLSSecret)
	for i := range a.Ports {
		p := &a.Ports[i]
		p.Name = strings.TrimSpace(p.Name)
		if p.Name == "" {
			if i == 0 {
				p.Name = "http"
			} else {
				p.Name = fmt.Sprintf("http-%d", i+1)
			}
		}
		if p.Container == 0 {
			p.Container = p.Service
		}
		if p.Service == 0 {
			p.Service = p.Container
		}
	}
	return a
}

func (r *resolver) resolveK8s(sel Selection) {
	r.plan.Type = TypeK8s
	if len(sel.Resources) == 0 {
		r.addError("no_resources", "", "至少选择一种 Kubernetes 资源")
		return
	}

	seen := map[string]bool{}
	for _, raw := range sel.Resources {
		kind := strings.TrimSpace(raw)
		if kind == "" {
			continue
		}
		if _, ok := SpecByKind(kind); !ok {
			r.addError("unknown_kind", kind, "不支持的 Kubernetes 资源: "+kind)
			continue
		}
		if seen[kind] {
			r.diags = append(r.diags, Diagnostic{Severity: SeverityInfo, Code: "duplicate_kind", Kind: kind, Message: kind + " 重复选择，已去重"})
			continue
		}
		seen[kind] = true
		r.present[kind] = true
	}
	if len(r.present) == 0 {
		r.addError("no_resources", "", "至少选择一种 Kubernetes 资源")
		return
	}
	if r.hasError() {
		return
	}

	r.expand()
	r.workloadRules()
	r.applyDefaults()
	r.validateK8s()
	if r.hasError() {
		return
	}

	names := r.allocateNames()
	ns := r.resolveNamespace(names)
	nodes := r.buildNodes(names, ns)
	sort.SliceStable(nodes, func(i, j int) bool {
		if nodes[i].Level != nodes[j].Level {
			return nodes[i].Level < nodes[j].Level
		}
		if nodes[i].Kind != nodes[j].Kind {
			return nodes[i].Kind < nodes[j].Kind
		}
		return nodes[i].Name < nodes[j].Name
	})
	r.plan.Nodes = nodes
	r.plan.Edges = buildEdges(nodes)
	r.semanticDiags(ns)
}

func (r *resolver) expand() {
	for i := 0; i < 12; i++ {
		added := false
		for _, kind := range sortedKeys(r.present) {
			spec, ok := SpecByKind(kind)
			if !ok {
				continue
			}
			for _, req := range spec.Requires {
				if r.present[req.Kind] {
					continue
				}
				if req.Mode == ModeAutoAdd {
					r.present[req.Kind] = true
					r.auto[req.Kind] = true
					r.diags = append(r.diags, Diagnostic{
						Severity: SeverityInfo, Code: "auto_added", Kind: req.Kind,
						Message: "已自动补充 " + req.Kind, Hint: kind + " 依赖它",
					})
					added = true
				} else {
					r.addError("missing_dependency", kind, kind+" 需要 "+req.Kind)
				}
			}
		}
		if !added {
			return
		}
	}
}

func (r *resolver) workloadRules() {
	if len(r.plan.App.SecretEnv) > 0 {
		r.addAuto("Secret", "检测到密钥配置，自动补充 Secret")
	}
	if r.present["StatefulSet"] && !r.present["Service"] {
		r.addAuto("Service", "StatefulSet 需要 Headless Service")
	}
	if r.present["Service"] && r.primaryWorkload() == "" {
		r.addAuto("Deployment", "Service 需要工作负载提供 Endpoints")
	}
	if r.present["HorizontalPodAutoscaler"] {
		scalable := r.scalableWorkloads()
		switch {
		case len(scalable) == 0:
			if r.primaryWorkload() != "" {
				r.addConflict("hpa_unsupported_target", "HorizontalPodAutoscaler", "HPA 只能指向 Deployment / StatefulSet / ReplicaSet")
			} else {
				r.addAuto("Deployment", "HPA 需要可伸缩目标")
			}
		case len(scalable) > 1:
			r.addConflict("hpa_ambiguous_target", "HorizontalPodAutoscaler", "同时存在多个可伸缩工作负载，HPA 目标不明确")
		}
	}
	for _, kind := range []string{"PodDisruptionBudget", "NetworkPolicy"} {
		if r.present[kind] && r.primaryWorkload() == "" {
			r.addAuto("Deployment", kind+" 需要工作负载")
		}
	}
	if r.present["RoleBinding"] {
		if !r.present["Role"] && !r.present["ClusterRole"] {
			r.addAuto("Role", "RoleBinding 默认绑定最小权限 Role")
		}
		if r.present["Role"] && r.present["ClusterRole"] {
			r.diags = append(r.diags, Diagnostic{
				Severity: SeverityWarning, Code: "role_ref_ambiguous", Kind: "RoleBinding",
				Message: "同时选中 Role 与 ClusterRole，RoleBinding 将绑定 Role",
			})
		}
	}
}

func (r *resolver) addAuto(kind, hint string) {
	if r.present[kind] {
		return
	}
	r.present[kind] = true
	r.auto[kind] = true
	r.diags = append(r.diags, Diagnostic{
		Severity: SeverityInfo, Code: "auto_added", Kind: kind,
		Message: "已自动补充 " + kind, Hint: hint,
	})
}

func (r *resolver) applyDefaults() {
	needsImage, needsPort, needsReplicas := false, false, false
	for kind := range r.present {
		if spec, ok := SpecByKind(kind); ok {
			needsImage = needsImage || spec.Needs.Image
			needsPort = needsPort || spec.Needs.Port
			needsReplicas = needsReplicas || spec.Needs.Replicas
		}
	}
	if needsImage {
		if r.plan.App.Image == "" {
			r.plan.App.Image = "nginx"
		}
		if r.plan.App.Tag == "" {
			r.plan.App.Tag = "latest"
		}
	}
	if needsPort && len(r.plan.App.Ports) == 0 {
		r.plan.App.Ports = []Port{{Name: "http", Container: 80, Service: 80}}
	}
	if needsReplicas && r.plan.App.Replicas < 1 {
		r.plan.App.Replicas = 1
	}
}

func (r *resolver) primaryWorkload() string {
	for _, kind := range WorkloadKinds {
		if r.present[kind] {
			return kind
		}
	}
	return ""
}

func (r *resolver) scalableWorkloads() []string {
	var out []string
	for _, kind := range ScalableKinds {
		if r.present[kind] {
			out = append(out, kind)
		}
	}
	return out
}

func (r *resolver) hpaTarget() string {
	scalable := r.scalableWorkloads()
	if len(scalable) == 1 {
		return scalable[0]
	}
	return ""
}

func (r *resolver) workloadList() []string {
	var out []string
	for _, kind := range WorkloadKinds {
		if r.present[kind] {
			out = append(out, kind)
		}
	}
	return out
}

func (r *resolver) hasLongRunningWorkload() bool {
	for _, kind := range []string{"Deployment", "StatefulSet", "DaemonSet", "ReplicaSet"} {
		if r.present[kind] {
			return true
		}
	}
	return false
}

func (r *resolver) allocateNames() map[string]string {
	names := make(map[string]string, len(r.present))
	for kind := range r.present {
		spec, _ := SpecByKind(kind)
		if kind == "Namespace" && r.plan.App.Namespace != "" {
			names[kind] = r.plan.App.Namespace
			continue
		}
		names[kind] = r.plan.App.Name + "-" + spec.Suffix
	}
	return names
}

func (r *resolver) resolveNamespace(names map[string]string) string {
	ns := r.plan.App.Namespace
	if r.present["Namespace"] {
		if ns == "" {
			ns = names["Namespace"]
			r.plan.App.Namespace = ns
			r.diags = append(r.diags, Diagnostic{Severity: SeverityInfo, Code: "namespace_created", Kind: "Namespace", Message: "将创建命名空间 " + ns})
		} else {
			names["Namespace"] = ns
		}
	} else if ns != "" {
		r.diags = append(r.diags, Diagnostic{Severity: SeverityInfo, Code: "namespace_external", Message: "命名空间 " + ns + " 需要已存在，本工具不会创建它"})
	}
	return ns
}

func (r *resolver) buildNodes(names map[string]string, ns string) []Node {
	primary := r.primaryWorkload()
	target := r.hpaTarget()
	nodes := make([]Node, 0, len(names))
	for kind := range r.present {
		spec, _ := SpecByKind(kind)
		node := Node{Kind: kind, Name: names[kind], Level: spec.Level, AutoAdded: r.auto[kind]}
		if spec.Scope == Namespaced && ns != "" {
			node.Namespace = ns
		}
		node.Refs = r.bindRefs(kind, names, primary, target)
		nodes = append(nodes, node)
	}
	return nodes
}

func (r *resolver) bindRefs(kind string, names map[string]string, primary, hpaTarget string) map[string]Ref {
	refs := map[string]Ref{}
	add := func(target, key string) {
		if name, ok := names[target]; ok {
			refs[key] = Ref{Kind: target, Name: name}
		}
	}
	switch kind {
	case "Deployment", "StatefulSet", "DaemonSet", "CronJob", "Job", "ReplicaSet":
		add("ConfigMap", "configmap")
		add("Secret", "secret")
		add("PersistentVolumeClaim", "pvc")
		add("ServiceAccount", "sa")
		if kind == "StatefulSet" {
			add("Service", "service")
		}
	case "Service":
		if primary != "" {
			refs["workload"] = Ref{Kind: primary, Name: names[primary]}
		}
	case "Ingress":
		add("Service", "service")
		add("IngressClass", "ingressclass")
	case "HorizontalPodAutoscaler":
		if hpaTarget != "" {
			refs["workload"] = Ref{Kind: hpaTarget, Name: names[hpaTarget]}
		}
	case "PodDisruptionBudget", "NetworkPolicy":
		if primary != "" {
			refs["workload"] = Ref{Kind: primary, Name: names[primary]}
		}
	case "EndpointSlice":
		add("Service", "service")
	case "PersistentVolumeClaim":
		add("StorageClass", "storageclass")
	case "RoleBinding":
		add("ServiceAccount", "sa")
		if _, ok := r.present["Role"]; ok {
			add("Role", "role")
		} else {
			add("ClusterRole", "clusterrole")
		}
	case "ClusterRoleBinding":
		add("ServiceAccount", "sa")
		add("ClusterRole", "clusterrole")
	}
	if len(refs) == 0 {
		return nil
	}
	return refs
}

func buildEdges(nodes []Node) []Edge {
	seen := map[string]bool{}
	var edges []Edge
	for _, node := range nodes {
		keys := make([]string, 0, len(node.Refs))
		for key := range node.Refs {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			ref := node.Refs[key]
			relation := relationFor(node.Kind, key)
			sig := node.Kind + "|" + ref.Kind + "|" + relation
			if seen[sig] {
				continue
			}
			seen[sig] = true
			edges = append(edges, Edge{From: node.Kind, To: ref.Kind, Relation: relation})
		}
	}
	return edges
}

func relationFor(sourceKind, refKey string) string {
	switch refKey {
	case "workload":
		switch sourceKind {
		case "Service":
			return "selects"
		case "HorizontalPodAutoscaler":
			return "scales"
		case "PodDisruptionBudget":
			return "protects"
		case "NetworkPolicy":
			return "targets"
		}
	case "service":
		return "routes-to"
	case "configmap", "secret", "pvc":
		return "mounts"
	case "sa":
		if sourceKind == "RoleBinding" || sourceKind == "ClusterRoleBinding" {
			return "grants-to"
		}
		return "uses-sa"
	case "storageclass", "ingressclass":
		return "uses-class"
	case "role", "clusterrole":
		return "binds"
	case "namespace":
		return "in"
	}
	return refKey
}

func (r *resolver) semanticDiags(ns string) {
	add := func(severity Severity, code, kind, message, hint string) {
		r.diags = append(r.diags, Diagnostic{Severity: severity, Code: code, Kind: kind, Message: message, Hint: hint})
	}
	if r.present["NetworkPolicy"] {
		add(SeverityWarning, "netpol_default_deny", "NetworkPolicy", "已生成默认拒绝策略（含 DNS 放行），请确认业务出站流量", "按需补充 egress 规则")
	}
	if r.present["StorageClass"] {
		add(SeverityWarning, "placeholder_provisioner", "StorageClass", "StorageClass 使用 no-provisioner，PVC 不会动态供给", "请按集群实际 provisioner 修改")
	}
	if r.present["IngressClass"] {
		add(SeverityWarning, "placeholder_controller", "IngressClass", "IngressClass 使用 ingress-nginx 控制器标识", "请确认与集群实际控制器一致")
	}
	if r.present["Secret"] && len(r.plan.App.SecretEnv) == 0 {
		add(SeverityWarning, "secret_empty", "Secret", "Secret 内容为空，请部署前填充", "在密钥配置中填写键值，或部署后手动修改")
	}
	if r.present["ConfigMap"] && len(r.plan.App.Env) == 0 {
		add(SeverityWarning, "configmap_empty", "ConfigMap", "ConfigMap 内容为空（未配置环境变量）", "")
	}
	if r.present["HorizontalPodAutoscaler"] && r.plan.App.Resources.Requests["cpu"] == "" {
		add(SeverityWarning, "hpa_no_requests", "HorizontalPodAutoscaler", "未设置 CPU requests，基于利用率的 HPA 无法工作", "为应用配置 resources.requests.cpu")
	}
	if r.present["PodDisruptionBudget"] && r.plan.App.Replicas <= 1 {
		add(SeverityWarning, "pdb_single_replica", "PodDisruptionBudget", "副本数为 1 时 PDB 使用 maxUnavailable=1", "生产环境建议至少 2 副本")
	}
	if r.present["Service"] && (r.present["Job"] || r.present["CronJob"]) {
		add(SeverityWarning, "service_job", "Service", "Service 选择的是 Job/CronJob 的 Pod，Pod 结束后 Endpoints 会消失", "")
	}
	if r.present["Ingress"] && r.plan.App.Host == "" {
		add(SeverityInfo, "ingress_any_host", "Ingress", "未填写域名，Ingress 将匹配任意 Host", "设置域名以获得可预期路由")
	}
	if r.plan.App.TLSSecret != "" && r.plan.App.Host == "" {
		add(SeverityWarning, "tls_without_host", "Ingress", "配置了 TLS 但未填写域名，TLS 段不会生成", "")
	}
	if r.plan.App.Host != "" && !r.present["Ingress"] {
		add(SeverityWarning, "host_ignored", "Ingress", "已填写域名但未选择 Ingress，域名不会生效", "勾选 Ingress 以生成路由")
	}
	if r.plan.App.TLSSecret != "" && !r.present["Ingress"] {
		add(SeverityWarning, "tls_ignored", "Ingress", "已填写 TLS Secret 但未选择 Ingress，TLS 不会生效", "")
	}
	if r.plan.App.Probe.Enabled {
		switch {
		case r.primaryWorkload() == "":
			add(SeverityWarning, "probe_ignored_no_workload", "", "已开启探针但未选择工作负载，探针不会生成", "选择 Deployment/StatefulSet/DaemonSet 等长驻工作负载")
		case !r.hasLongRunningWorkload():
			add(SeverityWarning, "probe_ignored_batch", "", "Job/CronJob 不生成探针，探针配置已忽略", "")
		case len(r.plan.App.Ports) == 0:
			add(SeverityWarning, "probe_ignored_no_ports", "", "已开启探针但未配置端口，探针不会生成", "填写端口或关闭探针")
		}
	}
	if r.present["CronJob"] {
		add(SeverityWarning, "cronjob_default_schedule", "CronJob", "CronJob 使用固定 schedule 0 * * * *（每小时），请按需修改", "")
	}
	if r.present["PersistentVolumeClaim"] {
		add(SeverityWarning, "pvc_default_size", "PersistentVolumeClaim", "PVC 使用固定 10Gi 存储，请按需修改", "")
	}
	if r.present["EndpointSlice"] {
		add(SeverityWarning, "endpointslice_empty", "EndpointSlice", "EndpointSlice 的 endpoints 为空，通常应由控制器管理", "确认是否真的需要手写 EndpointSlice")
	}
	if r.present["ResourceQuota"] {
		add(SeverityWarning, "quota_placeholder", "ResourceQuota", "ResourceQuota 使用固定配额（4CPU/8Gi 请求，8CPU/16Gi 限制）", "按集群实际配额修改")
	}
	if r.present["LimitRange"] {
		add(SeverityWarning, "limitrange_placeholder", "LimitRange", "LimitRange 使用固定默认值（500m/512Mi）", "按集群实际策略修改")
	}
	if r.present["PriorityClass"] {
		add(SeverityWarning, "priorityclass_placeholder", "PriorityClass", "PriorityClass 使用固定 value 1000", "按调度策略修改")
	}
	workloads := r.workloadList()
	if len(workloads) > 1 {
		add(SeverityWarning, "multiple_workloads", "", "同时选择多个工作负载，它们共享 app 标签，Service/NetworkPolicy 会同时选中", "建议只保留一个工作负载")
		if r.present["StatefulSet"] && r.present["Service"] {
			add(SeverityWarning, "statefulset_service_not_headless", "StatefulSet", "StatefulSet 的 Service 不是 Headless，稳定网络标识会失效", "为 StatefulSet 单独提供 Headless Service")
		}
	}
	if ns == "" && len(r.plan.Nodes) > 0 {
		add(SeverityInfo, "namespace_default", "", "未显式指定命名空间，资源将部署到集群默认命名空间", "")
	}
}

func sortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
