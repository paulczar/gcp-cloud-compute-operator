package utils

const (
	// Finalizer is the string for the k8s finalizer
	Finalizer = "finalizer.compute.gce"
	// ReconcilePeriodAnnotation is the annotation to accept alternative reconcile-period
	ReconcilePeriodAnnotation = "compute.gce/reconcile-period"
	// ProjectIDAnnotation is the annotation for accept an alternative Project ID
	ProjectIDAnnotation = "compute.gce/project-id"
)

// GetAnnotation returns a thing
func GetAnnotation(annotations map[string]string, a string) string {
	if res, ok := annotations[a]; ok {
		return res
	}
	return ""
}

// Contains verifies if a list of strings contains a given string
func Contains(l []string, s string) bool {
	for _, elem := range l {
		if elem == s {
			return true
		}
	}
	return false
}
