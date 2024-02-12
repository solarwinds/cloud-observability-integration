/* Copyright 2022 SolarWinds Worldwide, LLC. All rights reserved.
*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at:
*
*	http://www.apache.org/licenses/LICENSE-2.0
*
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and limitations
* under the License.
 */

package main

import (
	"regexp"

	"go.opentelemetry.io/collector/model/pdata"
	semconv "go.opentelemetry.io/collector/model/semconv/v1.5.0"
)
var (
    detectHostIdRegExp = regexp.MustCompile(`^(?P<HostId>(i-|ip-)[\w\-]+)`)
    detectRegionRegExp = regexp.MustCompile(`(?P<Region>\w{2}-\w+-\d+)`)
)
type OtlpRequestBuilder interface {
    SetHostId(hostId string) (OtlpRequestBuilder)
    SetCloudAccount(account string) (OtlpRequestBuilder)
    SetLogGroup(logGroup string) (OtlpRequestBuilder)
    SetLogStream(logStream string) (OtlpRequestBuilder)
    AddLogEntry(entryId string, timestamp int64, message, region string, attributes ...map[string]interface{}) (OtlpRequestBuilder)
    MatchHostId(hostId string) (bool)
    HasHostId() (bool)
    GetLogs() pdata.Logs
    HasContainerName() (bool)
    MatchContainerName(clusterUid string, namespaceName string, podName string, containerName string) (bool)
    SetKubernetesPodName(podName string) (OtlpRequestBuilder)
    SetKubernetesNamespaceName(namespaceName string) (OtlpRequestBuilder)
    SetKubernetesClusterUid(clusterUid string) (OtlpRequestBuilder)
    SetKubernetesContainerName(containerName string) (OtlpRequestBuilder)
    SetKubernetesContainerImage(containerImage string) (OtlpRequestBuilder)
    SetKubernetesPodUID(podUID string) (OtlpRequestBuilder)
    SetKubernetesContainerId(containerId string) (OtlpRequestBuilder)
    SetKubernetesNodeName(nodeName string) (OtlpRequestBuilder)
    SetKubernetesPodLabels(podLabels map[string]string) (OtlpRequestBuilder)
    SetKubernetesPodAnnotations(podAnnotations map[string]string) (OtlpRequestBuilder)
    SetKubernetesManifestVersion(manifestVersion string, defaultVersion string) (OtlpRequestBuilder)
    SetOtelAttributes(podName string, containerName string) (OtlpRequestBuilder)
}

type otlpRequestBuilder struct {
    logs pdata.Logs
    resLogs pdata.ResourceLogs
    instrLogsSlice pdata.InstrumentationLibraryLogsSlice
    instrLogs pdata.InstrumentationLibraryLogs
    hostId string
    parsedRegion string
    parsedHostId string
}

func NewOtlpRequestBuilder() (builder OtlpRequestBuilder){
    logs := pdata.NewLogs()
    resLogs := logs.ResourceLogs().AppendEmpty()
    resLogs.SetSchemaUrl(semconv.SchemaURL)
    instrLogsSlice := resLogs.InstrumentationLibraryLogs()
    builder = &otlpRequestBuilder{ logs :  logs, resLogs: resLogs, instrLogsSlice: instrLogsSlice}
    return
}

func (rb * otlpRequestBuilder) SetHostId(hostId string) (builder OtlpRequestBuilder) {
    rb.hostId = hostId

    attrs := rb.resLogs.Resource().Attributes()
    if rb.hostId != "" {
        attrs.UpsertString(semconv.AttributeHostID, rb.hostId)
        attrs.UpsertString(semconv.AttributeCloudPlatform, semconv.AttributeCloudPlatformAWSEC2)
    } else {
        attrs.Delete(semconv.AttributeHostID)
        attrs.Delete(semconv.AttributeCloudPlatform)
    }
    builder = rb
    return
}

func (rb * otlpRequestBuilder) SetCloudAccount(account string) (builder OtlpRequestBuilder) {
    attrs := rb.resLogs.Resource().Attributes()
    attrs.UpsertString(semconv.AttributeCloudAccountID, account)
    builder = rb
    return
}

