package test

import (
	"net/url"
	"regexp"
	"strings"
)

type runtimeSmokeContract struct {
	command       string
	args          []string
	envOverrides  []string
	readinessURLs []string
	responseToken string
}

func csharpWebAPIRuntimeSmokeContract() runtimeSmokeContract {
	return runtimeSmokeContract{
		command:      "dotnet",
		args:         []string{"run"},
		envOverrides: nil,
		readinessURLs: []string{
			"http://localhost:5000/Weatherforecast",
			"http://localhost:5000/WeatherForecast",
		},
		responseToken: "temperaturec",
	}
}

var aspNetListeningURLPattern = regexp.MustCompile(`Now listening on:\s+(https?://\S+)`)

func csharpWebAPIReadinessURLs(output string, contract runtimeSmokeContract) []string {
	matches := aspNetListeningURLPattern.FindAllStringSubmatch(output, -1)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		baseURL := strings.TrimRight(match[1], "/")
		if !strings.HasPrefix(baseURL, "http://") {
			continue
		}

		if parsedURL, err := url.Parse(baseURL); err == nil && parsedURL.Hostname() == "localhost" {
			loopbackURL := "http://127.0.0.1"
			if port := parsedURL.Port(); port != "" {
				loopbackURL += ":" + port
			}

			return []string{
				loopbackURL + "/Weatherforecast",
				loopbackURL + "/WeatherForecast",
				baseURL + "/Weatherforecast",
				baseURL + "/WeatherForecast",
			}
		}

		return []string{baseURL + "/Weatherforecast", baseURL + "/WeatherForecast"}
	}

	return contract.readinessURLs
}
