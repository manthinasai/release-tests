PIPELINES-33
# Verify Ecosystem E2E spec

Pre condition:
  * Validate Operator should be installed

## S2I golang pipelinerun: PIPELINES-33-TC03
Tags: e2e, ecosystem, non-admin, s2i
Component: Pipelines
Level: Integration
Type: Functional
Importance: Critical

Steps:
  * Verify ServiceAccount "pipeline" exist
  * Create
      |S.NO|resource_dir                                          |
      |----|------------------------------------------------------|
      |1   |testdata/ecosystem/pipelines/s2i-go.yaml              |
      |2   |testdata/pvc/pvc.yaml                                 |
  * Get tags of the imagestream "golang" from namespace "openshift" and store to variable "golang-tags"
  * Start and verify pipeline "s2i-go-pipeline" with param "VERSION" with values stored in variable "golang-tags" with workspace "name=source,claimName=shared-pvc"
