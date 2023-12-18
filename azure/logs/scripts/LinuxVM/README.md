## Install the Linux Diagnostics extension into Azure virtual machines

To send virtual machine logs to a SolarWinds endpoint, install the Windows Diagnostics extension and configure it using the [install_linux_diag_ext.ps1](install_linux_diag_ext.ps1) PowerShell script. This script internally uses the [Set-AzVMExtension](https://docs.microsoft.com/en-us/powershell/module/az.compute/set-azvmextension?view=azps-7.4.0) command.

Required arguments:
- `$vmName` Name of the virtual machine 

- `$vmResourceGroup` Virtual machine resource group

- `$storageAccountName` Name of the Storage account that was created using [default parameters](../../template/deploy-swi-azure-logs-forwarder.ps1)

- `$storageAccountResourceGroup` Resource group where the Storage is located. If the SolarWinds log processing pipeline was created using [default parameters](../../template/deploy-swi-azure-logs-forwarder.ps1), the storage account name is `swi-logs`.

- `$eventHubUri` Event hub URI in the format `https://<Namespace>.servicebus.windows.net/<Event-Hub>`, for example `https://swi-logsns.servicebus.windows.net/swi-logs`.

- `$eventHubPolicyName` and `$eventHubAccessPolicyKey` Name of the policy and policy key for accessing the SolarWinds log processing pipeline event hub. You can copy the values from the Azure Portal. Go to "Your Event Hub Namespace" > Shared access policies.
  The default policy name for the SolarWinds pipeline is `RootManageSharedAccessKey`.

Pre-requisites:
- The Linux diagnostic extension requires Python 2. If your virtual machine uses a distribution that doesn't include Python 2, install it.
- Refer to https://learn.microsoft.com/en-us/azure/virtual-machines/extensions/diagnostics-linux-v3#python-requirement

Usage:
- `./install_linux_diag_ext.ps1 -vmName my-vm-name -vmResourceGroup <my-resource-group-name> -storageAccountName storage-account-name -storageAccountResourceGroup swi-logs -eventHubUri swi-logsns-xyz123.servicebus.windows.net/swi-logs -eventHubPolicyName RootManageSharedAccessKey -eventHubAccessPolicyKey <your-policy-key-here>`


## Additional information

[Linux diagnostic extension documentation](https://docs.microsoft.com/en-us/azure/virtual-machines/extensions/diagnostics-linux?toc=%2Fazure%2Fazure-monitor%2Ftoc.json&tabs=powershell)

