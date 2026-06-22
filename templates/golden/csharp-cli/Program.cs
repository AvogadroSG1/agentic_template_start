public static class GreetingBuilder
{
    public static string BuildGreeting(string name) => $"hello, {name}!";
}

var target = args.Length > 0 ? args[0] : "world";
Console.WriteLine(GreetingBuilder.BuildGreeting(target));
