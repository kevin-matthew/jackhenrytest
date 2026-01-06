package weatherserver

import (
	"net/http"
	"testing"
)
import "github.com/stretchr/testify/assert"

// note that if these were real production-level unit tests, I'd really hammer
// into testing that expected errors returned as well. But these tests just make
// sure the weather gov api simply works.

const KCLat = 39.0997
const KCLon = -94.5786
const KCWeatherEndpoint = "https://api.weather.gov/gridpoints/EAX/44,51/forecast"

func TestForcastGrid(t *testing.T) {

	// shortcut: normally I would NOT use a real api endpoint, nor a real http
	// client. I'd use httptest.

	endpoint, err := forcastGrid(t.Context(), http.DefaultClient, KCLat, KCLon)
	assert.NoError(t, err)
	assert.Equal(t, KCWeatherEndpoint, endpoint)
}

func TestForcastGet(t *testing.T) {

	// shortcut: normally I would NOT use a real api endpoint, nor a real http
	// client. I'd use httptest.

	forecast, temprature, err := forecastGet(t.Context(), http.DefaultClient, KCWeatherEndpoint)
	assert.NoError(t, err)
	assert.NotEmpty(t, temprature)
	assert.NotEmpty(t, forecast)
}

func TestTempratureAnalog(t *testing.T) {
	assert.Equal(t, "hot", tempratureAnalog(100))
	assert.Equal(t, "moderate", tempratureAnalog(72))
	assert.Equal(t, "cold", tempratureAnalog(-1398123.3))
}
