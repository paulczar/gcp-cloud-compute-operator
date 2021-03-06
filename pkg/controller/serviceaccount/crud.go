package serviceaccount

import (
	"fmt"
	"log"

	"github.com/paulczar/gcp-cloud-compute-operator/pkg/utils"
	"google.golang.org/api/googleapi"
	iam "google.golang.org/api/iam/v1"
)

func (r *ReconcileServiceAccount) read() (*iam.ServiceAccount, error) {
	name := utils.ServiceAccountName(r.gce.ProjectID, r.spec.Name)
	result, err := r.gce.Service.Projects.ServiceAccounts.Get(name).Do()
	if err != nil {
		if googleapiError, ok := err.(*googleapi.Error); ok && googleapiError.Code != 404 {
			log.Printf("reconcile error: something strange went wrong with %s - %s", r.spec.Name, err.Error())
			return nil, err
		}
	}
	return result, nil
}

func (r *ReconcileServiceAccount) create() error {
	accountID := r.spec.Name
	project := "projects/" + r.gce.ProjectID
	r.spec.Name = ""
	sar := iam.CreateServiceAccountRequest{
		AccountId:      accountID,
		ServiceAccount: r.spec,
	}
	//spew.Dump(sar)
	_, err := r.gce.Service.Projects.ServiceAccounts.Create(project, &sar).Do()
	if err != nil {
		if googleapiError, ok := err.(*googleapi.Error); ok && googleapiError.Code == 409 {
			log.Printf("reconcile: Error, the name %s is unavailable because it was used recently", r.spec.Name)
			return fmt.Errorf("Error, the name %s is unavailable because it was used recently", r.spec.Name)
		} else {
			log.Printf("Error, failed to create resource %s: %s", r.spec.Name, err)
			return fmt.Errorf("Error, failed to create resource %s: %s", r.spec.Name, err)
		}
	}
	return nil
}

func (r *ReconcileServiceAccount) destroy() error {
	name := utils.ServiceAccountName(r.gce.ProjectID, r.spec.Name)
	_, err := r.gce.Service.Projects.ServiceAccounts.Delete(name).Do()
	if err != nil {
		if googleapiError, ok := err.(*googleapi.Error); ok && googleapiError.Code != 404 {
			if googleapiError.Code == 400 {
				return err
			}
			log.Printf("reconcile error: something strange went deleting resource %s - %s", r.spec.Name, err.Error())
			return err
		}
		if googleapiError, ok := err.(*googleapi.Error); ok && googleapiError.Code == 404 {
			log.Printf("reconcile: already deleted resource %s - %s", r.spec.Name, err.Error())
			return nil
		}
	}
	return nil
}
