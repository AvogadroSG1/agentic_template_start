// --------------------------------------------------------------------------------------------------------------------
// <copyright file="GreetingBuilder.cs" company="Stack Overflow">
//   Copyright (c) Stack Overflow. All rights reserved.
// </copyright>
// --------------------------------------------------------------------------------------------------------------------

/// <summary>Builds starter greeting strings for the CLI scaffold.</summary>
public static class GreetingBuilder
{
    /// <summary>Builds the greeting for the provided name.</summary>
    /// <param name="name">The name to greet.</param>
    /// <returns>The formatted greeting.</returns>
    public static string BuildGreeting(string name) => $"hello, {name}!";
}
