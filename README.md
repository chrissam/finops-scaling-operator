#   FinOps Scaling Operator

   This Kubernetes operator automates scaling of Deployments based on customizable schedules, helping to optimize resource utilization and reduce costs in your Kubernetes clusters.

##   Purpose

   The FinOps Scaling Operator allows you to define policies that automatically scale Kubernetes Deployments up or down at specified times. This is useful for:

   * Cost Optimization: Scaling down non-critical applications during off-peak hours to consume fewer resources.
   * Resource Management: Ensuring that applications have the necessary resources during peak times while freeing up resources when demand is low.
   * Automation: Automating scaling tasks, reducing the need for manual intervention.

##   Key Features

   * Customizable Schedules: Define scaling schedules with specific days and time ranges.
   * Timezone Support: Specify the timezone for your schedules to ensure accurate scaling.
   * Granular Control: Configure scaling policies at the namespace or individual Deployment level.
   * Exclusion Rules: Exclude specific namespaces or Deployments from scaling.
   * Forceful Scaling: Optionally enforce scaling in namespaces even without specific policies, based on a global schedule.
   * Operator Configuration: Global configuration options to control the operator's behavior.

##   Custom Resources

   The operator introduces the following Custom Resources:

   * `FinOpsScalePolicy`: Defines the scaling schedules and target Deployments.
   * `FinOpsOperatorConfig`: Configures the operator's global settings (e.g., excluded namespaces, scaling interval).

