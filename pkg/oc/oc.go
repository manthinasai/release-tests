package oc

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/getgauge-contrib/gauge-go/testsuit"
	"github.com/openshift-pipelines/release-tests/pkg/cmd"
	"github.com/openshift-pipelines/release-tests/pkg/config"
	"github.com/openshift-pipelines/release-tests/pkg/store"
)

// Create resources using oc command
func Create(path_dir, namespace string) {
	log.Printf("output: %s\n", cmd.MustSucceed("oc", "create", "-f", config.Path(path_dir), "-n", namespace).Stdout())
}

// Create resources using remote path using oc command
func CreateRemote(remote_path, namespace string) {
	log.Printf("output: %s\n", cmd.MustSucceed("oc", "create", "-f", remote_path, "-n", namespace).Stdout())
}

func Apply(path_dir, namespace string) {
	log.Printf("output: %s\n", cmd.MustSucceed("oc", "apply", "-f", config.Path(path_dir), "-n", namespace).Stdout())
}

// Delete resources using oc command
func Delete(path_dir, namespace string) {
	// Tekton Results sets a finalizer that prevent resource removal for some time
	// see parameters "store_deadline" and "forward_buffer"
	// by default, it waits at least 150 seconds
	log.Printf("output: %s\n", cmd.MustSuccedIncreasedTimeout(time.Second*300, "oc", "delete", "-f", config.Path(path_dir), "-n", namespace).Stdout())
}

// CreateNewProject Helps you to create new project
func CreateNewProject(ns string) {
	log.Printf("output: %s\n", cmd.MustSucceed("oc", "new-project", ns).Stdout())
}

// DeleteProject Helps you to delete new project
func DeleteProject(ns string) {
	log.Printf("output: %s\n", cmd.MustSucceed("oc", "delete", "project", ns).Stdout())
}

func DeleteProjectIgnoreErors(ns string) {
	log.Printf("output: %s\n", cmd.Run("oc", "delete", "project", ns).Stdout())
}

func LinkSecretToSA(secretname, sa, namespace string) {
	log.Printf("output: %s\n", cmd.MustSucceed("oc", "secret", "link", "serviceaccount/"+sa, "secrets/"+secretname, "-n", namespace).Stdout())
}

func CreateSecretWithSecretToken(secretname, namespace string) {
	log.Printf("output: %s\n", cmd.MustSucceed("oc", "create", "secret", "generic", secretname, "--from-literal=secretToken="+config.TriggersSecretToken, "-n", namespace).Stdout())
}

func EnableTLSConfigForEventlisteners(namespace string) {
	log.Printf("output: %s\n", cmd.MustSucceed("oc", "label", "namespace", namespace, "operator.tekton.dev/enable-annotation=enabled").Stdout())
}

func VerifyKubernetesEventsForEventListener(namespace string) {
	result := cmd.Run("oc", "-n", namespace, "get", "events")
	startedEvent := strings.Contains(result.String(), "dev.tekton.event.triggers.started.v1")
	successfulEvent := strings.Contains(result.String(), "dev.tekton.event.triggers.successful.v1")
	doneEvent := strings.Contains(result.String(), "dev.tekton.event.triggers.done.v1")
	if !startedEvent || !successfulEvent || !doneEvent {
		testsuit.T.Errorf("No events for successful, done and started")
	}
}

func UpdateTektonConfig(patch_data string) {
	log.Printf("output: %s\n", cmd.MustSucceed("oc", "patch", "tektonconfig", "config", "-p", patch_data, "--type=merge").Stdout())
}

func UpdateTektonConfigwithInvalidData(patch_data, errorMessage string) {
	result := cmd.Run("oc", "patch", "tektonconfig", "config", "-p", patch_data, "--type=merge")
	log.Printf("Output: %s\n", result.Stdout())
	if result.ExitCode != 1 {
		testsuit.T.Errorf("Expected exit code 1 but got %v", result.ExitCode)
	}
	if !strings.Contains(result.Stderr(), errorMessage) {
		testsuit.T.Errorf("Expected error message substring %v in %v", errorMessage, result.Stderr())
	}
}

func AnnotateNamespace(namespace, annotation string) {
	log.Printf("output: %s\n", cmd.MustSucceed("oc", "annotate", "namespace", namespace, annotation).Stdout())
}

