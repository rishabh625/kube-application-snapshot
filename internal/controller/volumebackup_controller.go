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
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	customerror "errors"

	snapshotv1 "github.com/kubernetes-csi/external-snapshotter/client/v6/apis/volumesnapshot/v1"
	backupv1alpha1 "github.com/rishabh625/kube-application-snapshot/api/v1alpha1"
	"github.com/rishabh625/kube-application-snapshot/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VolumeBackupReconciler reconciles a VolumeBackup object
type VolumeBackupReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=backup.infracloud-citadel-vbakup.com,resources=volumebackups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=backup.infracloud-citadel-vbakup.com,resources=volumebackups/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=backup.infracloud-citadel-vbakup.com,resources=volumebackups/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the VolumeBackup object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.4/pkg/reconcile
// +kubebuilder:rbac:groups=backup,resources=volumebackups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=snapshot.storage.k8s.io,resources=volumesnapshots,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=snapshot.storage.k8s.io,resources=volumesnapshotcontents,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=events,verbs=get;list;watch;create;update;patch;delete
func (r *VolumeBackupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// TODO(user): your logic here
	var vb backupv1alpha1.VolumeBackup
	if err := r.Get(ctx, req.NamespacedName, &vb); err != nil {
		if errors.IsNotFound(err) {
			log.Info("VolumeBackup not found, might be deleted")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	switch vb.Status.Phase {
	case util.COMPLETED, util.FAILED:
		// Optionally reprocess if spec has changed
		log.Info("VolumeBackup is already completed or failed", "Phase", vb.Status.Phase)
		return ctrl.Result{}, nil

	case "", "Pending":
		return r.handlePending(ctx, &vb)

	case "CreatingSnapshot":
		return r.handleCreatingSnapshot(ctx, &vb)

	default:
		log.Info("Unknown phase, requeuing", "Phase", vb.Status.Phase)
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *VolumeBackupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&backupv1alpha1.VolumeBackup{}).
		Named("volumebackup").
		Complete(r)
}

func (r *VolumeBackupReconciler) handlePending(ctx context.Context, vb *backupv1alpha1.VolumeBackup) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	if !r.resourceExists("volumesnapshotclasses.snapshot.storage.k8s.io") {
		distro := util.DetectDistro()
		log.Info("VolumeSnapshotClass CRD not found. You likely need to install snapshot support.")
		log.Info(util.SnapshotHelpMessage(distro))

		vb.Status.Phase = util.FAILED
		vb.Status.Message = "Missing VolumeSnapshotClass CRD. Install snapshot-controller and class." + util.SnapshotHelpMessage(distro)
		_, err := r.updateStatusToFailed(ctx, vb, "Missing VolumeSnapshotClass CRD. Install snapshot-controller and class.")
		if err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, customerror.New("VolumeSnapshotClass CRD not found")

	}
	// 1. Check PVC existence
	var pvc corev1.PersistentVolumeClaim
	if err := r.Get(ctx, types.NamespacedName{Name: vb.Spec.PvcName, Namespace: vb.Spec.PvcNamespace}, &pvc); err != nil {
		if errors.IsNotFound(err) {
			return r.updateStatusToFailed(ctx, vb, "PVC not found")
		}
		return ctrl.Result{}, err
	}

	// 2. Check VolumeSnapshotClass existence
	var vsc snapshotv1.VolumeSnapshotClass
	if err := r.Get(ctx, types.NamespacedName{Name: vb.Spec.VolumeSnapshotClassName}, &vsc); err != nil {
		if errors.IsNotFound(err) {
			return r.updateStatusToFailed(ctx, vb, "VolumeSnapshotClass not found")
		}
		return ctrl.Result{}, err
	}

	// 3. Create VolumeSnapshot
	snapshotName := fmt.Sprintf("%s-snap-%d", vb.Name, time.Now().Unix())
	vs := &snapshotv1.VolumeSnapshot{
		ObjectMeta: metav1.ObjectMeta{
			Name:      snapshotName,
			Namespace: vb.Spec.PvcNamespace,
		},
		Spec: snapshotv1.VolumeSnapshotSpec{
			VolumeSnapshotClassName: &vb.Spec.VolumeSnapshotClassName,
			Source: snapshotv1.VolumeSnapshotSource{
				PersistentVolumeClaimName: &vb.Spec.PvcName,
			},
		},
	}
	if err := controllerutil.SetControllerReference(vb, vs, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}

	err := r.Create(ctx, vs)
	if err != nil && !errors.IsAlreadyExists(err) {
		return r.updateStatusToFailed(ctx, vb, fmt.Sprintf("Failed to create VolumeSnapshot: %v", err))
	}

	// 4. Update VolumeBackup status
	vb.Status.Phase = "CreatingSnapshot"
	vb.Status.SnapshotHandle = snapshotName
	vb.Status.Message = "VolumeSnapshot creation initiated"
	if err := r.Status().Update(ctx, vb); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("VolumeSnapshot created", "Snapshot", snapshotName)
	return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
}

func (r *VolumeBackupReconciler) updateStatusToFailed(ctx context.Context, vb *backupv1alpha1.VolumeBackup, message string) (ctrl.Result, error) {
	vb.Status.Phase = util.FAILED
	vb.Status.Message = message
	err := r.Status().Update(ctx, vb)
	return ctrl.Result{}, err
}

func (r *VolumeBackupReconciler) handleCreatingSnapshot(ctx context.Context, vb *backupv1alpha1.VolumeBackup) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	var vs snapshotv1.VolumeSnapshot
	if err := r.Get(ctx, types.NamespacedName{Name: vb.Status.SnapshotHandle, Namespace: vb.Spec.PvcNamespace}, &vs); err != nil {
		if errors.IsNotFound(err) {
			return r.updateStatusToFailed(ctx, vb, "VolumeSnapshot not found")
		}
		return ctrl.Result{}, err
	}

	if vs.Status == nil || vs.Status.ReadyToUse == nil {
		log.Info("VolumeSnapshot status not ready yet, requeuing...")
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	if *vs.Status.ReadyToUse {
		now := metav1.Now()
		vb.Status.Phase = util.COMPLETED
		vb.Status.ReadyToUse = true
		vb.Status.CompletionTime = &now
		vb.Status.Message = "Snapshot is ready to use"
		return ctrl.Result{}, r.Status().Update(ctx, vb)
	}

	if vs.Status.Error != nil {
		return r.updateStatusToFailed(ctx, vb, *vs.Status.Error.Message)
	}

	log.Info("Snapshot not ready yet, requeueing")
	return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
}

func (r *VolumeBackupReconciler) resourceExists(gvr string) bool {
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(ctrl.GetConfigOrDie())
	if err != nil {
		return false
	}

	apiResourceLists, err := discoveryClient.ServerPreferredResources()
	if err != nil {
		return false
	}

	for _, list := range apiResourceLists {
		for _, res := range list.APIResources {
			fullName := list.GroupVersion + "." + res.Name
			if strings.EqualFold(fullName, gvr) {
				return true
			}
		}
	}
	return false
}
