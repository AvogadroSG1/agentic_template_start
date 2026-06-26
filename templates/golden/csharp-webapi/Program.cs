// <copyright file="Program.cs" company="{{.AuthorName}}">
// Copyright (c) {{.AuthorName}}. All rights reserved.
// </copyright>

var builder = Program.CreateBuilder(args);
var app = builder.Build();
Program.ConfigureApp(app);

app.Run();

/// <summary>
/// Provides shared application construction hooks for runtime and tests.
/// </summary>
public partial class Program
{
    /// <summary>
    /// Creates the application builder for the current entry point.
    /// </summary>
    /// <param name="args">The command-line arguments for the application.</param>
    /// <returns>The configured web application builder.</returns>
    public static WebApplicationBuilder CreateBuilder(string[] args)
    {
        return WebApplication.CreateBuilder(args);
    }

    /// <summary>
    /// Applies the shared HTTP endpoint configuration to the application.
    /// </summary>
    /// <param name="app">The application to configure.</param>
    public static void ConfigureApp(WebApplication app)
    {
        app.MapGet("/health", () => Results.Ok(new { status = "ok" }));
    }
}
