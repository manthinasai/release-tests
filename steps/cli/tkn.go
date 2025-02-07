package cli

import (
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"regexp"

	"github.com/getgauge-contrib/gauge-go/gauge"
	"github.com/openshift-pipelines/release-tests/pkg/pipelines"
	"github.com/openshift-pipelines/release-tests/pkg/store"
	"github.com/openshift-pipelines/release-tests/pkg/tkn"
)

var _ = gauge.Step("Start and verify pipeline <pipelineName> with param <paramName> with values stored in variable <variableName> with workspace <workspaceValue>", func(pipelineName, paramName, variableName, workspaceValue string) {
	values := store.GetScenarioDataSlice(variableName)
	params := make(map[string]string)
	workspaces := make(map[string]string)
	workspaces[strings.Split(workspaceValue, ",")[0]] = strings.Split(workspaceValue, ",")[1]
	// workspaceParts := strings.Split(workspaceValue, ",")
	// for _, part := range workspaceParts {
	// 	kv := strings.Split(part, "=")
	// 	if len(kv) == 2 {
	// 		workspaces[kv[0]] = kv[1]
	// 	} else {
	// 		log.Printf("Invalid format in workspace value: %s", part)
	// 	}
	// }
	// workspaceName := workspaces["name"]
	// claimName := workspaces["claimName"]
	var wg sync.WaitGroup // wait for all goroutine to finish
	for _, value := range values {
		if value == "latest" {
			continue
		}

		wg.Add(1)

		go func(value string) {
			defer wg.Done()
			log.Printf("Starting pipeline %s with param %s=%s", pipelineName, paramName, value)
			params[paramName] = value
			// templateFile := "testdata/ecosystem/pipelineruns/s2i-pipelinerun-template.yaml"
			CustomName := pipelineName + value
			pipelineRunName := tkn.StartPipeline(pipelineName, params, workspaces, store.Namespace(), "--use-param-defaults", "--name", CustomName)
			log.Printf("Pipeline run name is %v\n", pipelineRunName)
			// pipelines.StartPipelineRunWithTemplate(templateFile, pipelineName, params, workspaceName, claimName)
			// pipelines.ValidatePipelineRun(store.Clients(), pipelineName+"-run-"+value, "successful", "no", store.Namespace())
			pipelines.ValidatePipelineRun(store.Clients(), pipelineRunName, "successful", "no", store.Namespace())

		}(value)

		time.Sleep(3 * time.Second)
	}
	wg.Wait()
})

var _ = gauge.Step("Start and verify dotnet pipeline <pipelineName> with values stored in variable <variableName> with workspace <workspaceValue>", func(pipelineName, variableName, workspaceValue string) {
	values := store.GetScenarioDataSlice(variableName)
	params := make(map[string]string)
	workspaces := make(map[string]string)
	workspaces[strings.Split(workspaceValue, ",")[0]] = strings.Split(workspaceValue, ",")[1]
	workspaceParts := strings.Split(workspaceValue, ",")
	for _, part := range workspaceParts {
		kv := strings.Split(part, "=")
		if len(kv) == 2 {
			workspaces[kv[0]] = kv[1]
		} else {
			log.Printf("Invalid format in workspace value: %s", part)
		}
	}
	workspaceName := workspaces["name"]
	claimName := workspaces["claimName"]
	paramName := "VERSION"
	re := regexp.MustCompile(`\d+\.\d+`)
	var exampleRevision string
	var wg sync.WaitGroup // wait for all goroutine to finish
	for _, value := range values {
		if value == "latest" {
			continue
		}
		exampleRevision = re.FindString(value)
		params[paramName] = value
		versionInt, _ := strconv.ParseFloat(exampleRevision, 64)
		log.Printf("VersionInt is %v\n", versionInt)
		if versionInt >= 5.0 {
			params["EXAMPLE_REVISION"] = "dotnet-" + exampleRevision
		} else {
			params["EXAMPLE_REVISION"] = "dotnetcore-" + exampleRevision
		}
		log.Printf("params['EXAMPLE_REVISION'] is %v\n", params["EXAMPLE_REVISION"])
		log.Printf("Starting pipeline %s with param %s=%s and EXAMPLE_REVISION=%s", pipelineName, paramName, value, params["EXAMPLE_REVISION"])

		wg.Add(1)

		go func(pipelineName string, params map[string]string) {
			defer wg.Done()
			log.Printf("Params are %v\n", params)
			templateFile := "testdata/ecosystem/pipelineruns/s2i-pipelinerun-template.yaml"
			pipelines.StartPipelineRunWithTemplate(templateFile, pipelineName, params, workspaceName, claimName)
			// pipelineRunName := tkn.StartPipeline(pipelineName, params, workspaces, store.Namespace(), "--use-param-defaults")
			pipelines.ValidatePipelineRun(store.Clients(), pipelineName+"-run-"+value, "successful", "no", store.Namespace())
		}(pipelineName, params)

		time.Sleep(3 * time.Second)
	}
	wg.Wait()
})
