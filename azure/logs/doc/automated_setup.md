# Automated setup

Install the Azure PowerShell module:
https://docs.microsoft.com/en-us/powershell/azure/install-az-ps-msi?view=azps-7.2.0

Download the contents of [template directory](../template)

Connect to Azure:
Connect-AzAccount -Tenant xxxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxx -Subscription yyyyyyyyy-yyyy-yyyy-yyyy-yyyyyyyyy

Obtain your API key from the SolarWinds Observability portal:

Run the deployment script:
./deploy-swi-azure-logs-forwarder.ps1 -SwiApiKey <api_key>

## Logs forwarding
Logs must be forwarded for each resource from which logs will be monitored. It can be done following [guide](logs_forwarding.md)