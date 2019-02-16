package record

import (
	"fmt"
	"log"

	dns "google.golang.org/api/dns/v1"
	"google.golang.org/api/googleapi"
)

func (r *ReconcileRecord) read() (*dns.ResourceRecordSet, error) {
	records, err := r.gce.Service.ResourceRecordSets.List(r.gce.ProjectID, r.managedZoneName()).Do()
	if err != nil {
		if googleapiError, ok := err.(*googleapi.Error); ok && googleapiError.Code != 404 {
			log.Printf("reconcile error: something strange went wrong with %s - %s", r.spec.Name, err.Error())
			return nil, err
		}
	}
	if records == nil {
		return nil, fmt.Errorf("no valid zone found for %s", r.managedZone)
	}
	for _, record := range records.Rrsets {
		if record.Name == r.spec.Name {
			return record, nil
		}
	}
	return nil, nil
}

func (r *ReconcileRecord) recordSet() []*dns.ResourceRecordSet {
	txtString := "heritage=k8s-gcp-operator,name=" + r.k8sObject.Name + ",namespace=" + r.k8sObject.Namespace
	txtRecord := &dns.ResourceRecordSet{
		Name:    r.spec.Name,
		Type:    "TXT",
		Rrdatas: []string{txtString},
		Ttl:     r.spec.Ttl,
	}
	return []*dns.ResourceRecordSet{r.spec, txtRecord}
}

func (r *ReconcileRecord) create() error {
	change := &dns.Change{Additions: r.recordSet()}
	_, err := r.gce.Service.Changes.Create(r.gce.ProjectID, r.managedZoneName(), change).Do()
	if err != nil {
		if googleapiError, ok := err.(*googleapi.Error); ok && googleapiError.Code == 409 {
			log.Printf("reconcile: Error, the name %s is unavailable because it was used recently", r.spec.Name)
			return fmt.Errorf("Error, the name %s is unavailable because it was used recently", r.spec.Name)
		} else if googleapi.IsNotModified(googleapiError) {
			return nil
		} else {
			log.Printf("Error, failed to create resource %s: %s", r.spec.Name, err)
			return fmt.Errorf("Error, failed to create resource %s: %s", r.spec.Name, err)
		}
	}
	return nil
}

func (r *ReconcileRecord) destroy() error {
	change := &dns.Change{Deletions: r.recordSet()}
	_, err := r.gce.Service.Changes.Create(r.gce.ProjectID, r.managedZoneName(), change).Do()
	if err != nil {
		if googleapiError, ok := err.(*googleapi.Error); ok && googleapiError.Code == 409 {
			log.Printf("reconcile: Error, the name %s is unavailable because it was used recently", r.spec.Name)
			return fmt.Errorf("Error, the name %s is unavailable because it was used recently", r.spec.Name)
		} else if googleapi.IsNotModified(googleapiError) {
			return nil
		} else {
			log.Printf("Error, failed to create resource %s: %s", r.spec.Name, err)
			return fmt.Errorf("Error, failed to create resource %s: %s", r.spec.Name, err)
		}
	}
	return nil
}

func (r *ReconcileRecord) managedZoneName() string {
	zones, _ := r.gce.Service.ManagedZones.List(r.gce.ProjectID).Do()
	for _, zone := range zones.ManagedZones {
		if zone.DnsName == r.managedZone {
			return zone.Name
		}
		if zone.DnsName == r.managedZone+"." {
			return zone.Name
		}
		if zone.Name == r.managedZone {
			return zone.Name
		}
	}
	return ""
}
