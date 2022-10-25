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

#r "Azure.Messaging.EventHubs"
#r "System.Text.Json"
#r "System.Memory.Data"


using System;
using System.Text;
using Azure.Messaging.EventHubs;
using System.Net.Http;
using System.Net.Http.Headers;
using System.Text.Json;


static async Task PostLog(Dictionary<string, string> otelAttributes, byte[] data)
{
    var uri = System.Environment.GetEnvironmentVariable("SWI_OTEL_ENDPOINT", EnvironmentVariableTarget.Process) ?? throw new InvalidOperationException("SWI_OTEL_ENDPOINT is not defined");
    var token = System.Environment.GetEnvironmentVariable("SWI_API_KEY", EnvironmentVariableTarget.Process) ?? throw new InvalidOperationException("SWI_API_KEY is not defined");

    using (HttpClient client = new HttpClient())
    {
        using (var content = new ByteArrayContent(data))
        {
            client.DefaultRequestVersion = new Version(2, 0);
            client.DefaultRequestHeaders.Authorization = new AuthenticationHeaderValue("Bearer", token);

            foreach (var attr in otelAttributes)
            {
                client.DefaultRequestHeaders.Add("X-Otel-Resource-Attr", $"{attr.Key}={attr.Value}");
            }

            content.Headers.ContentType = new MediaTypeHeaderValue("application/octet-stream");
            var response = await client.PostAsync(uri, content);

            response.EnsureSuccessStatusCode();
        }
    }
}

static async Task ProcessLogRecord(JsonElement record)
{
    var otelProps = new Dictionary<string, string>()
    {
        { "cloud.provider", "azure" }
    };

    if (record.TryGetProperty("time", out JsonElement time))
    {
        var dateTime = DateTime.Parse(time.GetString());
        var unixTimestamp = ((DateTimeOffset)dateTime).ToUnixTimeMilliseconds() * 1000000;

        otelProps.Add("Timestamp", unixTimestamp.ToString());
    }

    if (record.TryGetProperty("resourceId", out JsonElement resourceId))
    {
        otelProps.Add("service.instance.id", resourceId.GetString());
    }

    if (record.TryGetProperty("location", out JsonElement location))
    {
        otelProps.Add("cloud.region", location.GetString());
    }

    if (record.TryGetProperty("level", out JsonElement level))
    {
        otelProps.Add("SeverityText", level.GetString());
    }

    var data = JsonSerializer.SerializeToUtf8Bytes(record);
    string elemStr = System.Text.Encoding.UTF8.GetString(data);

    await PostLog(otelProps, data);
}

public static async Task Run(EventData[] events, ILogger logger)
{
    var exceptions = new List<Exception>();

    foreach (EventData eventData in events)
    {
        try
        {
            string messageBody = Encoding.UTF8.GetString(eventData.EventBody);
            logger.LogDebug($"C# Event Hub trigger function processed a message: {messageBody}");

            var log = JsonSerializer.Deserialize<dynamic>(messageBody);

            if (log.TryGetProperty("records", out JsonElement recordsElement))
            {
                var records = recordsElement.EnumerateArray();
                foreach (var record in records)
                {
                    await ProcessLogRecord(record);
                }
            }
            else
            {
                await ProcessLogRecord(log);
            }
        }
        catch (Exception e)
        {
            // We need to keep processing the rest of the batch - capture this exception and continue.
            // Also, consider capturing details of the message that failed processing so it can be processed again later.
            exceptions.Add(e);
        }
    }

    // Once processing of the batch is complete, if any messages in the batch failed processing throw an exception so that there is a record of the failure.

    if (exceptions.Count > 1)
        throw new AggregateException(exceptions);

    if (exceptions.Count == 1)
        throw exceptions.Single();
}
