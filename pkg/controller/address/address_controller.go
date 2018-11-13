package address

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/mitchellh/mapstructure"
	addressesv1alpha1 "github.com/paulczar/gcp-cloud-compute-operator/pkg/apis/addresses/v1alpha1"
	"github.com/paulczar/gcp-cloud-compute-operator/pkg/gce"
	"github.com/paulczar/gcp-cloud-compute-operator/pkg/utils"
	compute "google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	finalizer                 = "finalizer.compute.gce"
	reconcilePeriodAnnotation = "compute.gce/reconcile-period"
)

// Add creates a new Address Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	gce, err := gce.New("")
	if err != nil {
		panic(err)
	}
	return &ReconcileAddress{
		client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
		gce:    gce,
		reconcileResult: reconcile.Result{
			RequeueAfter: time.Duration(5 * time.Second),
		},
		address: &compute.Address{},
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("address-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Address
	err = c.Watch(&source.Kind{Type: &addressesv1alpha1.Address{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner Address
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &addressesv1alpha1.Address{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileAddress{}

// ReconcileAddress reconciles a Address object
type ReconcileAddress struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client          client.Client
	scheme          *runtime.Scheme
	gce             *gce.Client
	reconcileResult reconcile.Result
	annotations     map[string]string
	address         *compute.Address
	k8sAddressSpec  *compute.Address
}

// Reconcile reads that state of the cluster for a Address object and makes changes based on the state read
// and what is in the Address.Spec
func (r *ReconcileAddress) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	log.Printf("Reconciling Address %s/%s\n", request.Namespace, request.Name)

	// Fetch the Address r.k8sObject
	k8sObject := &addressesv1alpha1.Address{}
	err := r.client.Get(context.TODO(), request.NamespacedName, k8sObject)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return r.reconcileResult, nil
		}
		// Error reading the object - requeue the request.
		return r.reconcileResult, err
	}

	// Define a new gcp sqladmin instance object
	err = mapstructure.Decode(k8sObject.Spec, &r.k8sAddressSpec)
	if err != nil {
		panic(err)
	}

	// fetch annotations
	r.annotations = k8sObject.GetAnnotations()

	// update requeue duration based on annotation
	duration, err := time.ParseDuration(utils.GetAnnotation(r.annotations, reconcilePeriodAnnotation))
	if err == nil {
		r.reconcileResult.RequeueAfter = duration
	}

	// check if the resource is set to be deleted
	// stolen from https://github.com/operator-framework/operator-sdk/blob/fc9b6b1277b644d152534b22614351aa3d1405ba/pkg/ansible/controller/reconcile.go
	deleted := k8sObject.GetDeletionTimestamp() != nil
	pendingFinalizers := k8sObject.GetFinalizers()
	finalizerExists := len(pendingFinalizers) > 0
	if !finalizerExists && !deleted && !utils.Contains(pendingFinalizers, finalizer) {
		log.Printf("Adding finalizer %s to resource", finalizer)
		finalizers := append(pendingFinalizers, finalizer)
		k8sObject.SetFinalizers(finalizers)
		err := r.client.Update(context.TODO(), k8sObject)
		if err != nil {
			return r.reconcileResult, err
		}
	}

	// fetch the corresponding object from GCE
	gceObject, err := r.read()
	if err != nil {
		return r.reconcileResult, err
	}
	// if it doesn't existin in gcp and is set to be deleted,
	// then we can strip out the finalizer to let k8s actually delete it.
	if gceObject == nil && deleted && finalizerExists {
		log.Printf("reconcile: remove finalizer %s from %s/%s", finalizer, k8sObject.Namespace, k8sObject.Name)
		finalizers := []string{}
		for _, pendingFinalizer := range pendingFinalizers {
			if pendingFinalizer != finalizer {
				finalizers = append(finalizers, pendingFinalizer)
			}
		}
		k8sObject.SetFinalizers(finalizers)
		err := r.client.Update(context.TODO(), k8sObject)
		if err != nil {
			return r.reconcileResult, err
		}
		//todo fix this to stop requeuing
		log.Printf("reconcile: Successfully deleted %s/%s, do not requeue", k8sObject.Namespace, k8sObject.Name)
		return reconcile.Result{Requeue: false}, nil
		//r.reconcileResult.RequeueAfter, _ = time.ParseDuration("10m")
	}
	// if not deleted and gceObject doesn't exist we can create one.
	if !deleted && gceObject == nil {
		log.Printf("reconcile: creating database instance %s", r.k8sAddressSpec.Name)
		err := r.create()
		return r.reconcileResult, err
	}

	if gceObject != nil {
		//spew.Dump(gceObject)
		if deleted && finalizerExists {
			log.Printf("reconcile: time to delete %s/%s", r.k8sAddressSpec.Region, r.k8sAddressSpec.Name)
			err := r.destroy()
			if err != nil {
				r.reconcileResult.RequeueAfter, _ = time.ParseDuration("5s")
				return r.reconcileResult, err
			}
			r.reconcileResult.RequeueAfter, _ = time.ParseDuration("5s")
			return r.reconcileResult, err
		}
		log.Printf("reconcile: database instance %s/%s already exists", r.k8sAddressSpec.Region, r.k8sAddressSpec.Name)
		// update our k8s resource to include status from database
		k8sObject.Status.Status = gceObject.Status
		k8sObject.Status.IPAddress = gceObject.Address
		k8sObject.Status.SelfLink = gceObject.SelfLink
		k8sObject.Status.Region = gceObject.Region
		log.Printf("reconcile: update k8s status for %s/%s", k8sObject.Namespace, k8sObject.Name)
		err = r.client.Update(context.TODO(), k8sObject)
		if err != nil {
			return r.reconcileResult, err
		}
		if gceObject.Status == "RESERVED" {
			log.Printf("reconcile: successfully created %s/%s, change requeue to 10mins so we don't stampede gcp.", k8sObject.Namespace, k8sObject.Name)
		}
		r.reconcileResult.RequeueAfter, _ = time.ParseDuration("10m")
		return r.reconcileResult, nil

	}

	// resource exists - don't requeue
	//log.Printf("Skip reconcile: Pod %s/%s already exists", found.Namespace, found.Name)
	return r.reconcileResult, nil
}

func (r *ReconcileAddress) read() (*compute.Address, error) {
	region := r.k8sAddressSpec.Region
	name := r.k8sAddressSpec.Name
	address, err := r.gce.Service.Addresses.Get(r.gce.ProjectID, region, name).Do()
	if err != nil {
		if googleapiError, ok := err.(*googleapi.Error); ok && googleapiError.Code != 404 {
			return nil, err
			log.Printf("reconcile error: something strange went wrong with %s/%s - %s", region, name, err.Error())
		}
	}
	return address, nil
}

func (r *ReconcileAddress) create() error {
	_, err := r.gce.Service.Addresses.Insert(r.gce.ProjectID, r.k8sAddressSpec.Region, r.k8sAddressSpec).Do()
	if err != nil {
		if googleapiError, ok := err.(*googleapi.Error); ok && googleapiError.Code == 409 {
			log.Printf("reconcile: Error, the name %s is unavailable because it was used recently", r.k8sAddressSpec.Name)
			return fmt.Errorf("Error, the name %s is unavailable because it was used recently", r.k8sAddressSpec.Name)
		} else {
			log.Printf("Error, failed to create instance %s: %s", r.k8sAddressSpec.Name, err)
			return fmt.Errorf("Error, failed to create instance %s: %s", r.k8sAddressSpec.Name, err)
		}
	}
	return nil
}

func (r *ReconcileAddress) destroy() error {
	_, err := r.gce.Service.Addresses.Delete(r.gce.ProjectID, r.k8sAddressSpec.Region, r.k8sAddressSpec.Name).Do()
	if err != nil {
		if googleapiError, ok := err.(*googleapi.Error); ok && googleapiError.Code != 404 {
			log.Printf("reconcile error: something strange went deleting database instance %s - %s", r.k8sAddressSpec.Name, err.Error())
			return err
		}
		if googleapiError, ok := err.(*googleapi.Error); ok && googleapiError.Code == 404 {
			log.Printf("reconcile: already deletd database instance %s - %s", r.k8sAddressSpec.Name, err.Error())
			return nil
		}
	}
	return nil
}
