package domain

import "time"

// ConfigRequest 配置请求结构体
// 字段按生成类型分组：
//
//	[通用] Type, AppName, Image, Tag, Port, Env
//	[K8s]  Replicas, K8sResource
//	[Dockerfile] DockerfileBaseImage, DockerfileWorkDir, DockerfileCmd
type ConfigRequest struct {
	Type    string            `json:"type" binding:"required,oneof=compose k8s dockerfile kustomize" validate:"required,oneof=compose k8s dockerfile kustomize"`
	AppName string            `json:"app_name" binding:"required" validate:"required,min=1,max=50"`
	Image   string            `json:"image" binding:"required" validate:"required,min=1,max=100"`
	Tag     string            `json:"tag" binding:"required" validate:"required,min=1,max=50"`
	Port    int               `json:"port" binding:"required,min=1,max=65535" validate:"min=0,max=65535"`
	Env     map[string]string `json:"env" validate:"omitempty"`

	Replicas    int      `json:"replicas" validate:"omitempty,min=1,max=100"`
	K8sResource []string `json:"k8s_resource" validate:"omitempty,dive,oneof=Deployment Service ConfigMap Namespace Ingress StatefulSet PersistentVolumeClaim Secret DaemonSet CronJob ServiceAccount HorizontalPodAutoscaler ResourceQuota LimitRange NetworkPolicy Role RoleBinding ClusterRole ClusterRoleBinding StorageClass"`

	DockerfileBaseImage string   `json:"dockerfile_base_image" validate:"omitempty,min=1,max=100"`
	DockerfileWorkDir   string   `json:"dockerfile_workdir" validate:"omitempty,min=1,max=100"`
	DockerfileCmd       []string `json:"dockerfile_cmd" validate:"omitempty,dive,min=1,max=100"`
}

// ConfigResult 配置结果结构体
type ConfigResult struct {
	Type   string `json:"type"`
	Config string `json:"config"`
}

// ConfigRecord 配置记录结构体
type ConfigRecord struct {
	ID        int64         `json:"id"`
	Request   ConfigRequest `json:"request"`
	Result    ConfigResult  `json:"result"`
	CreatedAt time.Time     `json:"created_at"`
}