func (rb * otlpRequestBuilder) SetLogGroup(logGroup string) (builder OtlpRequestBuilder) {
    attrs := rb.resLogs.Resource().Attributes()
    attrs.UpsertString(semconv.AttributeAWSLogGroupNames, logGroup)
    builder = rb
    return
}

func (rb * otlpRequestBuilder) MatchContainerName(clusterUid string, namespaceName string, podName string, containerName string) (bool) {
    attrs := rb.resLogs.Resource().Attributes()

    attrsContainerName, containerNameExists := attrs.Get(semconv.AttributeK8SContainerName)
    attrsPodName, podNameExists := attrs.Get(semconv.AttributeK8SPodUID)
    attrsNamespaceName, namespaceNameExists := attrs.Get(semconv.AttributeK8SNamespaceName)
    attrsClusterUid, clusterUidExists := attrs.Get("sw.k8s.cluster.uid")

    if !podNameExists || !namespaceNameExists || !clusterUidExists || !containerNameExists {
        return false
    }

    return attrsContainerName.StringVal() == containerName &&
        attrsPodName.StringVal() == podName &&
        attrsNamespaceName.StringVal() == namespaceName &&
        attrsClusterUid.StringVal() == clusterUid
}

func (rb * otlpRequestBuilder) HasContainerName() (bool) {
    attrs := rb.resLogs.Resource().Attributes()

    _, containerNameExists := attrs.Get(semconv.AttributeK8SContainerName)
    _, podUidExists := attrs.Get(semconv.AttributeK8SPodUID)
    _, namespaceNameExists := attrs.Get(semconv.AttributeK8SNamespaceName)
    _, clusterUidExists := attrs.Get("sw.k8s.cluster.uid")

    if podUidExists && namespaceNameExists && clusterUidExists && containerNameExists {
        return true
    }

    return false
}

func (rb * otlpRequestBuilder) SetKubernetesPodName(podName string) (builder OtlpRequestBuilder) {
    attrs := rb.resLogs.Resource().Attributes()
    attrs.UpsertString(semconv.AttributeK8SPodName, podName)
    builder = rb
    return
}

func (rb * otlpRequestBuilder) SetKubernetesNamespaceName(namespaceName string) (builder OtlpRequestBuilder) {
    attrs := rb.resLogs.Resource().Attributes()
    attrs.UpsertString(semconv.AttributeK8SNamespaceName, namespaceName)
    builder = rb
    return
}

func (rb * otlpRequestBuilder) SetKubernetesClusterUid(clusterUid string) (builder OtlpRequestBuilder) {
    attrs := rb.resLogs.Resource().Attributes()
    attrs.UpsertString("sw.k8s.cluster.uid", clusterUid)
    builder = rb
    return
}

func (rb * otlpRequestBuilder) SetKubernetesContainerName(containerName string) (builder OtlpRequestBuilder) {
    attrs := rb.resLogs.Resource().Attributes()
    attrs.UpsertString(semconv.AttributeK8SContainerName, containerName)
    builder = rb
    return
}

func (rb * otlpRequestBuilder) SetKubernetesContainerImage(containerImage string) (builder OtlpRequestBuilder) {
    attrs := rb.resLogs.Resource().Attributes()
    attrs.UpsertString("k8s.container.image.name", containerImage)
    builder = rb
    return
}

func (rb * otlpRequestBuilder) SetKubernetesPodUID(podUID string) (builder OtlpRequestBuilder) {
    attrs := rb.resLogs.Resource().Attributes()
    attrs.UpsertString(semconv.AttributeK8SPodUID, podUID)
    builder = rb
    return
}

func (rb * otlpRequestBuilder) SetKubernetesContainerId(containerId string) (builder OtlpRequestBuilder) {
    attrs := rb.resLogs.Resource().Attributes()
    attrs.UpsertString(semconv.AttributeContainerID, containerId)
    builder = rb
    return
}

func (rb * otlpRequestBuilder) SetKubernetesNodeName(nodeName string) (builder OtlpRequestBuilder) {
    attrs := rb.resLogs.Resource().Attributes()
    attrs.UpsertString(semconv.AttributeK8SNodeName, nodeName)
    builder = rb
    return
}

