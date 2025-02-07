package pipelines

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/getgauge-contrib/gauge-go/testsuit"
	"github.com/openshift-pipelines/release-tests/pkg/clients"
	resource "github.com/openshift-pipelines/release-tests/pkg/config"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type PipelineRunTemplate struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name string `yaml:"name"`
	} `yaml:"metadata"`
	Spec struct {
		PipelineRef struct {
			Name string `yaml:"name"`
		} `yaml:"pipelineRef"`
		Params []struct {
			Name  string `yaml:"name"`
			Value string `yaml:"value"`
		} `yaml:"params"`
		Workspaces []struct {
			Name                  string `yaml:"name"`
			PersistentVolumeClaim struct {
				ClaimName string `yaml:"claimName"`
			} `yaml:"persistentVolumeClaim"`
		} `yaml:"workspaces"`
	} `yaml:"spec"`
}

// checkLabelPropagation checks that labels are correctly propagating from
// Pipelines, PipelineRuns, and Tasks to TaskRuns and Pods.
func checkLabelPropagation(c *clients.Clients, namespace string, pipelineRunName string, tr *v1.TaskRun) {
	// Our controllers add 4 labels automatically. If custom labels are set on
	// the Pipeline, PipelineRun, or Task then the map will have to be resized.
	labels := make(map[string]string, 4)

	// Check label propagation to PipelineRuns.
	pr, err := c.PipelineRunClient.Get(c.Ctx, pipelineRunName, metav1.GetOptions{})
	if err != nil {
		testsuit.T.Errorf("failed to get pipeline run for task run %s \n %v", tr.Name, err)
	}

	p, err := c.PipelineClient.Get(c.Ctx, pr.Spec.PipelineRef.Name, metav1.GetOptions{})
	if err != nil {
		testsuit.T.Errorf("failed to get pipeline for pipeline run %s \n %v", pr.Name, err)
	}

	// By default, controller doesn't add any labels to Pipelines
	for key, val := range p.ObjectMeta.Labels {
		labels[key] = val
	}

	// This label is added to every PipelineRun by the PipelineRun controller
	labels[pipeline.PipelineLabelKey] = p.Name
	AssertLabelsMatch(labels, pr.ObjectMeta.Labels)

	// Check label propagation to TaskRuns.
	for key, val := range pr.ObjectMeta.Labels {
		labels[key] = val
	}
	// This label is added to every TaskRun by the PipelineRun controller
	labels[pipeline.PipelineRunLabelKey] = pr.Name
	if tr.Spec.TaskRef != nil {
		task, err := c.TaskClient.Get(c.Ctx, tr.Spec.TaskRef.Name, metav1.GetOptions{})
		if err != nil {
			testsuit.T.Errorf("failed to get task for task run %s \n %v", tr.Name, err)
		}

		// By default, controller doesn't add any labels to Tasks
		for key, val := range task.ObjectMeta.Labels {
			labels[key] = val
		}
		// This label is added to TaskRuns that reference a Task by the TaskRun controller
		labels[pipeline.TaskLabelKey] = task.Name
	}
	AssertLabelsMatch(labels, tr.ObjectMeta.Labels)

	// PodName is "" if a retry happened and pod is deleted
	// This label is added to every Pod by the TaskRun controller
	if tr.Status.PodName != "" {
		// Check label propagation to Pods.
		pod := GetPodForTaskRun(c, namespace, tr)
		// This label is added to every Pod by the TaskRun controller
		labels[pipeline.TaskRunLabelKey] = tr.Name
		AssertLabelsMatch(labels, pod.ObjectMeta.Labels)
	}
}

// checkAnnotationPropagation checks that annotations are correctly propagating from
// Pipelines, PipelineRuns, and Tasks to TaskRuns and Pods.
func checkAnnotationPropagation(c *clients.Clients, namespace string, pipelineRunName string, tr *v1.TaskRun) {
	annotations := make(map[string]string)

	// Check annotation propagation to PipelineRuns.
	pr, err := c.PipelineRunClient.Get(c.Ctx, pipelineRunName, metav1.GetOptions{})
	if err != nil {
		testsuit.T.Errorf("failed to get pipeline run for task run %s \n %v", tr.Name, err)
	}

	p, err := c.PipelineClient.Get(c.Ctx, pr.Spec.PipelineRef.Name, metav1.GetOptions{})
	if err != nil {
		testsuit.T.Errorf("failed to get pipeline for pipeline run %s \n %v", pr.Name, err)
	}

	for key, val := range p.ObjectMeta.Annotations {
		annotations[key] = val
	}
	AssertAnnotationsMatch(annotations, pr.ObjectMeta.Annotations)

	// Check annotation propagation to TaskRuns.
	for key, val := range pr.ObjectMeta.Annotations {
		// Annotations created by Chains are created after task runs finish
		if !strings.HasPrefix(key, "chains.tekton.dev") && !strings.HasPrefix(key, "results.tekton.dev") {
			annotations[key] = val
		}
	}
	if tr.Spec.TaskRef != nil {
		task, err := c.TaskClient.Get(c.Ctx, tr.Spec.TaskRef.Name, metav1.GetOptions{})
		if err != nil {
			testsuit.T.Errorf("failed to get task for task run %s \n %v", tr.Name, err)
		}
		for key, val := range task.ObjectMeta.Annotations {
			annotations[key] = val
		}
	}
	AssertAnnotationsMatch(annotations, tr.ObjectMeta.Annotations)

	// Check annotation propagation to Pods.
	pod := GetPodForTaskRun(c, namespace, tr)
	AssertAnnotationsMatch(annotations, pod.ObjectMeta.Annotations)
}

