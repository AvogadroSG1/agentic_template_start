package test

import (
	"slices"
	"testing"
)

func TestCSharpWebAPIRuntimeSmokeContractUsesDotnetRun(t *testing.T) {
	contract := csharpWebAPIRuntimeSmokeContract()

	if contract.command != "dotnet" {
		t.Fatalf("command = %q, want dotnet", contract.command)
	}

	if !slices.Equal(contract.args, []string{"run"}) {
		t.Fatalf("args = %v, want [run]", contract.args)
	}

	if len(contract.envOverrides) != 0 {
		t.Fatalf("envOverrides = %v, want none", contract.envOverrides)
	}
}

func TestCSharpWebAPIRuntimeSmokeContractProbesDefaultWeatherForecastEndpoint(t *testing.T) {
	contract := csharpWebAPIRuntimeSmokeContract()

	expectedURLs := []string{
		"http://localhost:5000/Weatherforecast",
		"http://localhost:5000/WeatherForecast",
	}
	if !slices.Equal(contract.readinessURLs, expectedURLs) {
		t.Fatalf("readinessURLs = %v, want %v", contract.readinessURLs, expectedURLs)
	}

	if contract.responseToken != "temperaturec" {
		t.Fatalf("responseToken = %q, want temperaturec", contract.responseToken)
	}
}

func TestCSharpWebAPIRuntimeReadinessURLsPreferListeningAddressFromOutput(t *testing.T) {
	contract := csharpWebAPIRuntimeSmokeContract()
	output := `info: Microsoft.Hosting.Lifetime[14]
      Now listening on: http://localhost:59991
info: Microsoft.Hosting.Lifetime[14]
      Now listening on: https://localhost:7012`

	urls := csharpWebAPIReadinessURLs(output, contract)
	expected := []string{
		"http://127.0.0.1:59991/Weatherforecast",
		"http://127.0.0.1:59991/WeatherForecast",
		"http://localhost:59991/Weatherforecast",
		"http://localhost:59991/WeatherForecast",
	}
	if !slices.Equal(urls, expected) {
		t.Fatalf("readiness URLs = %v, want %v", urls, expected)
	}
}

func TestCSharpWebAPIRuntimeReadinessURLsFallBackToDefaultURLsWithoutListeningOutput(t *testing.T) {
	contract := csharpWebAPIRuntimeSmokeContract()

	urls := csharpWebAPIReadinessURLs("", contract)
	if !slices.Equal(urls, contract.readinessURLs) {
		t.Fatalf("readiness URLs = %v, want fallback %v", urls, contract.readinessURLs)
	}
}
