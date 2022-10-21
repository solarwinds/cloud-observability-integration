# Automated setup

Install the Azure PowerShell module:
https://docs.microsoft.com/en-us/powershell/azure/install-az-ps-msi?view=azps-7.2.0

Download the contents of [template directory](../template)

Open powershell console and connect to Azure:

`Connect-AzAccount -Tenant xxxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxx -Subscription yyyyyyyyy-yyyy-yyyy-yyyy-yyyyyyyyy`

Obtain and copy your API key from the SolarWinds Observability portal - see [API Tokens](https://documentation.solarwinds.com/en/success_center/observability/content/settings/api-tokens.htm) for details.

Run the deployment script in powershell console:

`./deploy-swi-azure-logs-forwarder.ps1 -SwiApiKey YourSolarWindsApiToken -swiOtelEndpoint YourOtelEndpoint -ResourceGroupName YourResourceGroupName -ProjectName YourProjectName -FunctionName YourFunctionName -ResourceGroupLocation YourResourceGroupLocation`

Replace:
1. YourSolarWindsApiToken with the text copied previously.
2. YourOtelEndpoint with your organization's Otel endpoint. See Endpoint URIs to determine your organization's endpoint. 
3. YourResourceGroupName with name defined for new resource group which will be created by script. Name of resource group must be unique, it means that resource group cannot be already created with same name in azure tenant to which logs resources will be deployed. Optional parameter, default value is `swi-logs`.
4. YourProjectName with name defined for new project which will be created by script. Name of project must be unique, it means that project cannot be already created with same name in azure tenant to which logs resources will be deployed. Optional parameter, default value is `swi-logs`.
5. YourFunctionName with name defined for function which will be created by script. Name of function must be unique, it means that function cannot be already created with same name in azure tenant to which logs resources will be deployed. Optional parameter, default value is `forwarder-function`.
6. YourResourceGroupLocation with region name in which will these resource deployed. Optional parameter, default value is `eastus`.


## Logs forwarding
Forward logs you want to see in website to created event hub. It can be done following [guide](logs_forwarding.md)
