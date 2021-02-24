package controller

import (
	"context"
	"github.com/go-logr/logr"
	"github.com/thedevsaddam/gojsonq"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

type Workload struct {
	client.Client
	Log    logr.Logger
	Object client.Object
}

func (w *Workload) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var object = &unstructured.Unstructured{}
	object.SetGroupVersionKind(w.Object.GetObjectKind().GroupVersionKind())
	err := w.Client.Get(ctx, types.NamespacedName{Name: req.Name, Namespace: req.Namespace}, object)
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return ctrl.Result{}, err
		} else {
			return ctrl.Result{}, err
		}
	}
	jsonData, err := object.MarshalJSON()
	if err != nil {
		w.Log.Error(err, "get json data error")
	}

	data := gojsonq.New().FromString(string(jsonData)).Find("spec.template.spec.containers")
	if data == nil {
		data = gojsonq.New().FromString(string(jsonData)).Find("spec.containers")
	}
	//TODO: get data as array and get image ,and get secret and update the json data object
	return ctrl.Result{}, nil
}

func (w *Workload) SetupWithManager(ctx context.Context, mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(w.Object).WithEventFilter(predicate.Funcs{
		CreateFunc: func(event event.CreateEvent) bool {
			return true
		},
		UpdateFunc: func(updateEvent event.UpdateEvent) bool {
			return true
		},
		DeleteFunc: func(deleteEvent event.DeleteEvent) bool {
			return true
		},
	}).Complete(w)
}