func AnnotateNamespaceIgnoreErrors(namespace, annotation string) {
	log.Printf("output: %s\n", cmd.Run("oc", "annotate", "namespace", namespace, annotation).Stdout())
}

func RemovePrunerConfig() {
	cmd.Run("oc", "patch", "tektonconfig", "config", "-p", "[{ \"op\": \"remove\", \"path\": \"/spec/pruner\" }]", "--type=json")
}

func LabelNamespace(namespace, label string) {
	log.Printf("output: %s\n", cmd.MustSucceed("oc", "label", "namespace", namespace, label).Stdout())
}

func DeleteResource(resourceType, name string) {
	// Tekton Results sets a finalizer that prevent resource removal for some time
	// see parameters "store_deadline" and "forward_buffer"
	// by default, it waits at least 150 seconds
	log.Printf("output: %s\n", cmd.MustSuccedIncreasedTimeout(time.Second*300, "oc", "delete", resourceType, name, "-n", store.Namespace()).Stdout())
}

func DeleteResourceInNamespace(resourceType, name, namespace string) {
	log.Printf("output: %s\n", cmd.MustSucceed("oc", "delete", resourceType, name, "-n", namespace).Stdout())
}

func CheckProjectExists(projectName string) bool {
	commandResult := cmd.Run("oc", "project", projectName)
	return commandResult.ExitCode == 0 && !strings.Contains(commandResult.String(), "error")
}

func SecretExists(secretName string, namespace string) bool {
	return !strings.Contains(cmd.Run("oc", "get", "secret", secretName, "-n", namespace).String(), "Error")
}

func CreateSecretForGitResolver(secretData string) {
	cmd.MustSucceed("oc", "create", "secret", "generic", "github-auth-secret", "--from-literal", "github-auth-key="+secretData, "-n", "openshift-pipelines")
}

func CreateSecretInNamespace(secretData, secretName, namespace string) {
	cmd.MustSucceed("oc", "create", "secret", "generic", secretName, "--from-literal", "private-repo-token="+secretData, "-n", namespace)
}

func CreateSecretForWebhook(tokenSecretData, webhookSecretData, namespace string) {
	cmd.MustSucceed("oc", "create", "secret", "generic", "gitlab-webhook-config", "--from-literal", "provider.token="+tokenSecretData, "--from-literal", "webhook.secret="+webhookSecretData, "-n", namespace)
}

func EnableConsolePlugin() {
	json_output := cmd.MustSucceed("oc", "get", "consoles.operator.openshift.io", "cluster", "-o", "jsonpath={.spec.plugins}").Stdout()
	log.Printf("Already enabled console plugins: %s", json_output)
	var plugins []string

	if len(json_output) > 0 {
		err := json.Unmarshal([]byte(json_output), &plugins)

		if err != nil {
			testsuit.T.Errorf("Could not parse consoles.operator.openshift.io CR: %v", err)
		}

		if slices.Contains(plugins, config.ConsolePluginDeployment) {
			log.Printf("Pipelines console plugin is already enabled.")
			return
		}
	}

	plugins = append(plugins, config.ConsolePluginDeployment)

	patch_data := "{\"spec\":{\"plugins\":[\"" + strings.Join(plugins, "\",\"") + "\"]}}"
	cmd.MustSucceed("oc", "patch", "consoles.operator.openshift.io", "cluster", "-p", patch_data, "--type=merge").Stdout()
}

func GetSecretsData(secretName, namespace string) string {
	return cmd.MustSucceed("oc", "get", "secrets", secretName, "-n", namespace, "-o", "jsonpath=\"{.data}\"").Stdout()
}

func CreateChainsImageRegistrySecret(dockerConfig string) {
	cmd.MustSucceed("oc", "create", "secret", "generic", "chains-image-registry-credentials", "--from-literal=.dockerconfigjson="+dockerConfig, "--from-literal=config.json="+dockerConfig, "--type=kubernetes.io/dockerconfigjson")
}

