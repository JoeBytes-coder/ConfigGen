package domain

import (
	"fmt"
	"sort"
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"
)

func isDNSLabel(s string, max int) bool {
	if s == "" || len(s) > max {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		ok := (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-'
		if !ok {
			return false
		}
	}
	return s[0] != '-' && s[len(s)-1] != '-'
}

func validImagePart(s string) bool {
	return s != "" && !strings.ContainsAny(s, " \t\r\n")
}

func isEnvKey(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		ok := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.'
		if !ok {
			return false
		}
	}
	return true
}

func (r *resolver) addError(code, kind, message string) {
	r.diags = append(r.diags, Diagnostic{Severity: SeverityError, Code: code, Kind: kind, Message: message})
}

func (r *resolver) addConflict(code, kind, message string) {
	r.conflict = true
	r.addError(code, kind, message)
}

func (r *resolver) hasError() bool {
	for _, d := range r.diags {
		if d.Severity == SeverityError {
			return true
		}
	}
	return false
}

func (r *resolver) validateK8s() {
	app := &r.plan.App
	if !isDNSLabel(app.Name, 44) {
		r.addError("invalid_app_name", "", "应用名称必须是 1-44 位小写字母、数字或连字符（DNS-1123）")
	}
	if app.Namespace != "" && !isDNSLabel(app.Namespace, 63) {
		r.addError("invalid_namespace", "", "命名空间必须符合 DNS-1123 规范")
	}

	needsImage := false
	for kind := range r.present {
		if spec, ok := SpecByKind(kind); ok && spec.Needs.Image {
			needsImage = true
		}
	}
	if needsImage {
		if app.Image == "" {
			r.addError("image_required", "", "工作负载需要镜像")
		} else if !validImagePart(app.Image) {
			r.addError("invalid_image", "", "镜像名称不能包含空白字符")
		}
		if app.Tag == "" {
			r.addError("tag_required", "", "镜像标签不能为空")
		} else if !validImagePart(app.Tag) {
			r.addError("invalid_tag", "", "镜像标签不能包含空白字符")
		}
	}

	portNames := map[string]bool{}
	for _, p := range app.Ports {
		if p.Container < 1 || p.Container > 65535 {
			r.addError("invalid_port", "", fmt.Sprintf("容器端口 %d 超出 1-65535", p.Container))
		}
		if p.Service < 1 || p.Service > 65535 {
			r.addError("invalid_port", "", fmt.Sprintf("服务端口 %d 超出 1-65535", p.Service))
		}
		if !isDNSLabel(p.Name, 15) {
			r.addError("invalid_port_name", "", "端口名必须是 1-15 位小写字母、数字或连字符")
		}
		if portNames[p.Name] {
			r.addError("duplicate_port_name", "", "端口名重复: "+p.Name)
		}
		portNames[p.Name] = true
	}

	for key := range app.Env {
		if !isEnvKey(key) {
			r.addError("invalid_env_key", "", "环境变量名不合法: "+key)
		}
	}
	for key := range app.SecretEnv {
		if !isEnvKey(key) {
			r.addError("invalid_env_key", "", "密钥名不合法: "+key)
		}
	}
	if app.Host != "" && strings.ContainsAny(app.Host, " \t\r\n") {
		r.addError("invalid_host", "", "域名不能包含空白字符")
	}
	if app.TLSSecret != "" && !isDNSLabel(app.TLSSecret, 63) {
		r.addError("invalid_tls_secret", "", "TLS Secret 名称不合法")
	}
	if msg := validateResourceQuantities(app); msg != "" {
		r.addError("invalid_resource_quantity", "", msg)
	}
}

func validateResourceQuantities(app *AppContext) string {
	check := func(label string, values map[string]string) string {
		keys := make([]string, 0, len(values))
		for key := range values {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			value := strings.TrimSpace(values[key])
			if value == "" {
				return fmt.Sprintf("%s %s 的值为空", label, key)
			}
			if _, err := resource.ParseQuantity(value); err != nil {
				return fmt.Sprintf("%s %s=%q 不是合法的 quantity", label, key, values[key])
			}
		}
		return ""
	}
	if msg := check("requests", app.Resources.Requests); msg != "" {
		return msg
	}
	return check("limits", app.Resources.Limits)
}
