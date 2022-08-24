# Copyright 2022 SolarWinds Worldwide, LLC. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at:
#
#	http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and limitations
# under the License.

param (
    [Parameter(mandatory)]
    $SwiApiKey,
    $swiOtelEndpoint = " https://api.dc-01.cloud.solarwinds.com/v1/logs",

    $ResourceGroupLocation  = "eastus",
    $ResourceGroupName = "swi-logs",
    $Projectname = "swi-logs",
    $FunctionName = "forwarder-function"
)

$code = Get-Content .\run.csx -Raw

# Create ResoureGroup
New-AzResourceGroup -Name $ResourceGroupName -Location $ResourceGroupLocation

$deploymentArgs = @{
    TemplateFile = "resource_template.json"
    ResourceGroupName = $ResourceGroupName
    Location = $ResourceGroupLocation

    ProjectName = $Projectname
    FunctionName = $FunctionName
    FunctionSourceCode = $code

    SwiApiKey = $swiApiKey
    SwiOtelEndpoint = $swiOtelEndpoint
}

try {
    New-AzResourceGroupDeployment @deploymentArgs -Verbose -ErrorAction Stop
} catch {
    Write-Error "Deployment failed"
    Write-Error $_
}