func CopySecret(secretName string, sourceNamespace string, destNamespace string) {
	secretJson := cmd.MustSucceed("oc", "get", "secret", secretName, "-n", sourceNamespace, "-o", "json").Stdout()
	cmdOutput := cmd.MustSucceed("bash", "-c", fmt.Sprintf(`echo '%s' | jq 'del(.metadata["namespace", "creationTimestamp", "resourceVersion", "selfLink", "uid", "annotations"]) | .data |= with_entries(if .key == "github-auth-key" then .key = "token" else . end)'`, secretJson)).Stdout()
	cmd.MustSucceed("bash", "-c", fmt.Sprintf(`echo '%s' | kubectl apply -n %s -f -`, cmdOutput, destNamespace))
	log.Printf("Successfully copied secret %s from %s to %s", secretName, sourceNamespace, destNamespace)
}

// OLM SkipRange Validation

// PackageManifestList represents the list of package manifests
type PackageManifestList struct {
	Items []PackageManifest `json:"items"`
}

// PackageManifest represents a package manifest from the operator catalog
type PackageManifest struct {
	Metadata struct {
		Name string `json:"name"`
	} `json:"metadata"`
	Status struct {
		Channels []Channel `json:"channels"`
	} `json:"status"`
}

// Channel represents a channel in the package manifest
type Channel struct {
	Name           string `json:"name"`
	CurrentCSVDesc struct {
		Annotations map[string]string `json:"annotations"`
	} `json:"currentCSVDesc"`
}

// resolveOlmCatalog returns the catalog to query based on environment variables.
// OPERATOR_ENVIRONMENT takes priority (stage → custom-operators, prod → redhat-operators),
// then CATALOG_SOURCE, then auto-detect by trying both catalogs.
func resolveOlmCatalog() string {
	operatorEnv := os.Getenv("OPERATOR_ENVIRONMENT")
	catalogSource := os.Getenv("CATALOG_SOURCE")
	log.Printf("OLM catalog resolution: OPERATOR_ENVIRONMENT=%q CATALOG_SOURCE=%q", operatorEnv, catalogSource)

	switch operatorEnv {
	case "pre-stage", "stage":
		log.Printf("OLM catalog selected: custom-operators (from OPERATOR_ENVIRONMENT=%q)", operatorEnv)
		return "custom-operators"
	case "prod":
		log.Printf("OLM catalog selected: redhat-operators (from OPERATOR_ENVIRONMENT=%q)", operatorEnv)
		return "redhat-operators"
	}
	if catalogSource != "" {
		log.Printf("OLM catalog selected: %q (from CATALOG_SOURCE)", catalogSource)
		return catalogSource
	}
	log.Printf("OLM catalog: no env var set, will auto-detect")
	return "" // empty → auto-detect
}

// FetchOlmSkipRange fetches OLM skipRange data from the package manifest.
// Returns a map of channel name → skipRange string.
func FetchOlmSkipRange() (map[string]string, error) {
	// Determine which catalog(s) to query
	fixedCatalog := resolveOlmCatalog()
	catalogs := []string{"custom-operators", "redhat-operators"} // auto-detect order
	if fixedCatalog != "" {
		catalogs = []string{fixedCatalog} // explicit env config → no fallback
	}

	var packageManifestJSON string
	var selectedCatalog string

	for _, catalog := range catalogs {
		result := cmd.Run("oc", "get", "packagemanifest", "-n", "openshift-marketplace",
			"--selector=catalog="+catalog, "-o", "json")
		if result.ExitCode == 0 && strings.Contains(result.Stdout(), "openshift-pipelines-operator-rh") {
			packageManifestJSON = result.Stdout()
			selectedCatalog = catalog
			log.Printf("OLM catalog found: %q has openshift-pipelines-operator-rh", selectedCatalog)
			break
		}
		log.Printf("OLM catalog %q: openshift-pipelines-operator-rh not found (exit=%d)", catalog, result.ExitCode)
	}

	if packageManifestJSON == "" {
		return nil, fmt.Errorf("package manifest 'openshift-pipelines-operator-rh' not found in any catalog")
	}

	var packageManifestList PackageManifestList
	if err := json.Unmarshal([]byte(packageManifestJSON), &packageManifestList); err != nil {
		return nil, fmt.Errorf("failed to unmarshal package manifest: %w", err)
	}

	// Find openshift-pipelines-operator-rh package
	var targetPackage *PackageManifest
	for i := range packageManifestList.Items {
		if packageManifestList.Items[i].Metadata.Name == "openshift-pipelines-operator-rh" {
			targetPackage = &packageManifestList.Items[i]
			break
		}
	}

	if targetPackage == nil {
		return nil, fmt.Errorf("package 'openshift-pipelines-operator-rh' not found in catalog '%s'", selectedCatalog)
	}

	// Build channel to skipRange mapping
	channelSkipRangeMap := make(map[string]string)
	for _, channel := range targetPackage.Status.Channels {
		skipRange, exists := channel.CurrentCSVDesc.Annotations["olm.skipRange"]
		if exists && skipRange != "" {
			channelSkipRangeMap[channel.Name] = skipRange
		}
	}

	if len(channelSkipRangeMap) == 0 {
		return nil, fmt.Errorf("no valid OLM Skip Ranges found in catalog '%s'", selectedCatalog)
	}

	return channelSkipRangeMap, nil
}

