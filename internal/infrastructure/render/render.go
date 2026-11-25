package render

import (
	"fmt"
	"strings"

	sigsyaml "sigs.k8s.io/yaml"

	"configgen/internal/domain"
	"configgen/internal/infrastructure/builders"
)

// Render 把 Plan 渲染为多文件产物与一个可直接 apply 的 bundle。
func Render(plan domain.Plan) ([]domain.Artifact, string, error) {
	files := make([]domain.Artifact, 0, len(plan.Nodes))
	for _, node := range plan.Nodes {
		obj, err := builders.Build(plan.App, node)
		if err != nil {
			return nil, "", fmt.Errorf("%s/%s: %w", node.Kind, node.Name, err)
		}
		data, err := sigsyaml.Marshal(obj)
		if err != nil {
			return nil, "", fmt.Errorf("marshal %s/%s: %w", node.Kind, node.Name, err)
		}
		content := string(data)
		if !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		files = append(files, domain.Artifact{
			Path:    strings.ToLower(node.Kind) + ".yaml",
			Kind:    node.Kind,
			Content: content,
		})
	}

	if err := validateK8sArtifacts(files); err != nil {
		return nil, "", err
	}

	var bundle strings.Builder
	for _, file := range files {
		bundle.WriteString("---\n")
		bundle.WriteString(file.Content)
	}
	return files, bundle.String(), nil
}

// validateK8sArtifacts 是「输出即验收」的闸门：合法 YAML + kind/name 齐全 + 无重复。
func validateK8sArtifacts(files []domain.Artifact) error {
	seen := map[string]bool{}
	for _, file := range files {
		var obj map[string]any
		if err := sigsyaml.Unmarshal([]byte(file.Content), &obj); err != nil {
			return fmt.Errorf("%s: 生成的 YAML 非法: %w", file.Path, err)
		}
		kind, _ := obj["kind"].(string)
		metadata, _ := obj["metadata"].(map[string]any)
		name, _ := metadata["name"].(string)
		if kind == "" || name == "" {
			return fmt.Errorf("%s: 缺少 kind 或 metadata.name", file.Path)
		}
		key := kind + "/" + name
		if seen[key] {
			return fmt.Errorf("重复资源: %s", key)
		}
		seen[key] = true
	}
	return nil
}
