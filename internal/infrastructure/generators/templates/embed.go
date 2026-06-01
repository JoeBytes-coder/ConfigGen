package templates

import "embed"

//go:embed compose/*.tmpl
var ComposeFS embed.FS

//go:embed k8s/*.tmpl
var K8sFS embed.FS

//go:embed dockerfile/*.tmpl
var DockerfileFS embed.FS

//go:embed kustomize/*.tmpl
var KustomizeFS embed.FS
