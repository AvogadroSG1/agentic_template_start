// --------------------------------------------------------------------------------------------------------------------
// <copyright file="ProgramTests.cs" company="Stack Overflow">
//   Copyright (c) Stack Overflow. All rights reserved.
// </copyright>
// --------------------------------------------------------------------------------------------------------------------

using Xunit;

/// <summary>Exercises the starter greeting helper.</summary>
public sealed class ProgramTests
{
    /// <summary>Uses the provided name when building a greeting.</summary>
    [Fact]
    public void BuildGreetingUsesTheProvidedName()
    {
        Assert.Equal("hello, Peter!", GreetingBuilder.BuildGreeting("Peter"));
    }
}
