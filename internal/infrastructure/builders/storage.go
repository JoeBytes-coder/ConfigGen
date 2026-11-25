package builders

import (
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"configgen/internal/domain"
)

func init() {
	register("PersistentVolumeClaim", buildPVC)
	register("StorageClass", buildStorageClass)
}

func buildPVC(app domain.AppContext, n domain.Node) (any, error) {
	spec := corev1.PersistentVolumeClaimSpec{
		AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
		Resources: corev1.VolumeResourceRequirements{
			Requests: corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("10Gi")},
		},
	}
	if ref, ok := n.Refs["storageclass"]; ok {
		name := ref.Name
		spec.StorageClassName = &name
	}
	return &corev1.PersistentVolumeClaim{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "PersistentVolumeClaim"},
		ObjectMeta: objectMeta(app, n),
		Spec:       spec,
	}, nil
}

func buildStorageClass(app domain.AppContext, n domain.Node) (any, error) {
	reclaim := corev1.PersistentVolumeReclaimRetain
	binding := storagev1.VolumeBindingWaitForFirstConsumer
	return &storagev1.StorageClass{
		TypeMeta:          metav1.TypeMeta{APIVersion: "storage.k8s.io/v1", Kind: "StorageClass"},
		ObjectMeta:        metav1.ObjectMeta{Name: n.Name, Labels: labels(app)},
		Provisioner:       "kubernetes.io/no-provisioner",
		ReclaimPolicy:     &reclaim,
		VolumeBindingMode: &binding,
	}, nil
}
