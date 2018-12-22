package serviceaccount

import (
	"context"
	"log"
	"time"

	iamv1 "github.com/paulczar/gcp-cloud-compute-operator/pkg/apis/iam/v1"
	gce "github.com/paulczar/gcp-cloud-compute-operator/pkg/gce/iam"
	"github.com/paulczar/gcp-cloud-compute-operator/pkg/utils"
	gceIam "google.golang.org/api/iam/v1"
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

// Add creates a new ServiceAccount Controller and adds it to the Manager. The Manager will set fields on the Controller
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
	return &ReconcileServiceAccount{
		client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
		gce:    gceNew,
		reconcileResult: reconcile.Result{
			RequeueAfter: time.Duration(5 * time.Second),
		},
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("serviceaccount-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource ServiceAccount
	err = c.Watch(&source.Kind{Type: &iamv1.ServiceAccount{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner ServiceAccount
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &iamv1.ServiceAccount{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileServiceAccount{}

// ReconcileServiceAccount reconciles a ServiceAccount object
type ReconcileServiceAccount struct {
	client          client.Client
	scheme          *runtime.Scheme
	gce             *gce.Client
	reconcileResult reconcile.Result
	annotations     map[string]string
	spec            *gceIam.ServiceAccount
}

// Reconcile reads that state of the cluster for a ServiceAccount object and makes changes based on the state read
// and what is in the ServiceAccount.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileServiceAccount) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	kind := "IAM Service Account"
	log.Printf("Reconciling %s: %s/%s\n", kind, request.Namespace, request.Name)

	var finalizer = utils.Finalizer
	// Fetch the Address k8sObject
	k8sObject := &iamv1.ServiceAccount{}
	err := r.client.Get(context.TODO(), request.NamespacedName, k8sObject)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Printf("%s: Request object not found, could have been deleted after reconcile request.", kind)
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		log.Printf("%s: Error reading the object - requeue the request %s.", kind, err.Error())
		return r.reconcileResult, err
	}

	// Define a new instance object
	r.spec = k8sObject.Spec

	// fetch annotations
	r.annotations = k8sObject.GetAnnotations()

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
	deleted := k8sObject.GetDeletionTimestamp() != nil
	pendingFinalizers := k8sObject.GetFinalizers()
	finalizerExists := len(pendingFinalizers) > 0
	if !finalizerExists && !deleted && !utils.Contains(pendingFinalizers, finalizer) {
		log.Printf("Adding finalizer to %s %s/%s", k8sObject.Kind, k8sObject.Namespace, k8sObject.Name)
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
		log.Printf("reconcile: remove finalizer from %s %s/%s", k8sObject.Kind, k8sObject.Namespace, k8sObject.Name)
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
		log.Printf("reconcile: creating %s: %s/%s", kind, k8sObject.Namespace, k8sObject.Name)
		err := r.create()
		return r.reconcileResult, err
	}

	if gceObject != nil {
		//spew.Dump(gceObject)
		if deleted && finalizerExists {
			log.Printf("reconcile: deleting %s: %s/%s", kind, k8sObject.Namespace, k8sObject.Name)
			err := r.destroy()
			if err != nil {
				r.reconcileResult.RequeueAfter, _ = time.ParseDuration("5s")
				return r.reconcileResult, err
			}
			r.reconcileResult.RequeueAfter, _ = time.ParseDuration("5s")
			return r.reconcileResult, err
		}
		log.Printf("reconcile: %s: %s/%s already exists", kind, k8sObject.Namespace, k8sObject.Name)
		if k8sObject.Status.Status == "READY" {
			log.Printf("reconcile: successfully created %s: %s/%s, change requeue to 10mins so we don't stampede gcp.", kind, k8sObject.Namespace, k8sObject.Name)
			r.reconcileResult.RequeueAfter, _ = time.ParseDuration("10m")
			return r.reconcileResult, nil
		}
		if k8sObject.Status.Status == "FAILED" {
			return reconcile.Result{}, nil
		}
		// update our k8s resource to include status from IAM Service Account
		if gceObject.UniqueId != "" {
			k8sObject.Status.Status = "READY"
		}
		k8sObject.Status.ProjectId = gceObject.ProjectId
		k8sObject.Status.UniqueId = gceObject.UniqueId
		k8sObject.Status.Email = gceObject.Email
		k8sObject.Status.Name = gceObject.Name
		log.Printf("reconcile: update k8s status %s: for %s/%s", kind, k8sObject.Namespace, k8sObject.Name)
		err = r.client.Update(context.TODO(), k8sObject)
		if err != nil {
			return r.reconcileResult, err
		}
		return r.reconcileResult, nil

	}
	return reconcile.Result{}, nil
}
