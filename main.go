package main

import (
	_ "embed"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

const port = 8080

//go:embed index.html
var indexHTML []byte

func main() {
	if len(os.Args) > 1 && os.Args[1] == "-check" {
		healthCheck()
	}

	log.Println("AUTHOR Julia Jurczak")

	http.HandleFunc("/", serveHTML)
	http.HandleFunc("/health", serveHealth)
	http.HandleFunc("/api", serveWeather)

	log.Println("PORT", port)
	http.ListenAndServe(":"+strconv.Itoa(port), nil)
}

func serveHTML(w http.ResponseWriter, r *http.Request) {
	log.Println("GET", r.URL.Path)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(indexHTML)
}

func serveHealth(w http.ResponseWriter, r *http.Request) {
	log.Println("GET", r.URL.Path)
	w.Write([]byte("OK"))
}

func serveWeather(w http.ResponseWriter, r *http.Request) {
	coords := r.URL.Query().Get("coords")
	log.Println("GET", r.URL.Path+"?coords="+coords)

	parts := strings.SplitN(coords, ",", 2)
	if len(parts) != 2 {
		w.Write([]byte(`{"error":"wrong coordinates"}`))
		return
	}
	lat, lon := parts[0], parts[1]

	data, err := fetchWeather(lat, lon)
	if err != nil {
		w.Write([]byte(`{"error":"network error"}`))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"temperature":"` + data[0] + `","precipitation":"` + data[1] + `","wind":"` + data[2] + `","humidity":"` + data[3] + `","cloud_cover":"` + data[4] + `"}`))
}

func fetchWeather(lat, lon string) ([5]string, error) {
	client := http.Client{}

	resp, err := client.Get("https://api.open-meteo.com/v1/forecast" +
		"?latitude=" + lat + "&longitude=" + lon +
		"&current=temperature_2m,precipitation,wind_speed_10m,relative_humidity_2m,cloud_cover")
	if err != nil {
		return [5]string{}, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	jsonStr := string(body)
	log.Println("WEATHER RESPONSE", jsonStr)

	if start := strings.Index(jsonStr, `"current":{`); start >= 0 {
		jsonStr = jsonStr[start:]
	}

	return [5]string{
		extract(jsonStr, "temperature_2m"),
		extract(jsonStr, "precipitation"),
		extract(jsonStr, "wind_speed_10m"),
		extract(jsonStr, "relative_humidity_2m"),
		extract(jsonStr, "cloud_cover"),
	}, nil
}

func healthCheck() {
	client := http.Client{}
	r, err := client.Get("http://localhost:" + strconv.Itoa(port) + "/health")
	if err != nil || r.StatusCode != 200 {
		os.Exit(1)
	}
	os.Exit(0)
}

func extract(jsonStr, key string) string {
	needle := `"` + key + `":`

	start := strings.Index(jsonStr, needle)
	if start < 0 {
		return ""
	}

	remaining := jsonStr[start+len(needle):]
	remaining = strings.TrimLeft(remaining, " ")

	if len(remaining) > 0 && remaining[0] == '"' {
		remaining = remaining[1:]
		endQuote := strings.Index(remaining, `"`)
		if endQuote >= 0 {
			return remaining[:endQuote]
		}
		return ""
	}

	end := strings.IndexAny(remaining, ",}]")
	if end >= 0 {
		return remaining[:end]
	}

	return remaining
}
