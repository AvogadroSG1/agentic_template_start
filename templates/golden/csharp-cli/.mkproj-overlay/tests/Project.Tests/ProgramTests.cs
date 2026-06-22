using Xunit;

public sealed class ProgramTests
{
    [Fact]
    public void BuildGreetingUsesTheProvidedName()
    {
        Assert.Equal("hello, Peter!", GreetingBuilder.BuildGreeting("Peter"));
    }
}
