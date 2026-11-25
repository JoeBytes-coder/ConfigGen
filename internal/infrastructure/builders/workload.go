package builders

import (
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"configgen/internal/domain"
)

func init() {
	register("Deployment", buildDeployment)
	register("StatefulSet", buildStatefulSet)
	register("DaemonSet", buildDaemonSet)
	register("ReplicaSet", buildReplicaSet)
	register("Job", buildJob)
	register("CronJob", buildCronJob)
}

func podTemplate(app domain.AppContext, n domain.Node, withPorts bool) corev1.PodTemplateSpec {
	return corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{Labels: selectorLabels(app)},
		Spec:       podSpec(app, n, withPorts),
	}
}

func replicasOrDefault(app domain.AppContext) int32 {
	if app.Replicas < 1 {
		return 1
	}
	return app.Replicas
}

func buildDeployment(app domain.AppContext, n domain.Node) (any, error) {
	return &appsv1.Deployment{
		TypeMeta:   metav1.TypeMeta{APIVersion: "apps/v1", Kind: "Deployment"},
		ObjectMeta: objectMeta(app, n),
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(replicasOrDefault(app)),
			Selector: &metav1.LabelSelector{MatchLabels: selectorLabels(app)},
			Template: podTemplate(app, n, true),
		},
	}, nil
}

func buildStatefulSet(app domain.AppContext, n domain.Node) (any, error) {
	serviceName := app.Name + "-service"
	if ref, ok := n.Refs["service"]; ok {
		serviceName = ref.Name
	}
	return &appsv1.StatefulSet{
		TypeMeta:   metav1.TypeMeta{APIVersion: "apps/v1", Kind: "StatefulSet"},
		ObjectMeta: objectMeta(app, n),
		Spec: appsv1.StatefulSetSpec{
			ServiceName: serviceName,
			Replicas:    int32Ptr(replicasOrDefault(app)),
			Selector:    &metav1.LabelSelector{MatchLabels: selectorLabels(app)},
			Template:    podTemplate(app, n, true),
		},
	}, nil
}

func buildDaemonSet(app domain.AppContext, n domain.Node) (any, error) {
	return &appsv1.DaemonSet{
		TypeMeta:   metav1.TypeMeta{APIVersion: "apps/v1", Kind: "DaemonSet"},
		ObjectMeta: objectMeta(app, n),
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{MatchLabels: selectorLabels(app)},
			Template: podTemplate(app, n, true),
		},
	}, nil
}

func buildReplicaSet(app domain.AppContext, n domain.Node) (any, error) {
	return &appsv1.ReplicaSet{
		TypeMeta:   metav1.TypeMeta{APIVersion: "apps/v1", Kind: "ReplicaSet"},
		ObjectMeta: objectMeta(app, n),
		Spec: appsv1.ReplicaSetSpec{
			Replicas: int32Ptr(replicasOrDefault(app)),
			Selector: &metav1.LabelSelector{MatchLabels: selectorLabels(app)},
			Template: podTemplate(app, n, true),
		},
	}, nil
}

func buildJob(app domain.AppContext, n domain.Node) (any, error) {
	backoff := int32(3)
	template := podTemplate(app, n, false)
	template.Spec.RestartPolicy = corev1.RestartPolicyNever
	return &batchv1.Job{
		TypeMeta:   metav1.TypeMeta{APIVersion: "batch/v1", Kind: "Job"},
		ObjectMeta: objectMeta(app, n),
		Spec: batchv1.JobSpec{
			BackoffLimit: &backoff,
			Template:     template,
		},
	}, nil
}

func buildCronJob(app domain.AppContext, n domain.Node) (any, error) {
	template := podTemplate(app, n, false)
	template.Spec.RestartPolicy = corev1.RestartPolicyNever
	return &batchv1.CronJob{
		TypeMeta:   metav1.TypeMeta{APIVersion: "batch/v1", Kind: "CronJob"},
		ObjectMeta: objectMeta(app, n),
		Spec: batchv1.CronJobSpec{
			Schedule: "0 * * * *",
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{Template: template},
			},
		},
	}, nil
}
