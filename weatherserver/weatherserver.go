package weatherserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

// Server is a simple http server that serves weather forecasts. Communicating
// with the weather.gov api via the given Client.
//
// Note: usually I'd like to seperate the server logic from the Client logic,
// such as having a weathergovclient package that's imported by a weatherserver
// package. But for the sake of the exercise, I'll keep it all in this package.
type Server struct {
	Client *http.Client
}

// httpGet is a helper function to make http requests. Assumes url is a valid
// weather.gov endpoint that supports application/ld+json. The returning body
// will be unmarshalled into target.
func httpGet(ctx context.Context, client *http.Client, url string, target any) (err error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return
	}
	request.Header.Set("Accept", "application/ld+json")

	response, err := client.Do(request)
	if err != nil {
		return
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		err = fmt.Errorf("unexpected status code: %d", response.StatusCode)
		return
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		err = fmt.Errorf("failed to read body: %v", err)
		return
	}

	if err = json.Unmarshal(body, target); err != nil {
		err = fmt.Errorf("failed to unmarshal body: %v", err)
		return
	}

	return
}

var acceptedTypes = map[string]struct{}{
	"*/*":        {},
	"text/*":     {},
	"text/plain": {},
}

// validRequest checks the request for validity of the entire requiest and, if
// valid, returns the latitude and longitude. Otherwise, an http error is written
// to the response writer.
func validRequest(writer http.ResponseWriter, request *http.Request) (latitude, longitude float64, valid bool) {

	// filter all requests out except for valid ones
	if request.Method != http.MethodGet {
		http.Error(writer, "only GET allowed", http.StatusMethodNotAllowed)
		return
	}

	if request.URL.Path != "/weather" {
		http.Error(writer, "only /weather allowed", http.StatusNotFound)
		return
	}

	// Provide an error if they explicitly accept anything other than accept
	// text/plain.

	accept := request.Header.Get("Accept")
	if _, ok := acceptedTypes[accept]; !ok {
		http.Error(writer, "must accept text/plain", http.StatusNotAcceptable)
		return
	}

	var latitudeString = request.URL.Query().Get("lat")
	var longitudeString = request.URL.Query().Get("lon")

	if latitudeString == "" || longitudeString == "" {
		http.Error(writer, "query params 'lat' and 'lon' are required", http.StatusBadRequest)
		return
	}

	latitude, err := strconv.ParseFloat(latitudeString, 64)
	if err != nil {
		http.Error(writer, "invalid 'lat' query param: "+err.Error(), http.StatusBadRequest)
		return
	}

	longitude, err = strconv.ParseFloat(longitudeString, 64)
	if err != nil {
		http.Error(writer, "invalid 'lon' query param: "+err.Error(), http.StatusBadRequest)
		return
	}

	valid = true

	return
}

func (s Server) ServeHTTP(writer http.ResponseWriter, request *http.Request) {

	// shortcuts for this exercise:
	//
	//  1. no DDOS protection, just for the sake of the exercise. Usually I like
	//     to add some process-wide ip-based rate limiting.
	//  2. no caching. Lots of great ways to do this, but we'll just have the entire
	//     api cycle in-lined.
	//  3. no CORS headers. Again, lots of ways to do this, but we'll keep
	//     it simple.

	// set a timeout for the request as a whole.
	ctx := request.Context()
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	lat, lon, valid := validRequest(writer, request)
	if !valid {
		return
	}

	forecastEndpoint, err := forcastGrid(ctx, s.Client, lat, lon)
	if err != nil {

		// usually for vendor services (eg this external api) I'd log the
		// error and return a general 500 error. But for the sake of the
		// exercise, I'll return the upstream error too.
		http.Error(writer, "failed to fetch weather.gov points: "+err.Error(), http.StatusInternalServerError)
		return
	}

	weather, temprature, err := forecastGet(ctx, s.Client, forecastEndpoint)
	if err != nil {
		http.Error(writer, "failed to fetch weather.gov forecast/temprature: "+err.Error(), http.StatusInternalServerError)
		return
	}

	response := []byte(weather + ", " + temprature + "\n")
	writer.Header().Set("Content-Type", "text/plain")
	writer.Header().Set("Content-Length", strconv.Itoa(len(response)))

	_, _ = writer.Write(response)
	return
}

// forcastGrid uses the weather.gov api to generate an endpoint for the forecast
// for the given lat/lon.
func forcastGrid(ctx context.Context, client *http.Client, lat, lon float64) (forecastEndpoint string, err error) {

	const api = "https://api.weather.gov/points/%f,%f"

	type returnPayload struct {
		ForecastEndpoint string `json:"forecast"`
	}

	var payload returnPayload

	if err = httpGet(ctx, client, fmt.Sprintf(api, lat, lon), &payload); err != nil {
		return
	}

	return payload.ForecastEndpoint, nil
}

// forecastGet uses the weather.gov api to fetch the endpoint (assumed to be a
// /gridpoints/{wfo}/{x},{y}/forecast endpoint) and returns the current weather
// and temperature. More specifically, a mapped temprature and shortForecast of
// the first period.
//
// Note: does NOT support Celsius forecast. Returning temprature is assumed to
// be Fahrenheit (temperatureUnit = F).
func forecastGet(ctx context.Context, client *http.Client, endpoint string) (forecast, temprature string, err error) {

	type returnPayload struct {
		Periods []struct {
			Temperature   float64 `json:"temperature"`
			ShortForecast string  `json:"shortForecast"`
		} `json:"periods"`
	}

	var payload returnPayload

	if err = httpGet(ctx, client, endpoint, &payload); err != nil {
		return
	}

	if len(payload.Periods) == 0 {
		err = fmt.Errorf("no periods found")
		return
	}

	forecast = payload.Periods[0].ShortForecast
	temprature = tempratureAnalog(payload.Periods[0].Temperature)

	return
}

func tempratureAnalog(t float64) string {

	// I feel like this exercise is designed to judge my opinions on what a
	// comfortable temperature is lol

	if t < 40 {
		return "cold"
	}
	if t < 80 {
		return "moderate"
	}

	return "hot"
}
