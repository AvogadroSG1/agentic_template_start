// <copyright file="HealthEndpointTests.cs" company="{{.AuthorName}}">
// Copyright (c) {{.AuthorName}}. All rights reserved.
// </copyright>

using System.Net;
using Microsoft.AspNetCore.TestHost;
using Xunit;

/// <summary>
/// Verifies the scaffolded health endpoint responds successfully.
/// </summary>
public sealed class HealthEndpointTests
{
    /// <summary>
    /// Ensures the health endpoint returns HTTP 200.
    /// </summary>
    /// <returns>A task that completes when the assertion finishes.</returns>
    [Fact]
    public async Task HealthEndpointReturnsOk()
    {
        var builder = Program.CreateBuilder(Array.Empty<string>());
        builder.WebHost.UseTestServer();

        await using var app = builder.Build();
        Program.ConfigureApp(app);
        await app.StartAsync();

        using var client = app.GetTestClient();

        var response = await client.GetAsync("/health");

        Assert.Equal(HttpStatusCode.OK, response.StatusCode);
    }
}
