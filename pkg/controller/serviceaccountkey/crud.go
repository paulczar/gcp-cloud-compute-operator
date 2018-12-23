package serviceaccountkey

import (
	"fmt"
	"log"

	"github.com/paulczar/gcp-cloud-compute-operator/pkg/utils"
	"google.golang.org/api/googleapi"
	iam "google.golang.org/api/iam/v1"
)

func (r *ReconcileServiceAccountKey) read() (*iam.ServiceAccountKey, error) {
	name := r.k8sObject.Status.Name
	result, err := r.gce.Service.Projects.ServiceAccounts.Keys.Get(name).Do()
	if err != nil {
		if googleapiError, ok := err.(*googleapi.Error); ok && googleapiError.Code != 404 {
			log.Printf("reconcile error: something strange went wrong with %s - %s", name, err.Error())
			return nil, err
		}
	}
	return result, nil
}

func (r *ReconcileServiceAccountKey) create() (*iam.ServiceAccountKey, error) {
	serviceAccount := utils.GetAnnotation(r.annotations, utils.ServiceAccountAnnotation)
	fqn := utils.ServiceAccountFQN(r.gce.ProjectID, serviceAccount)
	req := &iam.CreateServiceAccountKeyRequest{
		KeyAlgorithm:   r.spec.KeyAlgorithm,
		PrivateKeyType: r.spec.PrivateKeyType,
	}
	//spew.Dump(sar)
	// projects/{PROJECT_ID}/serviceAccounts/{ACCOUNT}/keys/{key}
	key, err := r.gce.Service.Projects.ServiceAccounts.Keys.Create(fqn, req).Do()
	if err != nil {
		log.Printf("Error, failed to create key for service account %s: %s", serviceAccount, err)
		return nil, fmt.Errorf("Error, failed to create resource %s: %s", serviceAccount, err)
	}
	//log.Printf("created: %v", key)
	return key, nil
}

func (r *ReconcileServiceAccountKey) destroy() error {
	name := r.k8sObject.Status.Name
	_, err := r.gce.Service.Projects.ServiceAccounts.Keys.Delete(name).Do()
	if err != nil {
		if googleapiError, ok := err.(*googleapi.Error); ok && googleapiError.Code != 404 {
			if googleapiError.Code == 400 {
				return err
			}
			log.Printf("reconcile error: something strange went deleting resource %s - %s", name, err.Error())
			return err
		}
		if googleapiError, ok := err.(*googleapi.Error); ok && googleapiError.Code == 404 {
			log.Printf("reconcile: already deleted resource %s - %s", name, err.Error())
			return nil
		}
	}
	return nil
}