func (rb * otlpRequestBuilder) SetKubernetesPodLabels(podLabels map[string]string) (builder OtlpRequestBuilder) {
    attrs := rb.resLogs.Resource().Attributes()
    for key, value := range podLabels {
        attrs.UpsertString("k8s.pod.labels."+key, value)
    }
    builder = rb
    return
}

func (rb * otlpRequestBuilder) SetKubernetesPodAnnotations(podAnnotations map[string]string) (builder OtlpRequestBuilder) {
    attrs := rb.resLogs.Resource().Attributes()
    for key, value := range podAnnotations {
        attrs.UpsertString("k8s.pod.annotations."+key, value)
    }
    builder = rb
    return
}

func (rb * otlpRequestBuilder) SetKubernetesManifestVersion(manifestVersion string, defaultVersion string) (builder OtlpRequestBuilder) {
    attrs := rb.resLogs.Resource().Attributes()
    versionToSet := manifestVersion
    if versionToSet == "" {
        versionToSet = defaultVersion
    }

    attrs.UpsertString("sw.k8s.agent.manifest.version", versionToSet)
    builder = rb
    return
}

func (rb * otlpRequestBuilder) SetOtelAttributes(podName string, containerName string) (builder OtlpRequestBuilder) {
    attrs := rb.resLogs.Resource().Attributes()
    attrs.UpsertString("host.name", podName)
    attrs.UpsertString("service.name", containerName)
    builder = rb
    return
}

func (rb * otlpRequestBuilder) SetLogStream(logStream string) (builder OtlpRequestBuilder) {
    attrs := rb.resLogs.Resource().Attributes()
    attrs.InsertString(semconv.AttributeAWSLogStreamNames, logStream)
    matches := detectHostIdRegExp.FindStringSubmatch(logStream)
    matchIndex := detectHostIdRegExp.SubexpIndex("HostId")
    if matchIndex >= 0 && matchIndex < len(matches) {
        rb.parsedHostId = matches[matchIndex]
    }

    matches = detectRegionRegExp.FindStringSubmatch(logStream)
    matchIndex = detectRegionRegExp.SubexpIndex("Region")
    if matchIndex >= 0 && matchIndex < len(matches) {
        rb.parsedRegion = matches[matchIndex]
    }

    if rb.parsedHostId != "" && !rb.HasHostId() {
        rb.SetHostId(logStream)
    }
    builder = rb
    return
}

func (rb *otlpRequestBuilder) MatchHostId(hostId string) (bool) {
    return rb.hostId == hostId
}

func (rb *otlpRequestBuilder) HasHostId() (bool) {
    return rb.hostId != ""
}

func (rb *otlpRequestBuilder) AddLogEntry(itemId string, timestamp int64, message, region string, attributes ...map[string]interface{}) (builder OtlpRequestBuilder) {
    if rb.instrLogsSlice.Len()== 0 {
        rb.instrLogs = rb.instrLogsSlice.AppendEmpty()
    }
    logEntry := rb.instrLogs.Logs().AppendEmpty()
    logEntry.SetName(itemId)
    logEntry.SetTimestamp(pdata.Timestamp(timestamp))
    logEntry.Body().SetStringVal(message)
    if region != "" {
        logEntry.Attributes().UpsertString(semconv.AttributeCloudRegion, region)
    } else if rb.parsedRegion != "" {
        logEntry.Attributes().UpsertString(semconv.AttributeCloudRegion, rb.parsedRegion)
    }

    if attributes != nil {
        for _, attrs := range attributes {
            for key, value := range attrs {
                switch v := value.(type) {
                case string:
                    logEntry.Attributes().UpsertString(key, v)
                case int:
                    logEntry.Attributes().UpsertInt(key, int64(v))
                }
            }
        }
    }
    
    builder = rb
    return
}

func (rb *otlpRequestBuilder) GetLogs() (logs pdata.Logs) {
    logs = rb.logs
    attrs := rb.resLogs.Resource().Attributes()
    attrs.InsertString(semconv.AttributeCloudProvider, semconv.AttributeCloudProviderAWS)

    return
}