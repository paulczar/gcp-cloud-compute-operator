package instance

import (
	"fmt"
	"log"

	compute "google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
)

func (r *ReconcileInstance) read() (*compute.Instance, error) {
	result, err := r.gce.Service.Instances.Get(r.gce.ProjectID, r.spec.Zone, r.spec.Name).Do()
	if err != nil {
		if googleapiError, ok := err.(*googleapi.Error); ok && googleapiError.Code != 404 {
			log.Printf("reconcile error: something strange went wrong with %s - %s", r.spec.Name, err.Error())
			return nil, err
		}
	}
	return result, nil
}

func (r *ReconcileInstance) create() error {
	_, err := r.gce.Service.Instances.Insert(r.gce.ProjectID, r.spec.Zone, r.spec).Do()
	if err != nil {
		if googleapiError, ok := err.(*googleapi.Error); ok && googleapiError.Code == 409 {
			log.Printf("reconcile: Error, the name %s is unavailable because it was used recently", r.spec.Name)
			return fmt.Errorf("Error, the name %s is unavailable because it was used recently", r.spec.Name)
		}
		log.Printf("Error, failed to create resource %s: %s", r.spec.Name, err)
		return fmt.Errorf("Error, failed to create resource %s: %s", r.spec.Name, err)
	}
	return nil
}

func (r *ReconcileInstance) destroy() error {
	_, err := r.gce.Service.Instances.Delete(r.gce.ProjectID, r.spec.Zone, r.spec.Name).Do()
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
