#https://docs.microsoft.com/en-us/azure/virtual-machines/extensions/diagnostics-linux?toc=%2Fazure%2Fazure-monitor%2Ftoc.json&tabs=powershell

param
(
[Parameter(Mandatory, HelpMessage="Virtual machine name")]
[string]$vmName,
[Parameter(Mandatory, HelpMessage="Virtual machine name resource group")]
[string]$vmResourceGroup,

[Parameter(Mandatory, HelpMessage="Storage account created for logs processing, eg. fnstorswilogs")]
[string]$storageAccountName,
[Parameter(Mandatory, HelpMessage="Storage account resource group")]
[string]$storageAccountResourceGroup,

[Parameter(Mandatory, HelpMessage="Event hub URI without prefix, eg. swi-logsns.servicebus.windows.net/swi-logs")]
[string]$eventHubUri,

[Parameter(Mandatory, HelpMessage="Policy name for accessing event hub")]
[string]$eventHubPolicyName,
[Parameter(Mandatory, HelpMessage="Policy key for accessing event hub")]
[string]$eventHubAccessPolicyKey
)

# Get the VM object
$vm = Get-AzVM -Name $vmName -ResourceGroupName $vmResourceGroup

# Get the public settings template and update the templated values for the storage account and resource ID
$public_settings_path = Join-Path $PSScriptRoot "public_settings.json"

$publicSettings = Get-Content $public_settings_path -Raw | ConvertFrom-Json
$publicSettings.StorageAccount = $storageAccountName
$publicSettings.ladCfg.diagnosticMonitorConfiguration.metrics.resourceId = $vm.Id
$publicSettingsString = $publicSettings | ConvertTo-Json -Depth 100

# https://docs.microsoft.com/en-us/rest/api/eventhub/generate-sas-token#powershell
Add-Type -AssemblyName System.Web
$Expires=([DateTimeOffset]::Now.ToUnixTimeSeconds())+3600*24*365*2
$SignatureString=[System.Web.HttpUtility]::UrlEncode($eventHubUri)+ "`n" + [string]$Expires
$HMAC = New-Object System.Security.Cryptography.HMACSHA256
$HMAC.key = [Text.Encoding]::ASCII.GetBytes($eventHubAccessPolicyKey)
$Signature = $HMAC.ComputeHash([Text.Encoding]::ASCII.GetBytes($SignatureString))
$Signature = [Convert]::ToBase64String($Signature)
$token = "sr=" + [System.Web.HttpUtility]::UrlEncode($eventHubUri) + "&sig=" + [System.Web.HttpUtility]::UrlEncode($Signature) + "&se=" + $Expires + "&skn=" + $eventHubPolicyName
$sasUrl = "https://" + $eventHubUri + '?' + $token

# Generate a SAS token for the agent to authenticate with the storage account
$sasToken = New-AzStorageAccountSASToken -Service Blob,Table -ResourceType Service,Container,Object -Permission "racwdlup" -Context (Get-AzStorageAccount -ResourceGroupName $storageAccountResourceGroup -AccountName $storageAccountName).Context -ExpiryTime $([System.DateTime]::Now.AddYears(2))
$sasToken = $sasToken.Substring(1,($sasToken.Length-1));

# Get the protected settings template and update the templated values for the storage account SAS token and event hub sink
$protectedSettings_path = Join-Path $PSScriptRoot "protected_settings.json"
$protectedSettings = Get-Content $protectedSettings_path -Raw | ConvertFrom-Json
$protectedSettings.storageAccountName = $storageAccountName
$protectedSettings.storageAccountSasToken = $sasToken
$protectedSettings.sinksConfig.sink[0].sasURL = $sasUrl
$protectedSettingsString = $protectedSettings | ConvertTo-Json -Depth 100

# Install the extension
Set-AzVMExtension -ResourceGroupName $vmResourceGroup -VMName $vmName -Location $vm.Location -ExtensionType LinuxDiagnostic -Publisher Microsoft.Azure.Diagnostics -Name LinuxDiagnostic -SettingString $publicSettingsString -ProtectedSettingString $protectedSettingsString -TypeHandlerVersion 4.0

# Show installed Extensions
Get-AzVMExtension -ResourceGroupName $vmResourceGroup -VMName $vmName