// skipRangeContainsVersion checks if a version is within or matches the skipRange bounds
// For new minor releases (e.g., 1.23.0 with skipRange ">=1.22.0 <1.23.0"),
// the upper bound <1.23.0 technically excludes 1.23.0, but matching the upper bound is valid
func skipRangeContainsVersion(skipRange, version string) bool {
	// First try string contains (works for patch versions within range)
	if strings.Contains(skipRange, version) {
		return true
	}

	// Check if version matches the upper bound (for new minor releases)
	// Extract upper bound from skipRange (format: ">=X.Y.Z <X.Y.Z")
	if strings.Contains(skipRange, "<") {
		parts := strings.Split(skipRange, "<")
		if len(parts) == 2 {
			upperBound := strings.TrimSpace(parts[1])
			if upperBound == version {
				return true
			}
		}
	}

	return false
}

// majorMinor returns the major.minor prefix of a version string (e.g. "1.23.0" → "1.23").
func majorMinor(version string) string {
	if idx := strings.LastIndex(version, "."); idx != -1 {
		return version[:idx]
	}
	return version
}

// ValidateOlmSkipRange validates that OSP_VERSION is covered by the OLM skipRange
// for the matching channel. Nightly builds (OSP_VERSION=5.0.5) are matched by
// skipRange alone without requiring a channel name match.
func ValidateOlmSkipRange() {
	skipRangeMap, err := FetchOlmSkipRange()
	if err != nil {
		testsuit.T.Fail(fmt.Errorf("failed to fetch OLM skip range: %w", err))
		return
	}

	ospVersion := os.Getenv("OSP_VERSION")
	if ospVersion == "" {
		testsuit.T.Fail(fmt.Errorf("OSP_VERSION environment variable not set"))
		return
	}

	isNightly := ospVersion == "5.0.5"
	ospMM := majorMinor(ospVersion)

	// Log available channels to aid debugging
	channelNames := make([]string, 0, len(skipRangeMap))
	for ch, sr := range skipRangeMap {
		channelNames = append(channelNames, ch)
		log.Printf("OLM channel=%q skipRange=%q", ch, sr)
	}

	// First pass: prefer versioned channel matching ospMM (e.g. "pipelines-1.23")
	for channel, skipRange := range skipRangeMap {
		if channel == "latest" {
			continue
		}
		if isNightly || strings.Contains(channel, ospMM) {
			if skipRangeContainsVersion(skipRange, ospVersion) {
				log.Printf("OLM validation passed: channel=%q skipRange=%q contains OSP_VERSION=%q", channel, skipRange, ospVersion)
				return
			}
		}
	}

	// Second pass: stage catalogs may only expose a "latest" channel — check it as fallback
	for channel, skipRange := range skipRangeMap {
		if skipRangeContainsVersion(skipRange, ospVersion) {
			log.Printf("OLM validation passed (fallback): channel=%q skipRange=%q contains OSP_VERSION=%q", channel, skipRange, ospVersion)
			return
		}
	}

	testsuit.T.Fail(fmt.Errorf("OSP_VERSION %q not found in skipRange for any channel (available: [%s])",
		ospVersion, strings.Join(channelNames, ", ")))
}

