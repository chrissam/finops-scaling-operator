/*
Copyright 2025.

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

package controller

import (
	"context"
	"fmt"
	"sync"
	"time"
	"strconv"
	"strings"

	//"github.com/robfig/cron/v3"
	appsv1 "k8s.io/api/apps/v1"
	//metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	corev1 "k8s.io/api/core/v1"

	finopsv1alpha1 "github.com/chrissam/k8s-scaling-operator/api/v1alpha1"
)

const (
	originalReplicasAnnotation = "finops.dev/original-replicas"
)



// +kubebuilder:rbac:groups=finops.devopsideas.com,resources=finopsscalepolicies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=finops.devopsideas.com,resources=finopsscalepolicies/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=finops.devopsideas.com,resources=finopsscalepolicies/finalizers,verbs=update
// +kubebuilder:rbac:groups=finops.devopsideas.com,resources=finopsscalepolicies,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups=finops.devopsideas.com,resources=finopsoperatorconfigs,verbs=get;list;update;patch;watch
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch

// FinOpsScalePolicyReconciler reconciles a FinOpsScalePolicy object
type FinOpsScalePolicyReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Config *finopsv1alpha1.FinOpsOperatorConfig // Store global config
	mutex  sync.Mutex                           // Protect concurrent Reconciles
	sem    chan struct{}                        // Semaphore for maxParallelOperations
}

// Load global FinOpsOperatorConfig into memory
func (r *FinOpsScalePolicyReconciler) loadConfig(ctx context.Context) error {
	log := log.FromContext(ctx)
	log.Info("Loading FinOpsOperatorConfig...")

	var configList finopsv1alpha1.FinOpsOperatorConfigList
	if err := r.List(ctx, &configList); err != nil {
		log.Error(err, "Failed to list FinOpsOperatorConfig")
		return err
	}

	if len(configList.Items) > 0 {
		r.Config = &configList.Items[0] // Store first found config
		log.Info("Loaded FinOpsOperatorConfig", "config", r.Config.Name)
	} else {
		log.Info("No FinOpsOperatorConfig found, using default values")

		// Set default values
		r.Config = &finopsv1alpha1.FinOpsOperatorConfig{
			Spec: finopsv1alpha1.FinOpsOperatorConfigSpec{
				ExcludedNamespaces:    []string{"kube-system", "kube-public", "kube-node-lease", "local-path-storage"},
				MaxParallelOperations: 5,
				CheckInterval:         "5m",
				ForceScaleDown:        false,
				ForceScaleDownSchedule: nil,
				ForceScaleDownTimezone: "UTC", 
			},
		}
	}

	// Initialize semaphore
	r.sem = make(chan struct{}, r.Config.Spec.MaxParallelOperations)

	return nil
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *FinOpsScalePolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues("Reconcile", req.NamespacedName)

	r.mutex.Lock()
	defer r.mutex.Unlock()

	log.Info("Reconciling...")

	// Load global FinOpsOperatorConfig
	if r.Config == nil {
		if err := r.loadConfig(ctx); err != nil {
			log.Error(err, "Failed to load FinOpsOperatorConfig")
			return ctrl.Result{}, err
		}
	}

	// Handle FinOpsOperatorConfig changes
	if req.NamespacedName.Name == "global-config" && req.NamespacedName.Namespace == "scaling-operator-system" {
		log.Info("Detected FinOpsOperatorConfig change, updating settings")
		if err := r.loadConfig(ctx); err != nil {
			log.Error(err, "Failed to reload FinOpsOperatorConfig")
			return ctrl.Result{}, err
		}
		// Force immediate requeue after config change
		return r.reconcileAllNamespaces(ctx) 
	}

	// Trigger periodic scan or specific namespace reconcile
	if req.NamespacedName.Name == "" && req.NamespacedName.Namespace == "" {
		return r.reconcileAllNamespaces(ctx)
	}

	if err := r.reconcileNamespace(ctx, req.NamespacedName.Namespace); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *FinOpsScalePolicyReconciler) reconcileAllNamespaces(ctx context.Context) (ctrl.Result, error) {
    log := log.FromContext(ctx).WithValues("ReconcileAll", "cluster")
    log.Info("Starting cluster-wide scan...")

    var wg sync.WaitGroup
    errChan := make(chan error) //  No buffer needed, will block if no receiver

    if r.Config.Spec.ForceScaleDown && r.Config.Spec.ForceScaleDownSchedule != nil {
        var namespaces corev1.NamespaceList
        if err := r.List(ctx, &namespaces); err != nil {
            log.Error(err, "Failed to list Namespaces")
            return ctrl.Result{}, err
        }

        for _, namespace := range namespaces.Items {
            if r.isNamespaceExcluded(namespace.Name) {
                log.V(1).Info("Skipping excluded namespace", "namespace", namespace.Name)
                continue
            }

            wg.Add(1)
            go func(ns string) {
                defer wg.Done()
                r.sem <- struct{}{}
                defer func() { <-r.sem }()

                var policies finopsv1alpha1.FinOpsScalePolicyList
                if err := r.List(ctx, &policies, client.InNamespace(ns)); err != nil {
                    errChan <- fmt.Errorf("failed to list FinOpsScalePolicies in namespace %s: %w", ns, err)
                    return
                }

				scalingTime := false
				if r.Config.Spec.ForceScaleDownSchedule != nil { // Add nil check
					tempSchedule := []*finopsv1alpha1.ScheduleSpec{
						{
							Days:      r.Config.Spec.ForceScaleDownSchedule.Days,
							StartTime: r.Config.Spec.ForceScaleDownSchedule.StartTime,
							EndTime:   r.Config.Spec.ForceScaleDownSchedule.EndTime,
						},
					}
					scalingTime = r.isScalingTime(tempSchedule, r.Config.Spec.ForceScaleDownTimezone, time.Now())
				}

                if len(policies.Items) == 0 { // No policy
                    if scalingTime {
                        log.Info("ForceScaleDown enabled and no policy found, scaling down namespace", "namespace", ns)
                        if err := r.manageDeploymentsWithoutPolicy(ctx, ns, true); err != nil {
                            errChan <- err
                        }
                    } else {
                        log.V(1).Info("ForceScaleDown enabled, no policy, scaling up namespace", "namespace", ns)
                        if err := r.manageDeploymentsWithoutPolicy(ctx, ns, false); err != nil {
                            errChan <- err
                        }
                    }

                } else { // Policy exists
                    log.V(1).Info("FinOpsScalePolicy found, using policy to reconcile namespace", "namespace", ns)
                    if err := r.reconcileNamespace(ctx, ns); err != nil {
                        errChan <- err
                    }
                }
            }(namespace.Name)
        }

    } else {
        // Original logic: Reconcile namespaces with FinOpsScalePolicies
        var policies finopsv1alpha1.FinOpsScalePolicyList
        if err := r.List(ctx, &policies); err != nil {
            log.Error(err, "Failed to list FinOpsScalePolicies")
            return ctrl.Result{}, err
        }

        namespaces := map[string]bool{}
        for _, policy := range policies.Items {
            namespaces[policy.Namespace] = true
        }

        for ns := range namespaces {
            wg.Add(1)
            go func(namespace string) {
                defer wg.Done()
                r.sem <- struct{}{}
                defer func() { <-r.sem }()
                if err := r.reconcileNamespace(ctx, namespace); err != nil {
                    errChan <- err
                }
            }(ns)
        }
    }

    wg.Wait()
    close(errChan)

    for err := range errChan {
        if err != nil {
            log.Error(err, "Error during namespace reconciliation", "error", err)
            return ctrl.Result{}, err
        }
    }

    interval, err := time.ParseDuration(r.Config.Spec.CheckInterval)
    if err != nil {
        log.Error(err, "Invalid CheckInterval, using default 5m")
        interval = 5 * time.Minute
    }

    log.Info("Cluster-wide scan completed.", "CheckInterval", interval)
    return ctrl.Result{RequeueAfter: interval}, nil
}


func (r *FinOpsScalePolicyReconciler) manageDeploymentsWithoutPolicy(ctx context.Context, namespace string, forceScaleDown bool) error {
    log := log.FromContext(ctx).WithValues("manageDeploymentsWithoutPolicy", namespace)
    log.Info("Entering manageDeploymentsWithoutPolicy", "namespace", namespace, "forceScaleDown", forceScaleDown)

    var deployments appsv1.DeploymentList
    if err := r.List(ctx, &deployments, client.InNamespace(namespace)); err != nil {
        log.Error(err, "Failed to list deployments in namespace", "namespace", namespace)
        return err
    }

    for _, deployment := range deployments.Items {
        if r.isDeploymentExcluded(deployment.Namespace, deployment.Name) {
            log.Info("Skipping excluded deployment", "deployment", deployment.Name, "namespace", deployment.Namespace)
            continue
        }

        originalReplicas, err := r.getOriginalReplicas(&deployment)
        if err != nil {
            log.Error(err, "Failed to get original replicas annotation", "deployment", deployment.Name, "namespace", namespace)
            return err
        }

        if forceScaleDown {
            minReplicas := int32(0) // Or get from config if needed

            if originalReplicas == nil && *deployment.Spec.Replicas > minReplicas {
                // Not yet scaled down, store original and scale down
                log.Info("Original replicas not found, storing and scaling down", "deployment", deployment.Name, "namespace", namespace, "replicas", *deployment.Spec.Replicas)
                if err := r.storeOriginalReplicas(&deployment, *deployment.Spec.Replicas); err != nil {
                    log.Error(err, "Failed to store original replicas annotation", "deployment", deployment.Name, "namespace", namespace)
                    return err
                }
                if err := r.scaleDeployment(ctx, &deployment, minReplicas); err != nil {
                    log.Error(err, "Failed to scale down deployment", "deployment", deployment.Name, "namespace", namespace, "minReplicas", minReplicas)
                    return err
                }

            } else if originalReplicas != nil && *deployment.Spec.Replicas != 0 {
                // Already scaled down, ensure it remains scaled down
                log.V(1).Info("Already scaled down, ensuring it remains so", "deployment", deployment.Name, "namespace", namespace)
                if err := r.scaleDeployment(ctx, &deployment, minReplicas); err != nil {
                    log.Error(err, "Failed to scale down deployment", "deployment", deployment.Name, "namespace", namespace, "minReplicas", minReplicas)
                    return err
                }
            }

        } else {
            // ForceScaleDown is false (or schedule is not active) - scale back up
            if originalReplicas != nil {
                log.Info("ForceScaleDown is false, scaling up", "deployment", deployment.Name, "namespace", namespace, "originalReplicas", *originalReplicas)
                if err := r.scaleDeployment(ctx, &deployment, *originalReplicas); err != nil {
                    log.Error(err, "Failed to scale up deployment", "deployment", deployment.Name, "namespace", namespace, "replicas", *originalReplicas)
                    return err
                }
                if err := r.clearOriginalReplicas(&deployment); err != nil {
                    log.Error(err, "Failed to clear original replicas annotation", "deployment", deployment.Name, "namespace", namespace)
                    return err
                }
            } else {
                log.V(1).Info("No original replicas found, skipping scale-up", "deployment", deployment.Name, "namespace", namespace)
            }
        }
    }
    return nil
}

func (r *FinOpsScalePolicyReconciler) reconcileNamespace(ctx context.Context, namespace string) error {
	log := log.FromContext(ctx).WithValues("ReconcileNamespace", namespace)
	log.Info("Processing namespace...")

	// Check namespace exclusion
	for _, excludedNS := range r.Config.Spec.ExcludedNamespaces {
		if namespace == excludedNS {
			log.Info("Skipping excluded namespace", "namespace", namespace)
			return nil
		}
	}

	var policies finopsv1alpha1.FinOpsScalePolicyList
	if err := r.List(ctx, &policies, client.InNamespace(namespace)); err != nil {
		log.Error(err, "Failed to list FinOpsScalePolicies in namespace", "namespace", namespace)
		return err
	}

	for _, policy := range policies.Items {
		if policy.Spec.OptOut {
			log.Info("OptOut is true, skipping scaling for this namespace", "namespace", namespace)
			return nil
		}

		var deployments appsv1.DeploymentList
		if err := r.List(ctx, &deployments, client.InNamespace(namespace)); err != nil {
			log.Error(err, "Failed to list deployments in namespace", "namespace", namespace)
			return err
		}

		for _, deployment := range deployments.Items {
			if r.isDeploymentExcluded(deployment.Namespace, deployment.Name) {
				log.Info("Skipping excluded deployment", "deployment", deployment.Name, "namespace", deployment.Namespace)
				continue
			}

			var schedules []*finopsv1alpha1.ScheduleSpec // Correct type
			minReplicas := int32(0) // Default to 0 if not set
			depOptOut := false

			if policy.Spec.DefaultSchedule != nil {
				schedules = append(schedules, policy.Spec.DefaultSchedule)
			}

			for _, depPolicy := range policy.Spec.Deployments {
				if depPolicy.Name == deployment.Name && depPolicy.Schedule != nil {
					schedules = []*finopsv1alpha1.ScheduleSpec{depPolicy.Schedule} // Override with deployment-specific schedule
					minReplicas = depPolicy.MinReplicas
					depOptOut = depPolicy.OptOut
					break
				} else if depPolicy.Name == deployment.Name {
					minReplicas = depPolicy.MinReplicas
					break
				}
			}

			if depOptOut { // Check the optOut field for Deployment
                log.Info("Deployment opted out of scaling", "deployment", deployment.Name, "namespace", namespace)
                continue // Skip to the next deployment
            }

			if r.isScalingTime(schedules, policy.Spec.Timezone, time.Now()) {
				log.Info("Scaling time is true", "deployment", deployment.Name, "namespace", deployment.Namespace)
				originalReplicas, err := r.getOriginalReplicas(&deployment)
				if err != nil {
					log.Error(err, "Failed to get original replicas annotation", "deployment", deployment.Name, "namespace", deployment.Namespace)
					return err
				}
				if originalReplicas == nil && *deployment.Spec.Replicas > minReplicas {
					log.Info("Original replicas not found, storing...", "deployment", deployment.Name, "namespace", deployment.Namespace, "replicas", *deployment.Spec.Replicas)
					if err := r.storeOriginalReplicas(&deployment, *deployment.Spec.Replicas); err != nil {
						log.Error(err, "Failed to store original replicas annotation", "deployment", deployment.Name, "namespace", deployment.Namespace)
						return err
					}
					if err := r.scaleDeployment(ctx, &deployment, minReplicas); err != nil {
						log.Error(err, "Failed to scale down deployment", "deployment", deployment.Name, "namespace", deployment.Namespace, "minReplicas", minReplicas)
						return err
					}
					log.Info("Successfully scaled down deployment", "deployment", deployment.Name, "namespace", deployment.Namespace, "minReplicas", minReplicas)
				} else if originalReplicas != nil && *deployment.Spec.Replicas > minReplicas {
					log.Info("Original replicas found", "deployment", deployment.Name, "namespace", deployment.Namespace, "originalReplicas", *originalReplicas)
					if err := r.scaleDeployment(ctx, &deployment, minReplicas); err != nil {
						log.Error(err, "Failed to scale down deployment", "deployment", deployment.Name, "namespace", deployment.Namespace, "minReplicas", minReplicas)
						return err
					}
					log.Info("Successfully scaled down deployment", "deployment", deployment.Name, "namespace", deployment.Namespace, "minReplicas", minReplicas)

				} else {
					log.Info("Original replicas not found", "deployment", deployment.Name, "namespace", deployment.Namespace)
				}
			} else {
				log.Info("Scaling time is false", "deployment", deployment.Name, "namespace", deployment.Namespace)
				originalReplicas, err := r.getOriginalReplicas(&deployment)
				if err != nil {
					log.Error(err, "Failed to get original replicas annotation", "deployment", deployment.Name, "namespace", deployment.Namespace)
					return err
				}
				if originalReplicas != nil && *deployment.Spec.Replicas != *originalReplicas {
					log.Info("Original replicas changed, scaling up...", "deployment", deployment.Name, "namespace", deployment.Namespace, "originalReplicas", *originalReplicas, "currentReplicas", *deployment.Spec.Replicas)
					if err := r.scaleDeployment(ctx, &deployment, *originalReplicas); err != nil {
						log.Error(err, "Failed to scale up deployment", "deployment", deployment.Name, "namespace", deployment.Namespace)
						return err
					}
					log.Info("Successfully scaled up deployment", "deployment", deployment.Name, "namespace", deployment.Namespace, "originalReplicas", *originalReplicas)
					if err := r.clearOriginalReplicas(&deployment); err != nil {
						log.Error(err, "Failed to clear original replicas annotation", "deployment", deployment.Name, "namespace", deployment.Namespace)
						return err
					}
					log.Info("Successfully scaled up deployment", "deployment", deployment.Name, "namespace", deployment.Namespace)
				} else {
					log.Info("No scaling required for deployment", "deployment", deployment.Name, "namespace", deployment.Namespace)
				}
			}
		}
	}
	log.Info("Finished processing namespace", "namespace", namespace)
	return nil
}


// isNamespaceExcluded checks if a namespace is globally excluded
func (r *FinOpsScalePolicyReconciler) isNamespaceExcluded(namespace string) bool {
    for _, excluded := range r.Config.Spec.ExcludedNamespaces {
        if excluded == namespace {
            return true
        }
    }
    return false
}

// isDeploymentExcluded checks if a deployment is globally excluded
func (r *FinOpsScalePolicyReconciler) isDeploymentExcluded(namespace, name string) bool {
	for _, excluded := range r.Config.Spec.ExcludedDeployments {
		if excluded.Namespace == namespace && excluded.Name == name {
			return true
		}
	}
	return false
}

// isScalingTime checks if the current time matches any of the provided time ranges.
func (r *FinOpsScalePolicyReconciler) isScalingTime(schedules []*finopsv1alpha1.ScheduleSpec, timezone string, now time.Time) bool {
    log := log.FromContext(context.Background())
    loc, err := time.LoadLocation(timezone)
    if err != nil {
        log.Error(err, "Failed to load timezone, skipping the namespace", "timezone", timezone)
		return false
    }

    for _, schedule := range schedules {
        if schedule == nil {
            continue // Skip nil schedules
        }

        if r.isDayActive(schedule.Days, now.In(loc)) && r.isTimeWithinRange(schedule.StartTime, schedule.EndTime, now.In(loc)) {
            log.Info("Current time is within schedule", "schedule", schedule, "now", now.In(loc))
            return true
        }
        log.Info("Current time is NOT within schedule", "schedule", schedule, "now", now.In(loc))
    }

    return false
}

// isDayActive checks if the given day is active in the schedule.
func (r *FinOpsScalePolicyReconciler) isDayActive(days []string, now time.Time) bool {
    log := log.FromContext(context.Background())
    log.Info("Entering isDayActive", "days", days, "now.Weekday()", now.Weekday().String()[:3]) // Log input

    for _, day := range days {
        if day == "*" || strings.EqualFold(day, now.Weekday().String()[:3]) {
            log.Info("Day is active", "day", day) // Log active day
            return true
        }
    }
    log.Info("Day is not active") // Log inactive
    return false
}

// isTimeWithinRange checks if the given time is within the start and end times.
func (r *FinOpsScalePolicyReconciler) isTimeWithinRange(startTimeStr string, endTimeStr string, now time.Time) bool {
    startTime, err := time.ParseInLocation("15:04", startTimeStr, now.Location())
    if err != nil {
        log := log.FromContext(context.Background())
        log.Error(err, "Invalid start time format", "startTime", startTimeStr)
        return false
    }

    endTime, err := time.ParseInLocation("15:04", endTimeStr, now.Location())
    if err != nil {
        log := log.FromContext(context.Background())
        log.Error(err, "Invalid end time format", "endTime", endTimeStr)
        return false
    }

    // Set the date of startTime and endTime to the date of now for comparison
    start := time.Date(now.Year(), now.Month(), now.Day(), startTime.Hour(), startTime.Minute(), 0, 0, now.Location())
    end := time.Date(now.Year(), now.Month(), now.Day(), endTime.Hour(), endTime.Minute(), 0, 0, now.Location())

    if endTime.Before(startTime) {
        // End time is on the next day
        endOfNextDay := end.AddDate(0, 0, 1)
        if now.After(start) || (now.Before(end) && now.Day() != endOfNextDay.Day()) || (now.Before(endOfNextDay) && now.Day() == endOfNextDay.Day()) {
            return true
        }
    } else {
        // startTime and endTime are on the same day
        if now.After(start) && now.Before(end) {
            return true
        }
    }

    return false
}

// scaleDeployment scales a deployment to the specified number of replicas
func (r *FinOpsScalePolicyReconciler) scaleDeployment(ctx context.Context, deployment *appsv1.Deployment, replicas int32) error {
	log := log.FromContext(ctx)

	patch := client.MergeFrom(deployment.DeepCopy())
	deployment.Spec.Replicas = &replicas
	if err := r.Patch(ctx, deployment, patch); err != nil {
		return err
	}

	log.Info("Scaled deployment", "deployment", deployment.Name, "namespace", deployment.Namespace, "replicas", replicas)
	return nil
}

// getOriginalReplicas retrieves the original replica count for a deployment from its annotations.
func (r *FinOpsScalePolicyReconciler) getOriginalReplicas(deployment *appsv1.Deployment) (*int32, error) {
    log := log.FromContext(context.Background()).WithValues("getOriginalReplicas", deployment.Name, "namespace", deployment.Namespace) // ADDED
    log.Info("Entering getOriginalReplicas")                                                                                   // ADDED
    log.Info("Deployment Annotations", "annotations", deployment.Annotations)                                                 // ADDED
    if val, ok := deployment.Annotations[originalReplicasAnnotation]; ok {
        log.Info("Annotation found", "value", val) // ADDED
        replicas, err := parseInt32(val)
        if err != nil {
            log.Error(err, "Invalid value for annotation", "annotation", originalReplicasAnnotation, "error", err)
            return nil, fmt.Errorf("invalid value for annotation %s: %w", originalReplicasAnnotation, err)
        }
        log.Info("Parsed replicas", "replicas", replicas) // ADDED
        return &replicas, nil
    }
    log.Info("Annotation not found") // ADDED
    return nil, nil
}


// storeOriginalReplicas stores the original replica count for a deployment in its annotations.
func (r *FinOpsScalePolicyReconciler) storeOriginalReplicas(deployment *appsv1.Deployment, replicas int32) error {
    log := log.FromContext(context.Background()).WithValues("storeOriginalReplicas", deployment.Name, "namespace", deployment.Namespace)
    log.Info("Entering storeOriginalReplicas", "replicas", replicas)

    // Fetch the latest version of the deployment to avoid cache inconsistencies
    latestDeployment := &appsv1.Deployment{}
    err := r.Get(context.Background(), client.ObjectKey{Name: deployment.Name, Namespace: deployment.Namespace}, latestDeployment)
    if err != nil {
        log.Error(err, "Failed to get latest deployment")
        return err
    }

    // Log existing annotations
    if latestDeployment.Annotations != nil {
        log.Info("Existing annotations", "annotations", latestDeployment.Annotations)
    } else {
        log.Info("No existing annotations")
    }

    // Create a deep copy for modification
    patchedDeployment := latestDeployment.DeepCopy()

    // Ensure annotations map is initialized
    if patchedDeployment.Annotations == nil {
        patchedDeployment.Annotations = make(map[string]string)
    }

    // Set the annotation
    annotationValue := fmt.Sprintf("%d", replicas)
    patchedDeployment.Annotations[originalReplicasAnnotation] = annotationValue
    log.Info("Set/Updated annotation", "annotation", originalReplicasAnnotation, "value", annotationValue)

    // Apply the patch
    err = r.Patch(context.Background(), patchedDeployment, client.MergeFrom(latestDeployment))
    if err != nil {
        log.Error(err, "Failed to patch deployment with annotation")
        return err
    }

    log.Info("Successfully patched deployment with annotation")
    return nil
}


// clearOriginalReplicas removes the stored original replica count for a deployment from its annotations.
func (r *FinOpsScalePolicyReconciler) clearOriginalReplicas(deployment *appsv1.Deployment) error {
    log := log.FromContext(context.Background()).WithValues("clearOriginalReplicas", deployment.Name, "namespace", deployment.Namespace)
    log.Info("Entering clearOriginalReplicas")

    if deployment.Annotations == nil {
        log.Info("No annotations present, skipping deletion")
        return nil
    }

    // Create a deep copy to avoid modifying the original object directly
    patchedDeployment := deployment.DeepCopy()

    // Remove the annotation
    if _, exists := patchedDeployment.Annotations[originalReplicasAnnotation]; exists {
        delete(patchedDeployment.Annotations, originalReplicasAnnotation)
        log.Info("Deleted annotation", "annotation", originalReplicasAnnotation)

        // Use StrategicMergeFrom to properly patch the object
        err := r.Patch(context.Background(), patchedDeployment, client.StrategicMergeFrom(deployment))
        if err != nil {
            log.Error(err, "Failed to patch deployment to remove annotation")
            return err
        }

        log.Info("Successfully patched deployment to remove annotation")
        return nil
    }

    log.Info("Annotation not found, skipping deletion")
    return nil
}


// parseInt32 helper function to parse string to int32
func parseInt32(s string) (int32, error) {
    parsed, err := strconv.ParseInt(s, 10, 32) // Base 10, 32-bit
    if err != nil {
        return 0, err
    }
    return int32(parsed), nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *FinOpsScalePolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&finopsv1alpha1.FinOpsScalePolicy{}).
		Named("finopsscalepolicy").
		Watches(&finopsv1alpha1.FinOpsOperatorConfig{}, &handler.EnqueueRequestForObject{}).
		Complete(r)
}