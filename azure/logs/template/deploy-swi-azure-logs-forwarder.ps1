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