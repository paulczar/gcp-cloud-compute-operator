package address

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
	err = c.Watch(&source.Kind{Type: &computev1.Address{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner Address
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &computev1.Address{},
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
	spec            *gceCompute.Address
}

// Reconcile reads that state of the cluster for a Address object and makes changes based on the state read
// and what is in the Address.Spec
func (r *ReconcileAddress) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	log.Printf("Reconciling Address %s/%s\n", request.Namespace, request.Name)

	// Fetch the Address r.k8sObject
	k8sObject := &computev1.Address{}
	err := r.client.Get(context.TODO(), request.NamespacedName, k8sObject)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Printf("Request object not found, could have been deleted after reconcile request.")
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		log.Printf("Error reading the object - requeue the request.")
		return r.reconcileResult, err
	}

	// Define a new instance object
	err = mapstructure.Decode(k8sObject.Spec, &r.spec)
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

	// log into GCE using project
	if utils.GetAnnotation(r.annotations, utils.ProjectIDAnnotation) != "" {
		r.gce, err = gce.New(utils.ProjectIDAnnotation)
		if err != nil {
			panic(err)
		}
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
		log.Printf("reconcile: creating database instance %s", r.spec.Name)
		err := r.create()
		return r.reconcileResult, err
	}

	if gceObject != nil {
		//spew.Dump(gceObject)
		if deleted && finalizerExists {
			log.Printf("reconcile: time to delete %s/%s", r.spec.Region, r.spec.Name)
			err := r.destroy()
			if err != nil {
				r.reconcileResult.RequeueAfter, _ = time.ParseDuration("5s")
				return r.reconcileResult, err
			}
			r.reconcileResult.RequeueAfter, _ = time.ParseDuration("5s")
			return r.reconcileResult, err
		}
		log.Printf("reconcile: database instance %s/%s already exists", r.spec.Region, r.spec.Name)
		if k8sObject.Status.Status == "RESERVED" && k8sObject.Status.IPAddress != "" {
			log.Printf("reconcile: successfully created %s/%s, change requeue to 10mins so we don't stampede gcp.", k8sObject.Namespace, k8sObject.Name)
			r.reconcileResult.RequeueAfter, _ = time.ParseDuration("10m")
			return r.reconcileResult, nil
		}
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
		return r.reconcileResult, nil

	}
	return reconcile.Result{}, nil
}
