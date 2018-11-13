package image

import (
	"fmt"
	"log"

	compute "google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
)

func (r *ReconcileImage) read() (*compute.Image, error) {
	name := r.spec.Name
	address, err := r.gce.Service.Images.Get(r.gce.ProjectID, name).Do()
	if err != nil {
		if googleapiError, ok := err.(*googleapi.Error); ok && googleapiError.Code != 404 {
			return nil, err
			log.Printf("reconcile error: something strange went wrong with %s - %s", name, err.Error())
		}
	}
	return address, nil
}

func (r *ReconcileImage) create() error {
	//r.spec.Labels
	if r.spec.Labels == nil {
		r.spec.Labels = map[string]string{}
	}
	r.spec.Labels["k8s_operator"] = "true"
	r.spec.Labels["k8s_namespace"] = r.k8sObject.Namespace
	r.spec.Labels["k8s_name"] = r.k8sObject.Name
	_, err := r.gce.Service.Images.Insert(r.gce.ProjectID, r.spec).Do()
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

func (r *ReconcileImage) destroy() error {
	_, err := r.gce.Service.Images.Delete(r.gce.ProjectID, r.spec.Name).Do()
	if err != nil {
		if googleapiError, ok := err.(*googleapi.Error); ok && googleapiError.Code != 404 {
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
