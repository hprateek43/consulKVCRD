/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"time"

	consulv1alpha1 "code.pan.run/prisma-saas/ConsulKVCRD/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"gopkg.in/yaml.v2"
)

const consulKVFinalizer = "consul.panw.com/finalizer"
const reconcileRequeueDuration = 60

// ConsulKVReconciler reconciles a ConsulKV object
type ConsulKVReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Log      logr.Logger
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=consul.panw.com,resources=consulkvs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=consul.panw.com,resources=consulkvs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=consul.panw.com,resources=consulkvs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ConsulKV object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.1/pkg/reconcile
func (r *ConsulKVReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	reqLogger := r.Log.WithValues("consulKV", req.NamespacedName)
	reqLogger.Info("Reconciling ConsulKV")

	// Lookup the ConsulKV instance for this reconcile request
	consulKVinstance := &consulv1alpha1.ConsulKV{}
	err := r.Get(ctx, req.NamespacedName, consulKVinstance)

	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			reqLogger.Info("ConsulKV resource not found. Ignoring since object must be deleted.")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		reqLogger.Error(err, "Failed to get ConsulKV.")
		return ctrl.Result{}, err
	}

	r.Recorder.Event(consulKVinstance, "Normal", "ReconcileStarted", "Started reconciliation for ConsulKV")

	//new logic
	rawYaml := make(map[interface{}]interface{})
	flattenKeys := make(map[string]string)

	err = yaml.Unmarshal([]byte(consulKVinstance.Spec.Keys), &rawYaml)
	FlattenNestedKeysToPath("", rawYaml, flattenKeys)

	consulKVinstance.Status.Keys = nil
	// convert nested map[interface{}]interface{} to flat map[string]string. This generates direct PUT paths for consul client
	for key, value := range flattenKeys {
		SyncKeyValueToConsul(key, value, reqLogger)
		consulKVinstance.Status.Keys = append(consulKVinstance.Status.Keys, key)
	}

	//publish event to CRD instance
	r.Recorder.Event(consulKVinstance, "Normal", "ReconcileKey", "All keys reconciled")

	isConsulKVMarkedToBeDeleted := consulKVinstance.GetDeletionTimestamp() != nil
	if isConsulKVMarkedToBeDeleted {
		if controllerutil.ContainsFinalizer(consulKVinstance, consulKVFinalizer) {
			reqLogger.Info("Delete request for consulKV")
			// Run finalization logic for memcachedFinalizer. If the
			// finalization logic fails, don't remove the finalizer so
			// that we can retry during the next reconciliation.
			if err := r.finalizeConsulKV(reqLogger, consulKVinstance); err != nil {
				return ctrl.Result{}, err
			}

			// Remove memcachedFinalizer. Once all finalizers have been
			// removed, the object will be deleted.
			controllerutil.RemoveFinalizer(consulKVinstance, consulKVFinalizer)
			err := r.Update(ctx, consulKVinstance)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer for this CR
	if !controllerutil.ContainsFinalizer(consulKVinstance, consulKVFinalizer) {
		controllerutil.AddFinalizer(consulKVinstance, consulKVFinalizer)
		err = r.Update(ctx, consulKVinstance)
		if err != nil {
			reqLogger.Error(err, "Failed to add finalizer for CRD. Deletion may not guarantee cleanup.")
			return ctrl.Result{}, err
		}
	}

	// add the last sync timestamp. This can be used in future to detect drifted instances of CRD or CRDs that are failing reconciliation
	// add an operator to watch such CRDs and retry queue to more robust design
	consulKVinstance.Status.LastSynced = time.Now().Format(time.RFC850)
	err = r.Status().Update(ctx, consulKVinstance)
	if err != nil {
		reqLogger.Error(err, "Reconciliation failed. Failed to publish event update for CRD.")
		return ctrl.Result{}, err
	}

	consulKVinstance.Annotations["lastData"] = consulKVinstance.Spec.Keys
	consulKVinstance.SetAnnotations(consulKVinstance.Annotations)

	r.Recorder.Event(consulKVinstance, "Normal", "ReconcileFinished", "Finished reconciliation for ConsulKV")

	// requeue the instance of the crd for next reconciliation
	return ctrl.Result{Requeue: true, RequeueAfter: reconcileRequeueDuration * time.Second}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ConsulKVReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&consulv1alpha1.ConsulKV{}).
		Complete(r)
}

func (r *ConsulKVReconciler) finalizeConsulKV(reqLogger logr.Logger, c *consulv1alpha1.ConsulKV) error {
	// needs to do before the CR can be deleted. Examples
	// of finalizers include performing backups and deleting
	// resources that are not owned by this CR, like a PVC.

	//new logic
	rawYaml := make(map[interface{}]interface{})
	flattenKeys := make(map[string]string)

	err := yaml.Unmarshal([]byte(c.Spec.Keys), &rawYaml)
	if err != nil {
		reqLogger.Error(err, "Failed to load key pair sets for deletion")
	}
	FlattenNestedKeysToPath("", rawYaml, flattenKeys)

	fmt.Printf("--- m:\n%v\n\n", flattenKeys)

	for key := range flattenKeys {
		reqLogger.Info("Attempting to delete key: " + key)
		DeleteKeysFromConsul(key, reqLogger)
	}

	reqLogger.Info("Successfully finalized consulkv")
	return nil
}
