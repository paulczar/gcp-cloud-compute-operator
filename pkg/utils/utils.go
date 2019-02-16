package utils

import (
	"fmt"
	"strings"
)

const (
	// Finalizer is the string for the k8s finalizer
	Finalizer = "finalizer.compute.gce"
	// ReconcilePeriodAnnotation is the annotation to accept alternative reconcile-period
	ReconcilePeriodAnnotation = "compute.gce/reconcile-period"
	// ProjectIDAnnotation is the annotation for accept an alternative Project ID
	ProjectIDAnnotation = "compute.gce/project-id"
	// ServiceAccountAnnotation is the annotation to specify a serviceAccountName for IAM tasks
	ServiceAccountAnnotation = "iam.gce/serviceAccountName"
	// ManagedZoneAnnotation is the annotation to specify a managed zone for DNS tasks
	ManagedZoneAnnotation = "dns.gce/zone"
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

// ServiceAccountName takes the projectid and a name and returns
// a valid Service Account URL
func ServiceAccountName(projectID, name string) string {
	if strings.Contains(name, "projectID/") {
		return name
	}
	email := ServiceAccountEmail(projectID, name)
	return strings.Join([]string{"projects", projectID, "serviceAccounts", email}, "/")

}

// ServiceAccountEmail takes the projectid and a name and returns
// a valid service account email  address
func ServiceAccountEmail(projectID, name string) string {
	if strings.Contains(name, "@") {
		return name
	}
	return fmt.Sprintf("%s@%s.iam.gserviceaccount.com", name, projectID)
}

// ServiceAccountFQN creates the fully qualified name of the service account
func ServiceAccountFQN(projectID, name string) string {
	// If the service account id is already the fully qualified name
	if strings.HasPrefix(name, "projects/") {
		return name
	}
	if strings.Contains(name, "@") {
		return "projects/-/serviceAccounts/" + name
	}
	return fmt.Sprintf("projects/-/serviceAccounts/%s@%s.iam.gserviceaccount.com", name, projectID)
}
