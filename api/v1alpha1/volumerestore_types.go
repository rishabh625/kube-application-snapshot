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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// VolumeRestoreSpec defines the desired state of VolumeRestore.
type VolumeRestoreSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// The name of the VolumeBackup object, or directly the VolumeSnapshot object,
	// to restore from. Using VolumeSnapshot name is more direct.
	//+kubebuilder:validation:Required
	SourceSnapshotName string `json:"sourceSnapshotName"` // Assumes snapshot exists in the same namespace as the Restore request

	// The namespace where the source VolumeSnapshot resides.
	//+kubebuilder:validation:Required
	SourceSnapshotNamespace string `json:"sourceSnapshotNamespace"`

	// The name of the target PersistentVolumeClaim to be created from the snapshot.
	//+kubebuilder:validation:Required
	TargetPvcName string `json:"targetPvcName"`

	// The namespace where the target PVC should be created. Defaults to the VolumeRestore's namespace.
	// +optional
	TargetPvcNamespace string `json:"targetPvcNamespace,omitempty"`

	// Optional: Specify the StorageClassName for the restored PVC.
	// If not specified, it might default based on the VolumeSnapshotClass or cluster defaults.
	// +optional
	TargetStorageClassName *string `json:"targetStorageClassName,omitempty"`
}

// VolumeRestoreStatus defines the observed state of VolumeRestore.
type VolumeRestoreStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Represents the current phase of the restore operation.
	// +optional
	Status string `json:"status,omitempty"` // e.g., Pending, CreatingPVC, Completed, Failed

	// A human-readable message indicating details about the last transition.
	// +optional
	Message string `json:"message,omitempty"`

	// The name of the PersistentVolumeClaim created by this restore operation.
	// +optional
	RestoredPvcName string `json:"restoredPvcName,omitempty"`

	// The time the restore operation was completed.
	// +optional
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// VolumeRestore is the Schema for the volumerestores API.
type VolumeRestore struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VolumeRestoreSpec   `json:"spec,omitempty"`
	Status VolumeRestoreStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// VolumeRestoreList contains a list of VolumeRestore.
type VolumeRestoreList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VolumeRestore `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VolumeRestore{}, &VolumeRestoreList{})
}
