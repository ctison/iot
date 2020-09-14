package controllers

import (
	"context"
	"encoding/json"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/go-logr/logr"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/ctison/iot/fridge/fridge"
	iotv1 "github.com/ctison/iot/operator/api/v1"
)

// FridgeReconciler reconciles a Fridge object
type FridgeReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	// Store background jobs' stop channels.
	fridges map[string]func()
	MQTT    mqtt.Client
}

func (r *FridgeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&iotv1.Fridge{}).
		Complete(r)
}

// +kubebuilder:rbac:groups=iot.ctison.dev,resources=fridges,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=iot.ctison.dev,resources=fridges/status,verbs=get;update;patch

func (r *FridgeReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("fridge", req.NamespacedName)

	if r.fridges == nil {
		r.fridges = map[string]func(){}
	}

	// Retrieve the fridge.
	fridge := &iotv1.Fridge{}
	if err := r.Get(ctx, req.NamespacedName, fridge); err != nil {
		log.Error(err, "failed to fetch resource")
		return ctrl.Result{Requeue: true}, err
	}

	namespacedName := req.NamespacedName.String()
	finalizerName := "fridge.finalizers." + iotv1.GroupVersion.Group

	// Setup finalizer if not present.
	if fridge.DeletionTimestamp.IsZero() {
		if !containsString(fridge.Finalizers, finalizerName) {
			fridge.Finalizers = append(fridge.Finalizers, finalizerName)
			if err := r.Update(ctx, fridge); err != nil {
				log.Error(err, "failed to update resource")
				return ctrl.Result{}, err
			}
		}
	} else {
		if containsString(fridge.Finalizers, finalizerName) {
			log.Info("stop monitoring", zap.String("topic", fridge.Spec.Topic))
			if cancel, ok := r.fridges[namespacedName]; ok {
				cancel()
				delete(r.fridges, namespacedName)
			}
			fridge.Finalizers = removeString(fridge.Finalizers, finalizerName)
			if err := r.Update(ctx, fridge); err != nil {
				log.Error(err, "failed to update resource")
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Run fridgeMonitorLoop if none is running yet.
	if _, ok := r.fridges[namespacedName]; !ok {
		log.Info("start monitoring", "topic", fridge.Spec.Topic)
		ctx, cancel := context.WithCancel(context.Background())
		r.fridges[namespacedName] = cancel
		go r.fridgeMonitorLoop(ctx, log, fridge.Spec.Topic)
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

func (r *FridgeReconciler) fridgeMonitorLoop(ctx context.Context, log logr.Logger, topic string) {
	log = log.WithName("monitor").WithName(topic)
	_ = r.MQTT.Subscribe(topic, 0, func(client mqtt.Client, msg mqtt.Message) {
		var fridge fridge.FridgeModel
		if err := json.Unmarshal(msg.Payload(), &fridge); err != nil {
			log.Error(err, "failed to parse message")
			return
		}
		log.Info(string(msg.Payload()))
		if fridge.IsDoorOpen {
			_ = r.MQTT.Publish(topic+"/alert", 1, false, "close the door!")
		}
	})
	<-ctx.Done()
	r.MQTT.Unsubscribe(topic)
}

// Helper function to check if slice contains s.
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

// Helper function to delete s from slice.
func removeString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}
