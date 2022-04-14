param
(
[Parameter(Mandatory, HelpMessage="Virtual machine name")]
[string]$vmName,
[Parameter(Mandatory, HelpMessage="Virtual machine name resource group")]
[string]$vmResourceGroup,

[Parameter(Mandatory, HelpMessage="Storage account created for logs processing, eg. fnstorswilogs")]
[string]$storageAccountName,
[Parameter(Mandatory, HelpMessage="Storage account key")]
[string]$storageAccountKey,

[Parameter(Mandatory, HelpMessage="Event hub URI, eg. https://myNamespace.servicebus.windows.net/myEventHub")]
[string]$eventHubUri,

[Parameter(Mandatory, HelpMessage="Policy name for accessing event hub")]
[string]$eventHubAccessPolicyName,
[Parameter(Mandatory, HelpMessage="Policy key for accessing event hub")]
[string]$eventHubAccessPolicyKey
)

# Get the VM object
$vm = Get-AzVM -Name $vmName -ResourceGroupName $VMresourceGroup

$configTemplate_path = Join-Path $PSScriptRoot  "config_template.xml"
$diagnosticsConfig_path = Join-Path $PSScriptRoot "config_replaced.xml"

# Get the public settings template and update the templated values
[xml]$config = Get-Content $configTemplate_path -Raw

$config.DiagnosticsConfiguration.PublicConfig.WadCfg.DiagnosticMonitorConfiguration.Metrics.SetAttribute("resourceId", $vm.Id)
$config.DiagnosticsConfiguration.PublicConfig.WadCfg.SinksConfig.Sink.EventHub.SetAttribute("Url", $eventHubUri)
$config.DiagnosticsConfiguration.PublicConfig.WadCfg.SinksConfig.Sink.EventHub.SetAttribute("SharedAccessKeyName", $eventHubAccessPolicyName)

$config.DiagnosticsConfiguration.PublicConfig.StorageAccount = $storageAccountName

$config.DiagnosticsConfiguration.PrivateConfig.StorageAccount.SetAttribute("name", $storageAccountName)
$config.DiagnosticsConfiguration.PrivateConfig.StorageAccount.SetAttribute("key", $storageAccountKey)

$config.DiagnosticsConfiguration.PrivateConfig.EventHub.SetAttribute("Url", $eventHubUri)
$config.DiagnosticsConfiguration.PrivateConfig.EventHub.SetAttribute("SharedAccessKeyName", $eventHubAccessPolicyName)
$config.DiagnosticsConfiguration.PrivateConfig.EventHub.SetAttribute("SharedAccessKey", $eventHubAccessPolicyKey)

$config.Save($diagnosticsConfig_path)

# Install the extension
Set-AzVMDiagnosticsExtension -ResourceGroupName $vmResourceGroup -VMName $vmName -DiagnosticsConfigurationPath $diagnosticsConfig_path

# Show installed Extensions
Get-AzVMExtension -ResourceGroupName $vmResourceGroup -VMName $vmName


