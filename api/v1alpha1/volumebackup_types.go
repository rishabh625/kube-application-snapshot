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

// VolumeBackupSpec defines the desired state of VolumeBackup.
type VolumeBackupSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// The name of the VolumeBackup object, or directly the VolumeSnapshot object,
	// to restore from. Using VolumeSnapshot name is more direct.
	//+kubebuilder:validation:Required

	SnapshotName string `json:"snapshotName"` // Assumes snapshot exists in the same namespace as the Restore request
	// The name of the PersistentVolumeClaim to backup.
	//+kubebuilder:validation:Required

	PvcName string `json:"pvcName"`

	// The namespace where the PVC to backup resides.
	//+kubebuilder:validation:Required
	PvcNamespace string `json:"pvcNamespace"`

	// The name of the VolumeSnapshotClass to use for the backup.
	//+kubebuilder:validation:Required
	VolumeSnapshotClassName string `json:"volumeSnapshotClassName"`
}

// VolumeBackupStatus defines the observed state of VolumeBackup.
type VolumeBackupStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// Represents the current phase of the backup operation.
	// +optional
	Phase string `json:"phase,omitempty"` // e.g., Pending, CreatingSnapshot, Completed, Failed

	// A human-readable message indicating details about the last transition.
	// +optional
	Message string `json:"message,omitempty"`

	// The name of the VolumeSnapshot object created by this backup.
	// +optional
	SnapshotHandle string `json:"snapshotHandle,omitempty"`

	// The time the backup operation was completed.
	// +optional
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	ReadyToUse bool `json:"readyToUse,omitempty"` // Indicates if the backup is ready to be used
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// VolumeBackup is the Schema for the volumebackups API.
type VolumeBackup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VolumeBackupSpec   `json:"spec,omitempty"`
	Status VolumeBackupStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// VolumeBackupList contains a list of VolumeBackup.
type VolumeBackupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VolumeBackup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VolumeBackup{}, &VolumeBackupList{})
}