##   Installation



   There are two primary ways to install the FinOps Scaling Operator:

   ###   1.  Using `make` (Kubebuilder)

   If you have cloned the operator's source code, you can use the `make` commands provided by Kubebuilder to build and deploy the operator.

   **Important:** This operator uses webhooks for validating or mutating requests. Therefore, cert-manager is a mandatory prerequisite to manage the required certificates.

   **Prerequisites:**

   * `kubectl`: The Kubernetes CLI tool.
   * `make`: The GNU Make build automation tool.
   * Go: The Go programming language, version 1.23.0 or later.
   * A Kubernetes cluster (e.g., Minikube, kind, or a cloud-based cluster).
   * `cert-manager`: cert-manager must be installed in your cluster to manage certificates for the webhooks. Follow the [cert-manager installation instructions](https://cert-manager.io/docs/installation/) before proceeding.

   **Steps:**

   1.  Clone the repository:

       ```bash
       git clone <your-repository-url>
       cd <your-operator-directory>
       ```

   2.  *(Optional)* If you need to modify the deployment (e.g., change the target namespace), edit the `config/manager/kustomization.yaml` file. [cite: 70, 71]

   3.  Build and push the operator image (replace `<your-image>` with your desired image name and repository):

       ```bash
       make docker-build docker-push IMG=<your-image>
       ```

   4.  Deploy the operator to your Kubernetes cluster:

       ```bash
       make deploy
       ```

       This will install the CRDs, webhooks, and deploy the operator's controller manager.

   ###   2.  Using Helm (Recommended)

   The recommended way to install the FinOps Scaling Operator is using the Helm chart. Helm simplifies the deployment and management of the operator.

   **Important:** This operator uses webhooks for validating or mutating requests. Therefore, cert-manager is a mandatory prerequisite to manage the required certificates.

   **Prerequisites:**

   * `kubectl`: The Kubernetes CLI tool.
   * `helm`: The Helm CLI tool (version X.Y.Z or later - check `helm version`).
   * A Kubernetes cluster (e.g., Minikube, kind, or a cloud-based cluster).
   * `cert-manager`: cert-manager must be installed in your cluster to manage certificates for the webhooks. Follow the [cert-manager installation instructions](https://cert-manager.io/docs/installation/) before proceeding.

   **Steps:**

   1.  *(If you haven't packaged the chart yet)* Package the Helm chart:

       ```bash
       helm package ./charts/finops-scaling-operator
       ```

       This will create a `.tgz` file containing your chart.

   2.  Install the Helm chart:

       ```bash
       helm install <release-name> <path-to-chart.tgz> -n <operator-namespace>
       ```

       * Replace `<release-name>` with a name for this installation of the operator.
       * Replace `<path-to-chart.tgz>` with the path to the `.tgz` file created in the previous step.
       * Replace `<operator-namespace>` with the namespace where you want to deploy the operator (e.g., `scaling-operator-system`). If the namespace doesn't exist, you may need to create it first. [cite: 69]

   3.  *(Optional)* Customize the installation by providing a `values.yaml` file:

       ```bash
       helm install <release-name> <path-to-chart.tgz> -f <path-to-my-values.yaml> -n <operator-namespace>
       ```

       This allows you to override the default configuration values in the chart.

   **Post-Installation**

   After installing the operator, you can create `FinOpsScalePolicy` and `FinOpsOperatorConfig` resources to define your scaling policies. Refer to the [Configuration](#configuration) section for details.

   **Important Notes:**

   * The operator requires appropriate RBAC permissions to manage Deployments in the namespaces where you create `FinOpsScalePolicy` resources. The `make deploy` command and the Helm chart should handle this, but be sure to verify.


##   Configuration

###   FinOpsOperatorConfig

   The `FinOpsOperatorConfig` CR allows you to configure the operator's global behavior. This is usually managed by the cluster admins. Only one instance of `FinOpsOperatorConfig` can be created in a cluster. Here's an example:

   ```yaml
   apiVersion: finops.devopsideas.com/v1alpha1
   kind: FinOpsOperatorConfig
   metadata:
     name: global-config
     namespace: scaling-operator-system # Operator's namespace
   spec:
     excludedNamespaces:
       - kube-system
       - kube-public
       - cert-manager
     excludedDeployments:
       - name: special-app
         namespace: production
     maxParallelOperations: 5
     checkInterval: "5m"
     forceScaleDown: false
     forceScaleDownSchedule:
       days: ["Mon", "Tue", "Wed", "Thu", "Fri"]
       startTime: "18:00"
       endTime: "08:00"
     forceScaleDownTimezone: "America/New_York"
```

**Configuration Options:**

-   `excludedNamespaces`: A list of namespaces that the operator should ignore. Ensure to add the namespaces that are required for the overall k8s operations like kube-system, cert-manager, istio-system etc.
-   `excludedDeployments`: A list of specific Deployments (by name and namespace) to exclude.
-   `maxParallelOperations`: The maximum number of scaling operations the operator can perform concurrently.
-   `checkInterval`: How often the operator should scan the cluster for scaling activities.
-   `forceScaleDown`: (***Use this with caution in Prod cluster***) - A boolean to enable forceful scaling in namespaces even without a `FinOpsScalePolicy`. Default is set to False
-   `forceScaleDownSchedule`: A schedule to use for forceful scaling (if enabled).
    -   `days`: The days of the week to apply the schedule (e.g., `["Mon", "Tue", "Wed", "Thu", "Fri"]` or `["*"]` for all days).
    -   `startTime`: The start time of the scaling window (e.g., `"18:00"`).
    -   `endTime`: The end time of the scaling window.
-   `forceScaleDownTimezone`: The timezone for the forceful scaling schedule.

### FinOpsScalePolicy

The `FinOpsScalePolicy` Custom Resource empowers users to define and enforce automated scaling policies for Deployments within their respective namespaces. This resource provides fine-grained control over when and how Deployments are scaled, enabling namespace administrators and application owners to optimize resource consumption according to their specific needs.


``` yaml
apiVersion: finops.devopsideas.com/v1alpha1
kind: FinOpsScalePolicy
metadata:
  name: my-policy
  namespace: my-app-namespace
spec:
  timezone: "UTC"
  defaultSchedule:
    days: ["Mon", "Tue", "Wed", "Thu", "Fri"]
    startTime: "20:00"
    endTime: "06:00"
  deployments:
    - name: my-deployment
      minReplicas: 1
      schedule:
        days: ["*"]
        startTime: "22:00"
        endTime: "04:00"
      optOut: false
    - name: another-deployment
      minReplicas: 0
      optOut: true

```

**Policy Options:**

-   `timezone`: The timezone for the schedules in this policy.
-   `defaultSchedule`: A default schedule that applies to all Deployments in the namespace (unless overridden by deployment specific schedule).
    -   `days`: The days of the week for the schedule.
    -   `startTime`: The start time of the scaling window.
    -   `endTime`: The end time of the scaling window.
-   `deployments`: A list of Deployment-specific scaling configurations.
    -   `name`: The name of the Deployment.
    -   `minReplicas`: The minimum number of replicas to scale the Deployment down to.
    -   `schedule`: An optional schedule that overrides the `defaultSchedule` for this Deployment.
    -   `optOut`: A boolean to prevent scaling for this Deployment.
-   `optOut`: A boolean at the policy level to opt-out all deployments in the namespace.

## How It Works

The operator continuously monitors the Kubernetes API for `FinOpsOperatorConfig` and `FinOpsScalePolicy` resources.

-   When a `FinOpsOperatorConfig` is created or updated, the operator reloads its configuration.
-   The operator periodically scans the cluster (based on *checkInterval*) and checks if any Deployments need to be scaled based on the defined schedules.
-   The operator uses annotations to store the original replica count of Deployments before scaling them down, allowing them to be scaled back up correctly.



## License

Copyright 2025 Christopher Sam K.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.


