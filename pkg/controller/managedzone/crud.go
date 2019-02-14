package managedzone

import (
	"fmt"
	"log"

	dns "google.golang.org/api/dns/v1"
	"google.golang.org/api/googleapi"
)

func (r *ReconcileManagedZone) read() (*dns.ManagedZone, error) {
	address, err := r.gce.Service.ManagedZones.Get(r.gce.ProjectID, r.spec.Name).Do()
	if err != nil {
		if googleapiError, ok := err.(*googleapi.Error); ok && googleapiError.Code != 404 {
			log.Printf("reconcile error: something strange went wrong with %s - %s", r.spec.Name, err.Error())
			return nil, err
		}
	}
	return address, nil
}

func (r *ReconcileManagedZone) create() error {
	_, err := r.gce.Service.ManagedZones.Create(r.gce.ProjectID, r.spec).Do()
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

func (r *ReconcileManagedZone) destroy() error {
	err := r.gce.Service.ManagedZones.Delete(r.gce.ProjectID, r.spec.Name).Do()
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
