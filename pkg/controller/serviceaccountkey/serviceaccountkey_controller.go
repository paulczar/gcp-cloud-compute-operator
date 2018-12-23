package serviceaccountkey

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new ServiceAccountKey Controller and adds it to the Manager. The Manager will set fields on the Controller
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
	return &ReconcileServiceAccountKey{
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
	c, err := controller.New("serviceaccountkey-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource ServiceAccountKey
	err = c.Watch(&source.Kind{Type: &iamv1.ServiceAccountKey{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner ServiceAccountKey
	err = c.Watch(&source.Kind{Type: &corev1.Secret{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &iamv1.ServiceAccountKey{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileServiceAccountKey{}

// ReconcileServiceAccountKey reconciles a ServiceAccountKey object
type ReconcileServiceAccountKey struct {
	client          client.Client
	scheme          *runtime.Scheme
	gce             *gce.Client
	reconcileResult reconcile.Result
	annotations     map[string]string
	spec            *gceIam.ServiceAccountKey
	k8sObject       *iamv1.ServiceAccountKey
}

// Reconcile reads that state of the cluster for a ServiceAccountKey object and makes changes based on the state read
// and what is in the ServiceAccountKey.Spec
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileServiceAccountKey) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	kind := "IAM Service Account Key"
	log.Printf("Reconciling %s: %s/%s\n", kind, request.Namespace, request.Name)

	var finalizer = utils.Finalizer
	// Fetch the Address k8sObject
	r.k8sObject = &iamv1.ServiceAccountKey{}
	err := r.client.Get(context.TODO(), request.NamespacedName, r.k8sObject)
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
	// its valid for the k8sObject spec to be empty for serviceaccountkey
	// if so we should make it not nil.
	if r.k8sObject.Spec == nil {
		r.k8sObject.Spec = &gceIam.ServiceAccountKey{}
	}
	r.spec = r.k8sObject.Spec

	// fetch annotations
	r.annotations = r.k8sObject.GetAnnotations()

	// check for service account annotation and fail if not there
	if utils.GetAnnotation(r.annotations, utils.ServiceAccountAnnotation) == "" {
		log.Printf("cannot create %s %s/%s, must provide annotation: %s",
			r.k8sObject.Kind, r.k8sObject.Namespace, r.k8sObject.Name, utils.ServiceAccountAnnotation)
		r.k8sObject.Status.Status = "FAILED must provide annotation: " + utils.ServiceAccountAnnotation
		err = r.client.Update(context.TODO(), r.k8sObject)
		if err != nil {
			return r.reconcileResult, err
		}
		return reconcile.Result{Requeue: false}, nil
	}

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

	// check if object is set to be deleted
	deleted := r.k8sObject.GetDeletionTimestamp() != nil

	// if the object is not set for deletion, ensure it contains our finalizer
	pendingFinalizers := r.k8sObject.GetFinalizers()
	finalizerExists := len(pendingFinalizers) > 0
	if !finalizerExists && !deleted && !utils.Contains(pendingFinalizers, finalizer) {
		log.Printf("Adding finalizer to %s %s/%s", r.k8sObject.Kind, r.k8sObject.Namespace, r.k8sObject.Name)
		finalizers := append(pendingFinalizers, finalizer)
		r.k8sObject.SetFinalizers(finalizers)
		err := r.client.Update(context.TODO(), r.k8sObject)
		if err != nil {
			return r.reconcileResult, err
		}
		return reconcile.Result{Requeue: false}, nil
	}

	// fetch the corresponding object from GCE
	gceObject, err := r.read()
	if err != nil {
		return r.reconcileResult, err
	}

	// if it doesn't existin in gcp and is set to be deleted,
	// that means we've already deleted it so we can go ahead
	// and strip out the finalizer to let k8s actually delete it.
	if gceObject == nil && deleted && finalizerExists {
		log.Printf("reconcile: remove finalizer from %s %s/%s", r.k8sObject.Kind, r.k8sObject.Namespace, r.k8sObject.Name)
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
		log.Printf("reconcile: Successfully deleted %s/%s, do not requeue", r.k8sObject.Namespace, r.k8sObject.Name)
		return reconcile.Result{Requeue: false}, nil
	}

	// if object is not set to be deleted and gceObject doesn't exist we should create it.
	if !deleted && gceObject == nil {
		log.Printf("reconcile: creating %s: %s/%s", kind, r.k8sObject.Namespace, r.k8sObject.Name)
		key, err := r.create()

		// Great now we need to create the secret to hold our key
		//spew.Dump(key)
		secretName := request.Name
		if r.annotations["iam.gce/secretName"] != "" {
			secretName = r.annotations["iam.gce/secretName"]
		}
		secretNamespace := request.Namespace
		if r.annotations["iam.gce/secretNamespace"] != "" {
			secretNamespace = r.annotations["iam.gce/secretNamespace"]
		}

		secret := newSecret(secretName, secretNamespace, key.PrivateKeyData)
		// Set AppService instance as the owner and controller
		if err := controllerutil.SetControllerReference(r.k8sObject, secret, r.scheme); err != nil {
			return reconcile.Result{}, err
		}
		// Check if this Secret already exists
		found := &corev1.Secret{}
		err = r.client.Get(context.TODO(), types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace}, found)
		if err != nil && errors.IsNotFound(err) {
			log.Printf("Creating a new secret %s/%s\n", secret.Namespace, secret.Name)
			err = r.client.Create(context.TODO(), secret)
			if err != nil {
				return reconcile.Result{}, err
			}
			// Secret created successfully - don't requeue
			r.k8sObject.Status.Status = "READY"
			r.k8sObject.Status.Name = key.Name
			err = r.client.Update(context.TODO(), r.k8sObject)
			if err != nil {
				return r.reconcileResult, err
			}
			return reconcile.Result{}, nil
		} else if err != nil {
			return reconcile.Result{}, err
		}
		// Secret already exists - don't requeue
		log.Printf("Skip reconcile: Secret %s/%s already exists", found.Namespace, found.Name)
		return reconcile.Result{}, nil
	}

	// if GCE object exists, we need to determine what to do with it.
	if gceObject != nil {

		// If object is set to be deleted and the finalizer exists, that means we still
		// need to ask GCE to delete the resource.
		if deleted && finalizerExists {
			log.Printf("reconcile: deleting %s: %s/%s", kind, r.k8sObject.Namespace, r.k8sObject.Name)
			r.k8sObject.Status.Status = "DELETING"
			err = r.client.Update(context.TODO(), r.k8sObject)
			if err != nil {
				return r.reconcileResult, err
			}
			err := r.destroy()
			if err != nil {
				r.reconcileResult.RequeueAfter, _ = time.ParseDuration("5s")
				return r.reconcileResult, err
			}
			r.reconcileResult.RequeueAfter, _ = time.ParseDuration("5s")
			return r.reconcileResult, err
		}

		// If the resource exists we should reconcile status and outputs
		if r.k8sObject.Status.Status == "READY" {
			log.Printf("reconcile: already exists %s: %s/%s.", kind, r.k8sObject.Namespace, r.k8sObject.Name)
			return reconcile.Result{}, nil
		}
		if r.k8sObject.Status.Status == "FAILED" {
			return reconcile.Result{}, nil
		}
	}
	return reconcile.Result{}, nil
}

func newSecret(name, namespace, key string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"key": []byte(key),
		},
	}
}