func GetPodForTaskRun(c *clients.Clients, namespace string, tr *v1.TaskRun) *corev1.Pod {
	// The Pod name has a random suffix, so we filter by label to find the one we care about.
	pods, err := c.KubeClient.Kube.CoreV1().Pods(namespace).List(c.Ctx, metav1.ListOptions{
		LabelSelector: pipeline.TaskRunLabelKey + " = " + tr.Name,
	})
	if err != nil {
		testsuit.T.Errorf("failed to get pod for task run %s \n %v", tr.Name, err)
	}

	if numPods := len(pods.Items); numPods != 1 {
		testsuit.T.Errorf("Expected 1 pod for task run %s, but got %d pods", tr.Name, numPods)
	}
	return &pods.Items[0]
}

func AssertLabelsMatch(expectedLabels, actualLabels map[string]string) {
	for key, expectedVal := range expectedLabels {
		if actualVal := actualLabels[key]; actualVal != expectedVal {
			testsuit.T.Errorf("Expected labels containing %s=%s but labels were %v", key, expectedVal, actualLabels)
		}
	}
}

func AssertAnnotationsMatch(expectedAnnotations, actualAnnotations map[string]string) {
	for key, expectedVal := range expectedAnnotations {
		if actualVal := actualAnnotations[key]; actualVal != expectedVal {
			testsuit.T.Errorf("Expected annotations containing %s=%s but annotations were %v", key, expectedVal, actualAnnotations)
		}
	}
}

func Cast2pipelinerun(obj runtime.Object) (*v1.PipelineRun, error) {
	var run *v1.PipelineRun
	unstruct, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return nil, err
	}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstruct, &run); err != nil {
		return nil, err
	}
	return run, nil
}

func createKeyValuePairs(m map[string]string) string {
	b := new(bytes.Buffer)
	for key, value := range m {
		fmt.Fprintf(b, "%s = %s\n", key, value)
	}
	return b.String()
}

// func ReadandUpdateYamlFile(filepath string, pipelineRef string, params map[string]string, workspacename string, claimName string) ([]byte, error) {
// 	prYaml, err := os.ReadFile(resource.Path(filepath))
// 	if err != nil {
// 		return nil, fmt.Errorf("error reading YAML file: %w", err)
// 	}

// 	var pipelineRun PipelineRunTemplate
// 	err = yaml.Unmarshal(prYaml, &pipelineRun)
// 	if err != nil {
// 		return nil, fmt.Errorf("error unmarshalling YAML: %w", err)
// 	}

// 	pipelineRun.Metadata.Name = fmt.Sprintf("%s-run-%s", pipelineRef, params["VERSION"])
// 	pipelineRun.Spec.PipelineRef.Name = pipelineRef
// 	for i, param := range pipelineRun.Spec.Params {
// 		if newValue, exists := params[param.Name]; exists {
// 			pipelineRun.Spec.Params[i].Value = newValue
// 		}
// 	}
// 	pipelineRun.Spec.Workspaces[0].Name = workspacename
// 	pipelineRun.Spec.Workspaces[0].PersistentVolumeClaim.ClaimName = claimName

// 	updatedYaml, err := yaml.Marshal(&pipelineRun)
// 	if err != nil {
// 		return nil, fmt.Errorf("error marshalling YAML: %w", err)
// 	}

// 	return updatedYaml, nil
// }

func ReadandUpdateYamlFile(filepath string, pipelineRef string, params map[string]string, workspacename string, claimName string) ([]byte, error) {
	prYaml, err := os.ReadFile(resource.Path(filepath))
	if err != nil {
		return nil, fmt.Errorf("error reading YAML file: %w", err)
	}

	var pipelineRun PipelineRunTemplate
	err = yaml.Unmarshal(prYaml, &pipelineRun)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling YAML: %w", err)
	}

	// Set metadata and pipeline ref
	pipelineRun.Metadata.Name = fmt.Sprintf("%s-run-%s", pipelineRef, params["VERSION"])
	pipelineRun.Spec.PipelineRef.Name = pipelineRef

	// Update existing params or add new ones if they don't exist
	for paramName, paramValue := range params {
		// Try to update an existing param
		updated := false
		for i, param := range pipelineRun.Spec.Params {
			if param.Name == paramName {
				pipelineRun.Spec.Params[i].Value = paramValue
				updated = true
				break
			}
		}
		// If the param doesn't exist, add it
		if !updated {
			pipelineRun.Spec.Params = append(pipelineRun.Spec.Params, struct {
				Name  string `yaml:"name"`
				Value string `yaml:"value"`
			}{
				Name:  paramName,
				Value: paramValue,
			})
		}
	}

	// Update workspaces
	pipelineRun.Spec.Workspaces[0].Name = workspacename
	pipelineRun.Spec.Workspaces[0].PersistentVolumeClaim.ClaimName = claimName

	// Marshal the updated struct back to YAML
	updatedYaml, err := yaml.Marshal(&pipelineRun)
	if err != nil {
		return nil, fmt.Errorf("error marshalling YAML: %w", err)
	}
	log.Printf("Final params: %+v", pipelineRun.Spec.Params)

	return updatedYaml, nil
}
