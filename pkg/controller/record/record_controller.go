package record

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	dnsv1 "github.com/paulczar/gcp-cloud-compute-operator/pkg/apis/dns/v1"
	gce "github.com/paulczar/gcp-cloud-compute-operator/pkg/gce/dns"
	"github.com/paulczar/gcp-cloud-compute-operator/pkg/utils"
	gceDNS "google.golang.org/api/dns/v1"
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

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new Record Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	gceNew, err := gce.New("")
	if err != nil {
		panic(err)
	}
	return &ReconcileRecord{
		client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
		gce:    gceNew,
		reconcileResult: reconcile.Result{
			RequeueAfter: time.Duration(5 * time.Second),
		},
		k8sObject: &dnsv1.Record{},
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("record-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Record
	err = c.Watch(&source.Kind{Type: &dnsv1.Record{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner Record
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &dnsv1.Record{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileRecord{}

// ReconcileRecord reconciles a Record object
type ReconcileRecord struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client          client.Client
	scheme          *runtime.Scheme
	gce             *gce.Client
	reconcileResult reconcile.Result
	annotations     map[string]string
	spec            *gceDNS.ResourceRecordSet
	k8sObject       *dnsv1.Record
	managedZone     string
}

// Reconcile reads that state of the cluster for a Record object and makes changes based on the state read
// and what is in the Record.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileRecord) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	var finalizer = utils.Finalizer
	log.Printf("Reconciling record %s/%s\n", request.Namespace, request.Name)
	// Fetch the Address r.k8sObject
	err := r.client.Get(context.TODO(), request.NamespacedName, r.k8sObject)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Printf("Request object not found, could have been deleted after reconcile request.")
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		log.Printf("Error reading the object - requeue the request %s.", err.Error())
		return r.reconcileResult, err
	}
	var kind = r.k8sObject.TypeMeta.Kind

	// Define a new instance object
	r.spec = r.k8sObject.Spec
	if !strings.HasSuffix(r.spec.Name, ".") {
		r.spec.Name += "."
	}

	// fetch annotations
	r.annotations = r.k8sObject.GetAnnotations()

	// update requeue duration based on annotation
	duration, err := time.ParseDuration(utils.GetAnnotation(r.annotations, utils.ReconcilePeriodAnnotation))
	if err == nil {
		r.reconcileResult.RequeueAfter = duration
	}

	r.managedZone = utils.GetAnnotation(r.annotations, utils.ManagedZoneAnnotation)
	if r.managedZone == "" {
		log.Printf("Must provide Zone annotation %s.", utils.ManagedZoneAnnotation)
		return reconcile.Result{}, nil
	}

	// log into GCE using project
	if utils.GetAnnotation(r.annotations, utils.ProjectIDAnnotation) != "" {
		r.gce, err = gce.New(utils.ProjectIDAnnotation)
		if err != nil {
			panic(err)
		}
	}

	// fetch the corresponding object from GCE
	gceObject, err := r.read()
	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
		if strings.Contains(err.Error(), "no valid zone found") {
			r.k8sObject.Status.Status = "FAILED"
			err = r.client.Update(context.TODO(), r.k8sObject)
			if err != nil {
				return r.reconcileResult, err
			}
			return reconcile.Result{}, nil
		}
		return r.reconcileResult, err
	}

	// check if the resource is set to be deleted
	// stolen from https://github.com/operator-framework/operator-sdk/blob/fc9b6b1277b644d152534b22614351aa3d1405ba/pkg/ansible/controller/reconcile.go
	deleted := r.k8sObject.GetDeletionTimestamp() != nil
	pendingFinalizers := r.k8sObject.GetFinalizers()
	finalizerExists := len(pendingFinalizers) > 0
	if !finalizerExists && !deleted && !utils.Contains(pendingFinalizers, finalizer) {
		log.Printf("Adding finalizer %s to resource", finalizer)
		finalizers := append(pendingFinalizers, finalizer)
		r.k8sObject.SetFinalizers(finalizers)
		err := r.client.Update(context.TODO(), r.k8sObject)
		if err != nil {
			return r.reconcileResult, err
		}
	}

	// if it doesn't existin in gcp and is set to be deleted,
	// then we can strip out the finalizer to let k8s actually delete it.
	if gceObject == nil && deleted && finalizerExists {
		log.Printf("reconcile: remove finalizer %s from %s/%s", finalizer, r.k8sObject.Namespace, r.k8sObject.Name)
		finalizers := []string{}
		for _, pendingFinalizer := range pendingFinalizers {
			if pendingFinalizer != finalizer {
				finalizers = append(finalizers, pendingFinalizer)
			}
		}
		r.k8sObject.SetFinalizers(finalizers)
		err := r.client.Update(context.TODO(), r.k8sObject)
		if err != nil {
			return r.reconcileResult, err
		}
		//todo fix this to stop requeuing
		log.Printf("reconcile: Successfully deleted %s/%s, do not requeue", r.k8sObject.Namespace, r.k8sObject.Name)
		return reconcile.Result{Requeue: false}, nil
		//r.reconcileResult.RequeueAfter, _ = time.ParseDuration("10m")
	}
	// if not deleted and gceObject doesn't exist we can create one.
	if !deleted && gceObject == nil {
		log.Printf("reconcile: creating %s instance %s", kind, r.spec.Name)
		err := r.create()
		return r.reconcileResult, err
	}

	if gceObject != nil {
		//spew.Dump(gceObject)
		if deleted && finalizerExists {
			log.Printf("reconcile: time to delete %s", r.spec.Name)
			err := r.destroy()
			if err != nil {
				r.reconcileResult.RequeueAfter, _ = time.ParseDuration("5s")
				return r.reconcileResult, err
			}
			r.reconcileResult.RequeueAfter, _ = time.ParseDuration("5s")
			return r.reconcileResult, err
		}

		log.Printf("reconcile: resource %s already exists", r.spec.Name)
		if r.k8sObject.Status.Status == "READY" {
			log.Printf("reconcile: successfully created %s/%s, change requeue to 10mins so we don't stampede gcp.", r.k8sObject.Namespace, r.k8sObject.Name)
			r.reconcileResult.RequeueAfter, _ = time.ParseDuration("10m")
			return r.reconcileResult, nil
		}
		if r.k8sObject.Status.Status == "FAILED" {
			return reconcile.Result{}, nil
		}
		// update our k8s resource to include status from resource
		if gceObject.Name != "" {
			r.k8sObject.Status.Status = "READY"
		}
		log.Printf("reconcile: update k8s status for %s/%s", r.k8sObject.Namespace, r.k8sObject.Name)
		err = r.client.Update(context.TODO(), r.k8sObject)
		if err != nil {
			return r.reconcileResult, err
		}
		return r.reconcileResult, nil

	}
	return reconcile.Result{}, nil
}
