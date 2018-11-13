package network

import (
	"fmt"
	"log"

	compute "google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
)

func (r *ReconcileNetwork) read() (*compute.Network, error) {
	name := r.spec.Name
	address, err := r.gce.Service.Networks.Get(r.gce.ProjectID, name).Do()
	if err != nil {
		if googleapiError, ok := err.(*googleapi.Error); ok && googleapiError.Code != 404 {
			log.Printf("reconcile error: something strange went wrong with %s - %s", name, err.Error())
			return nil, err
		}
	}
	return address, nil
}

func (r *ReconcileNetwork) create() error {
	// otherwise will create a legacy network and baby jesus will cry
	r.spec.ForceSendFields = []string{"AutoCreateSubnetworks"}
	if !r.spec.AutoCreateSubnetworks {
		r.spec.AutoCreateSubnetworks = false
	}

	_, err := r.gce.Service.Networks.Insert(r.gce.ProjectID, r.spec).Do()
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

func (r *ReconcileNetwork) destroy() error {
	_, err := r.gce.Service.Networks.Delete(r.gce.ProjectID, r.spec.Name).Do()
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
