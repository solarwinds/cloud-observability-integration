## Install the Windows Diagnostics extension into Azure virtual machines

To send virtual machine logs to a SolarWinds endpoint, install the Windows Diagnostics extension and configure it using the [install_windows_diag_ext.ps1](install_windows_diag_ext.ps1) PowerShell script. This script internally uses the [Set-AzVMDiagnosticsExtension](https://docs.microsoft.com/en-us/powershell/module/az.compute/set-azvmdiagnosticsextension?view=azps-7.4.0) command.

Required arguments:
- `$vmName` Name of the virtual machine 

- `$vmResourceGroup` Virtual machine resource group

- `$storageAccountName` Name of the Storage account. If the SolarWinds log processing pipeline was created using [default parameters](../../template/deploy-swi-azure-logs-forwarder.ps1), the storage account name is `fnstorswilogs`.

- `$storageAccountKey` API key for accessing storage account. You can copy its value from the Azure Portal. Go to "Your Storage Account" > "Access keys" and copy key1.

- `$eventHubUri` Event hub URI in the format `https://<Namespace>.servicebus.windows.net/<Event-Hub>`, for example `https://swi-logsns.servicebus.windows.net/swi-logs`.

- `$eventHubPolicyName` and `$eventHubAccessPolicyKey` Name of the policy and policy key for accessing the SolarWinds log processing pipeline event hub. You can copy the values from the Azure Portal. Go to "Your Event Hub" > Shared access policies.
The default policy name for the SolarWinds pipeline is `sendlogs`.

## Additional information

[Configuration example](https://docs.microsoft.com/en-us/azure/virtual-machines/extensions/diagnostics-windows#sample-diagnostics-configuration)

[Configuration schema](https://docs.microsoft.com/en-us/azure/azure-monitor/agents/diagnostics-extension-schema-windows#xml)

[Diagnostics extension troubleshooting](https://docs.microsoft.com/en-us/azure/azure-monitor/agents/diagnostics-extension-troubleshooting)