// OLM SkipRange Upgrade Path Validation

// SkipRangeData holds skipRange information for upgrade validation
type SkipRangeData struct {
	PreUpgrade  map[string]string `json:"pre-upgrade-olm-skip-range"`
	PostUpgrade map[string]string `json:"post-upgrade-olm-skip-range"`
}

// GetOlmSkipRange fetches OLM skipRange and stores it for upgrade validation
// upgradeType: "pre-upgrade" or "post-upgrade"
// fileName: path to JSON file where skipRange data will be stored
func GetOlmSkipRange(upgradeType, fileName string) {
	skipRangeMap, err := FetchOlmSkipRange()
	if err != nil {
		testsuit.T.Fail(fmt.Errorf("failed to fetch OLM Skip Range: %w", err))
		return
	}

	filePath := config.Path(fileName)

	// Read existing data if file exists
	existingData := &SkipRangeData{
		PreUpgrade:  make(map[string]string),
		PostUpgrade: make(map[string]string),
	}

	if fileData, err := os.ReadFile(filePath); err == nil {
		_ = json.Unmarshal(fileData, existingData)
	}

	// Store skipRange data based on upgrade type
	if upgradeType == "pre-upgrade" {
		existingData.PreUpgrade = skipRangeMap
	} else if upgradeType == "post-upgrade" {
		existingData.PostUpgrade = skipRangeMap
	}

	// Write data back to file
	jsonData, err := json.MarshalIndent(existingData, "", "  ")
	if err != nil {
		testsuit.T.Fail(fmt.Errorf("failed to marshal skipRange data: %w", err))
		return
	}

	if err := os.WriteFile(filePath, jsonData, 0644); err != nil {
		testsuit.T.Fail(fmt.Errorf("failed to write skipRange data to file: %w", err))
		return
	}
}

// ValidateOlmSkipRangeDiff validates skipRange changes between pre and post upgrade
// fileName: path to JSON file containing pre and post upgrade skipRange data
func ValidateOlmSkipRangeDiff(fileName string) {
	filePath := config.Path(fileName)

	// Read skipRange data from file
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		testsuit.T.Fail(fmt.Errorf("failed to read skipRange file %s: %w", fileName, err))
		return
	}

	var skipRangeData SkipRangeData
	if err := json.Unmarshal(fileData, &skipRangeData); err != nil {
		testsuit.T.Fail(fmt.Errorf("failed to unmarshal skipRange data: %w", err))
		return
	}

	if len(skipRangeData.PreUpgrade) == 0 || len(skipRangeData.PostUpgrade) == 0 {
		testsuit.T.Fail(fmt.Errorf("missing pre-upgrade or post-upgrade skipRange data"))
		return
	}

	ospVersion := os.Getenv("OSP_VERSION")
	if ospVersion == "" {
		testsuit.T.Fail(fmt.Errorf("OSP_VERSION environment variable not set"))
		return
	}

	ospMM := majorMinor(ospVersion)
	log.Printf("OLM skipRange diff validation: OSP_VERSION=%q", ospVersion)

	log.Printf("Pre-upgrade channels:")
	for ch, sr := range skipRangeData.PreUpgrade {
		log.Printf("  channel=%q skipRange=%q", ch, sr)
	}
	log.Printf("Post-upgrade channels:")
	for ch, sr := range skipRangeData.PostUpgrade {
		log.Printf("  channel=%q skipRange=%q", ch, sr)
	}

	// Find the channel matching the current version in post-upgrade data
	targetChannel := ""
	for channel := range skipRangeData.PostUpgrade {
		if channel != "latest" && strings.Contains(channel, ospMM) {
			targetChannel = channel
			break
		}
	}

	if targetChannel == "" {
		testsuit.T.Fail(fmt.Errorf("no channel found for OSP_VERSION %q (major.minor: %q)", ospVersion, ospMM))
		return
	}

	log.Printf("Target channel for OSP_VERSION %q: %q", ospVersion, targetChannel)

	// Get pre and post upgrade skipRange for the target channel
	preSkipRange, preExists := skipRangeData.PreUpgrade[targetChannel]
	postSkipRange, postExists := skipRangeData.PostUpgrade[targetChannel]

	switch {
	case preExists && !postExists:
		// Channel disappeared after upgrade — should never happen
		testsuit.T.Fail(fmt.Errorf("channel %q existed before upgrade but is missing after upgrade", targetChannel))

	case !preExists && postExists:
		// New channel added for a new minor release — validate the new skipRange covers our version
		if !skipRangeContainsVersion(postSkipRange, ospVersion) {
			testsuit.T.Fail(fmt.Errorf("new channel %q has skipRange %q which does not include OSP_VERSION %q", targetChannel, postSkipRange, ospVersion))
			return
		}
		log.Printf("Minor upgrade validated: new channel %q skipRange=%q contains OSP_VERSION=%q", targetChannel, postSkipRange, ospVersion)

	default:
		// Patch upgrade: channel exists in both — post-upgrade skipRange must contain the new version
		if !skipRangeContainsVersion(postSkipRange, ospVersion) {
			testsuit.T.Fail(fmt.Errorf("post-upgrade skipRange %q for channel %q does not include OSP_VERSION %q (was: %q)", postSkipRange, targetChannel, ospVersion, preSkipRange))
			return
		}
		log.Printf("Patch upgrade validated: channel %q skipRange %q → %q contains OSP_VERSION=%q", targetChannel, preSkipRange, postSkipRange, ospVersion)
	}
}

