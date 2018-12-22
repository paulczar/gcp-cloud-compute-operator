package address

import (
	"fmt"
	"log"

	compute "google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
)

func (r *ReconcileAddress) read() (*compute.Address, error) {
	region := r.spec.Region
	name := r.spec.Name
	address, err := r.gce.Service.Addresses.Get(r.gce.ProjectID, region, name).Do()
	if err != nil {
		if googleapiError, ok := err.(*googleapi.Error); ok && googleapiError.Code != 404 {
			log.Printf("reconcile error: something strange went wrong with %s/%s - %s", region, name, err.Error())
			return nil, err
		}
	}
	return address, nil
}

func (r *ReconcileAddress) create() error {
	_, err := r.gce.Service.Addresses.Insert(r.gce.ProjectID, r.spec.Region, r.spec).Do()
	if err != nil {
		if googleapiError, ok := err.(*googleapi.Error); ok && googleapiError.Code == 409 {
			return fmt.Errorf("Error, the name %s is unavailable because it was used recently", r.spec.Name)
		}
		log.Printf("Error, failed to create instance %s: %s", r.spec.Name, err)
		return fmt.Errorf("Error, failed to create instance %s: %s", r.spec.Name, err)
	}
	return nil
}

func (r *ReconcileAddress) destroy() error {
	_, err := r.gce.Service.Addresses.Delete(r.gce.ProjectID, r.spec.Region, r.spec.Name).Do()
	if err != nil {
		if googleapiError, ok := err.(*googleapi.Error); ok && googleapiError.Code != 404 {
			log.Printf("reconcile error: something strange went deleting address instance %s - %s", r.spec.Name, err.Error())
			return err
		}
		if googleapiError, ok := err.(*googleapi.Error); ok && googleapiError.Code == 404 {
			log.Printf("reconcile: already deletd address instance %s - %s", r.spec.Name, err.Error())
			return nil
		}
	}
	return nil
}
