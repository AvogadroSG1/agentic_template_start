// --------------------------------------------------------------------------------------------------------------------
// <copyright file="Program.cs" company="Stack Overflow">
//   Copyright (c) Stack Overflow. All rights reserved.
// </copyright>
// --------------------------------------------------------------------------------------------------------------------

var target = args.Length > 0 ? args[0] : "world";
Console.WriteLine(GreetingBuilder.BuildGreeting(target));
