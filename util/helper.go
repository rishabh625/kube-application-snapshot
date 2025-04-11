package util

import (
	"strings"

	"k8s.io/client-go/discovery"
	ctrl "sigs.k8s.io/controller-runtime"
)

func DetectDistro() string {
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(ctrl.GetConfigOrDie())
	if err != nil {
		return UNKNOWN
	}
	version, err := discoveryClient.ServerVersion()
	if err != nil {
		return UNKNOWN
	}
	v := version.GitVersion
	switch {
	case strings.Contains(v, "eks"):
		return "EKS"
	case strings.Contains(v, "gke"):
		return "GKE"
	case strings.Contains(v, "k3s"):
		return "k3s"
	case strings.Contains(v, "rke2"):
		return "RKE2"
	case strings.Contains(v, "minikube"):
		return "minikube"
	case strings.Contains(v, "kind"):
		return "kind"
	case strings.Contains(v, "+e"): // OpenShift
		return "OpenShift"
	default:
		return UNKNOWN
	}
}

func SnapshotHelpMessage(distro string) string {
	// TODO - In progress
	/*switch distro {
		case "EKS":
			return `EKS detected. Install snapshot controller:
	  kubectl apply -f https://raw.githubusercontent.com/kubernetes-csi/external-snapshotter/v6.2.1/deploy/kubernetes/snapshot-controller/rbac-snapshot-controller.yaml
	  kubectl apply -f https://raw.githubusercontent.com/kubernetes-csi/external-snapshotter/v6.2.1/deploy/kubernetes/snapshot-controller/setup-snapshot-controller.yaml
	Create a VolumeSnapshotClass using AWS EBS CSI driver.`

		case "GKE":
			return `GKE detected. Install snapshot controller if not already present (non-Autopilot).
	Use pd.csi.storage.gke.io as your VolumeSnapshotClass driver.`

		case "k3s":
			return `k3s detected. Manually install snapshot-controller and a CSI driver with snapshot support.
	See: https://github.com/kubernetes-csi/external-snapshotter`

		case "RKE2":
			return `RKE2 detected. Snapshot support is not always pre-installed. Install the snapshot-controller and create a compatible VolumeSnapshotClass.`

		case "minikube", "kind":
			return fmt.Sprintf(`%s detected. Install snapshot-controller and use compatible CSI plugin (e.g., hostpath).
	Try:
	  kubectl apply -f https://raw.githubusercontent.com/kubernetes-csi/external-snapshotter/v6.2.1/deploy/kubernetes/snapshot-controller/setup-snapshot-controller.yaml`, distro)

		case "OpenShift":
			return `OpenShift detected. Install snapshot support via OperatorHub or use built-in snapshot-enabled CSI drivers.`

		default:
			return `Unknown Kubernetes distro. Install snapshot controller:
	  kubectl apply -f https://raw.githubusercontent.com/kubernetes-csi/external-snapshotter/v6.2.1/deploy/kubernetes/snapshot-controller/setup-snapshot-controller.yaml
	Create a VolumeSnapshotClass for your CSI driver.`
		}*/
	return ""
}
