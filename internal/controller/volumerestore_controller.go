/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"strconv"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	snapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v6/apis/volumesnapshot/v1"
	backupv1alpha1 "github.com/rishabh625/kube-application-snapshot/api/v1alpha1"
	"github.com/rishabh625/kube-application-snapshot/util"
)

// VolumeRestoreReconciler reconciles a VolumeRestore object
type VolumeRestoreReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=backup.infracloud-citadel-vbakup.com,resources=volumerestores,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=backup.infracloud-citadel-vbakup.com,resources=volumerestores/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=backup.infracloud-citadel-vbakup.com,resources=volumerestores/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the VolumeRestore object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.4/pkg/reconcile
func (r *VolumeRestoreReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// TODO(user): your logic here
	var vr backupv1alpha1.VolumeRestore
	if err := r.Get(ctx, req.NamespacedName, &vr); err != nil {
		if errors.IsNotFound(err) {
			log.Info("VolumeRestore not found, it might have been deleted")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	switch vr.Status.Phase {
	case util.COMPLETED, util.FAILED:
		// Optionally, you might check if spec changed; for now, no reconciliation is performed.
		log.Info("VolumeRestore is already completed or failed", "phase", vr.Status.Phase)
		return ctrl.Result{}, nil

	case "", "Pending":
		return r.handlePending(ctx, &vr)

	case "CreatingPVC":
		return r.handleCreatingPVC(ctx, &vr)

	default:
		log.Info("Unknown phase; requeuing", "phase", vr.Status.Phase)
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *VolumeRestoreReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&backupv1alpha1.VolumeRestore{}).
		Named("volumerestore").
		Complete(r)
}

func (r *VolumeRestoreReconciler) handlePending(ctx context.Context, vr *backupv1alpha1.VolumeRestore) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("Handling pending VolumeRestore", "name", vr.Name)
	// 1. Fetch the source VolumeSnapshot
	var snapshot snapshotv1.VolumeSnapshot
	snapshotKey := types.NamespacedName{
		Namespace: vr.Spec.SourceSnapshotNamespace,
		Name:      vr.Spec.SourceSnapshotName,
	}
	if err := r.Get(ctx, snapshotKey, &snapshot); err != nil {
		if errors.IsNotFound(err) {
			vr.Status.Phase = util.FAILED
			vr.Status.Message = fmt.Sprintf("Source VolumeSnapshot %s not found in namespace %s", vr.Spec.SourceSnapshotName, vr.Spec.SourceSnapshotNamespace)
			_ = r.Status().Update(ctx, vr)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// 2. Check if the snapshot is ready
	if snapshot.Status == nil || snapshot.Status.ReadyToUse == nil || !*snapshot.Status.ReadyToUse {
		vr.Status.Phase = "PendingSnapshot"
		vr.Status.Message = "Source VolumeSnapshot is not ready yet"
		_ = r.Status().Update(ctx, vr)
		// Requeue to check again later
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	// 3. Check if target PVC already exists
	// Determine the target namespace; default to the VolumeRestore namespace if not provided
	targetNamespace := vr.Spec.TargetPvcNamespace
	if targetNamespace == "" {
		targetNamespace = vr.Namespace
	}
	var pvc corev1.PersistentVolumeClaim
	pvcKey := types.NamespacedName{Name: vr.Spec.TargetPvcName, Namespace: targetNamespace}
	if err := r.Get(ctx, pvcKey, &pvc); err == nil {

		if vr.Spec.OverwritePVC {
			// Delete existing PVC
			if err := r.Delete(ctx, &pvc); err != nil {
				vr.Status.Phase = util.FAILED
				vr.Status.Message = fmt.Sprintf("Failed to delete existing PVC: %v", err)
				_ = r.Status().Update(ctx, vr)
				return ctrl.Result{}, err
			}
			// Requeue to continue after PVC is deleted
			return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
		} else {
			vr.Status.Phase = util.FAILED
			vr.Status.Message = fmt.Sprintf("Target PVC %s already exists in namespace %s", vr.Spec.TargetPvcName, targetNamespace)
			_ = r.Status().Update(ctx, vr)
			return ctrl.Result{}, nil
		}
	} else if err != nil && !errors.IsNotFound(err) {
		return ctrl.Result{}, err
	}
	// 4. Fetch the VolumeSnapshotContent to get the snapshot size
	var vsc snapshotv1.VolumeSnapshotContent
	if snapshot.Status.BoundVolumeSnapshotContentName == nil {
		vr.Status.Phase = util.FAILED
		vr.Status.Message = "VolumeSnapshot does not have a bound VolumeSnapshotContent"
		_ = r.Status().Update(ctx, vr)
		return ctrl.Result{}, nil
	}
	vscKey := types.NamespacedName{Name: *snapshot.Status.BoundVolumeSnapshotContentName}
	if err := r.Get(ctx, vscKey, &vsc); err != nil {
		vr.Status.Phase = util.FAILED
		vr.Status.Message = "Unable to fetch VolumeSnapshotContent"
		_ = r.Status().Update(ctx, vr)
		return ctrl.Result{}, err
	}
	if vsc.Status == nil || vsc.Status.RestoreSize == nil {
		vr.Status.Phase = util.FAILED
		vr.Status.Message = "VolumeSnapshotContent is missing restore size information"
		_ = r.Status().Update(ctx, vr)
		return ctrl.Result{}, nil
	}
	// Use the snapshot restore size
	size := *vsc.Status.RestoreSize
	s := strconv.Itoa(int(size))
	// 5. Construct the target PVC
	newPVC := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vr.Spec.TargetPvcName,
			Namespace: targetNamespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}, // Adjust as needed
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					"storage": resource.MustParse(s),
				},
			},
			DataSource: &corev1.TypedLocalObjectReference{
				APIGroup: func() *string {
					s := "snapshot.storage.k8s.io"
					return &s
				}(),
				Kind: "VolumeSnapshot",
				Name: vr.Spec.SourceSnapshotName,
			},
		},
	}
	// Optionally set StorageClass if specified
	if vr.Spec.TargetStorageClassName != nil {
		newPVC.Spec.StorageClassName = vr.Spec.TargetStorageClassName
	}

	// Set OwnerReference so that this PVC is tracked by the VolumeRestore CR
	if err := controllerutil.SetControllerReference(vr, newPVC, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}

	// 6. Create the PVC
	if err := r.Create(ctx, newPVC); err != nil {
		vr.Status.Phase = util.FAILED
		vr.Status.Message = fmt.Sprintf("Failed to create PVC: %v", err)
		_ = r.Status().Update(ctx, vr)
		return ctrl.Result{}, err
	}

	vr.Status.Phase = "CreatingPVC"
	vr.Status.Message = "PVC creation initiated"
	vr.Status.RestoredPvcName = vr.Spec.TargetPvcName
	if err := r.Status().Update(ctx, vr); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
}

