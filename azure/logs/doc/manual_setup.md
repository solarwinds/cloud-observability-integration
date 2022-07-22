# Manual installation

This document describes how to stream logs to SolarWinds Observability. An Azure resource generates logs and sends them to Event Hub. An Azure function processes such events, reads logs data, and sends them to SolarWinds Observability.

## [Create a resource group](https://docs.microsoft.com/en-us/azure/event-hubs/event-hubs-create#create-a-resource-group)

- Sign in to the Azure portal.
- In the left-hand navigation, select Resource groups. Then select Add.
- For Subscription, select the name of the Azure subscription in which you want to create the resource group.
= Type a unique name for the resource group. The system immediately checks to see if the name is available in the currently selected Azure subscription.
- Select a region for the resource group.
- Select Review + Create.

## [Create an Event Hubs namespace](https://docs.microsoft.com/en-us/azure/event-hubs/event-hubs-create#create-an-event-hubs-namespace)

- In the Azure portal, select Create a resource.
- Select Event Hubs in the navigational menu and select Add on the toolbar.
- On the Create namespace page, select the subscription, resource group, name, and the location for the namespace.
- Select Review + Create.
- On the Review + Create page, review the settings, and select Create.

## [Create an event hub](https://docs.microsoft.com/en-us/azure/event-hubs/event-hubs-create#create-an-event-hub)

- On the Event Hubs Namespace page, select Event Hubs in the left-hand menu.
- At the top of the window, select + Event Hub.
- Type a name for your event hub, then select Create.

## Set up your forwarding C# function app
Azure function apps have built-in triggers for Event Hub. When the triggers are executed, they pass the contents of log messages to the function app. This way, C# script code can be used to grab the logs and forward the contents to SolarWinds Observability via a simple HTTP POST call.

## [Create function app](https://docs.microsoft.com/en-us/azure/azure-functions/functions-create-function-app-portal#create-a-function-app)

- From the Azure portal menu or the Home page, select Create a resource.
- In the New page, select Compute > Function App.
- On the Basics page, use the function app settings: Subscription, Resource Group, Region, and Function App name.
- Select **Publish to Code, .NET as Runtime stack and Version 6**.
- Select Next:Hosting.
- Select a storage account, **operating system Windows**, and serverless plan type.
- Select Review + create to review the app configuration selections.
- On the Review + create page, review your settings, and then select Create to provision and deploy the function app.

## [Create an EventHub trigger function](https://docs.microsoft.com/en-us/azure/azure-functions/functions-bindings-event-hubs-trigger)
- From the left-hand menu of the Function App window, select Functions, then select Create from the top menu.
- In the Create Function window, ensure the Development environment property has **Develop in portal** and select the **EventHub trigger template**.
- Select Code + Test.
- Copy and Paste [function code](template/run.csx) to run.csx file.
- Add SWI_API_KEY environmental variable containing your API key obtained from SolarWinds portal.
- Add SWI_OTEL_ENDPOINT environmental variable containing the URI of the telemetry endpoint: https://api.dc-01.cloud.solarwinds.com/v1/logs
- Save the function.

## Logs forwarding
Forward logs you want to see in website to created event hub. It can be done following [guide](logs_forwarding.md)