// ValidateChannelSkipRangeBounds verifies that every non-latest channel has a
// properly structured skipRange where:
//   - format matches ">=X.Y.Z <A.B.C"
//   - upper bound major.minor matches the channel version (e.g. pipelines-1.23 → <1.23.x)
//   - lower bound major.minor is one minor version below (e.g. pipelines-1.23 → >=1.22.x)
func ValidateChannelSkipRangeBounds() {
	skipRangeMap, err := FetchOlmSkipRange()
	if err != nil {
		testsuit.T.Fail(fmt.Errorf("failed to fetch OLM skip range: %w", err))
		return
	}

	rangePattern := regexp.MustCompile(`^>=(\d+\.\d+\.\d+)\s+<(\d+\.\d+\.\d+)$`)
	channelPattern := regexp.MustCompile(`^pipelines-(\d+)\.(\d+)$`)

	log.Printf("OLM channel skipRange bounds validation:")
	var errs []string
	for channel, skipRange := range skipRangeMap {
		if channel == "latest" {
			log.Printf("  channel=%q skipRange=%q [skipped]", channel, skipRange)
			continue
		}

		cm := channelPattern.FindStringSubmatch(channel)
		if cm == nil {
			errs = append(errs, fmt.Sprintf("channel %q does not match expected pattern 'pipelines-X.Y'", channel))
			continue
		}

		rm := rangePattern.FindStringSubmatch(skipRange)
		if rm == nil {
			errs = append(errs, fmt.Sprintf("channel %q has invalid skipRange format %q (expected '>=X.Y.Z <A.B.C')", channel, skipRange))
			continue
		}

		lowerBound, upperBound := rm[1], rm[2]
		channelMM := fmt.Sprintf("%s.%s", cm[1], cm[2]) // e.g. "1.23"

		upperOK := majorMinor(upperBound) == channelMM
		minor, _ := strconv.Atoi(cm[2])
		expectedLowerMM := fmt.Sprintf("%s.%d", cm[1], minor-1)
		lowerOK := majorMinor(lowerBound) == expectedLowerMM

		log.Printf("  channel=%q skipRange=%q lowerBound=%q(expected %q.x)=%v upperBound=%q(expected %q.x)=%v",
			channel, skipRange, lowerBound, expectedLowerMM, lowerOK, upperBound, channelMM, upperOK)

		if !upperOK {
			errs = append(errs, fmt.Sprintf("channel %q: upper bound %q major.minor should be %q", channel, upperBound, channelMM))
		}
		if !lowerOK {
			errs = append(errs, fmt.Sprintf("channel %q: lower bound %q major.minor should be %q", channel, lowerBound, expectedLowerMM))
		}
	}

	if len(errs) > 0 {
		testsuit.T.Fail(fmt.Errorf("skipRange bounds validation failed:\n%s", strings.Join(errs, "\n")))
		return
	}
	log.Printf("All channel skipRange bounds are valid")
}