func (r *VolumeRestoreReconciler) handleCreatingPVC(ctx context.Context, vr *backupv1alpha1.VolumeRestore) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Determine target namespace (default to the VolumeRestore's namespace if not provided)
	targetNamespace := vr.Spec.TargetPvcNamespace
	if targetNamespace == "" {
		targetNamespace = vr.Namespace
	}

	// 1. Fetch the target PVC
	var pvc corev1.PersistentVolumeClaim
	pvcKey := types.NamespacedName{Name: vr.Status.RestoredPvcName, Namespace: targetNamespace}
	if err := r.Get(ctx, pvcKey, &pvc); err != nil {
		if errors.IsNotFound(err) {
			vr.Status.Phase = util.FAILED
			vr.Status.Message = "Restored PVC not found"
			_ = r.Status().Update(ctx, vr)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// 2. Examine the PVC status
	switch pvc.Status.Phase {
	case corev1.ClaimBound:
		now := metav1.Now()
		vr.Status.Phase = util.COMPLETED
		vr.Status.Message = "PVC successfully bound"
		vr.Status.CompletionTime = &now
		if err := r.Status().Update(ctx, vr); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil

	case corev1.ClaimPending:
		log.Info("PVC is still pending, requeuing")
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil

	default:
		vr.Status.Phase = util.FAILED
		vr.Status.Message = fmt.Sprintf("PVC entered unexpected phase: %s", pvc.Status.Phase)
		_ = r.Status().Update(ctx, vr)
		return ctrl.Result{}, nil
	}
}
