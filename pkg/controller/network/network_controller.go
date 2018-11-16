package network

import (
	"context"
	"log"
	"time"

	"github.com/mitchellh/mapstructure"
	computev1 "github.com/paulczar/gcp-cloud-compute-operator/pkg/apis/compute/v1"
	"github.com/paulczar/gcp-cloud-compute-operator/pkg/gce"
	"github.com/paulczar/gcp-cloud-compute-operator/pkg/utils"
	gceCompute "google.golang.org/api/compute/v1"
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

// Add creates a new Network Controller and adds it to the Manager. The Manager will set fields on the Controller
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
	return &ReconcileNetwork{
		client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
		gce:    gceNew,
		reconcileResult: reconcile.Result{
			RequeueAfter: time.Duration(5 * time.Second),
		},
		k8sObject: &computev1.Network{},
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("network-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Network
	err = c.Watch(&source.Kind{Type: &computev1.Network{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner Network
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &computev1.Network{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileNetwork{}

// ReconcileNetwork reconciles a Network object
type ReconcileNetwork struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client          client.Client
	scheme          *runtime.Scheme
	gce             *gce.Client
	reconcileResult reconcile.Result
	annotations     map[string]string
	spec            *gceCompute.Network
	k8sObject       *computev1.Network
}

// Reconcile reads that state of the cluster for a Network object and makes changes based on the state read
// and what is in the Network.Spec
func (r *ReconcileNetwork) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	log.Printf("Reconciling Network %s/%s\n", request.Namespace, request.Name)
	var finalizer = utils.Finalizer
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
	err = mapstructure.Decode(r.k8sObject.Spec, &r.spec)
	if err != nil {
		panic(err)
	}

	// fetch annotations
	r.annotations = r.k8sObject.GetAnnotations()

	// update requeue duration based on annotation
	duration, err := time.ParseDuration(utils.GetAnnotation(r.annotations, utils.ReconcilePeriodAnnotation))
	if err == nil {
		r.reconcileResult.RequeueAfter = duration
	}

	// log into GCE using project
	if utils.GetAnnotation(r.annotations, utils.ProjectIDAnnotation) != "" {
		r.gce, err = gce.New(utils.ProjectIDAnnotation)
		if err != nil {
			panic(err)
		}
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

	// fetch the corresponding object from GCE
	gceObject, err := r.read()
	if err != nil {
		return r.reconcileResult, err
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
		if gceObject.SelfLink != "" {
			r.k8sObject.Status.Status = "READY"
		}
		r.k8sObject.Status.SelfLink = gceObject.SelfLink
		r.k8sObject.Status.CreationTimestamp = gceObject.CreationTimestamp
		r.k8sObject.Status.Id = gceObject.Id
		log.Printf("reconcile: update k8s status for %s/%s", r.k8sObject.Namespace, r.k8sObject.Name)
		err = r.client.Update(context.TODO(), r.k8sObject)
		if err != nil {
			return r.reconcileResult, err
		}
		return r.reconcileResult, nil

	}
	return reconcile.Result{}, nil
}
