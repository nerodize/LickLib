package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	/*
		SLI 1: Upload Erfolgsrate
		...
	*/
	TrackUploadsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "licklib_track_uploads_total",
		Help: "Anzahl Track-Uploads nach Ergebnis",
	}, []string{"status"}) // Label hinzufügen

	/*
		wichtig für SLI 2:
	*/
	TrackUploadDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "licklib_track_upload_duration_seconds",
		Help:    "Dauer eines Track-Uploads in Sekunden", // sollte nicht verwendet werden
		Buckets: prometheus.DefBuckets,
	})

	/*
		wichtig für SLI 3: HTTP-Fehlerrate
	*/
	HTTPRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "licklib_http_requests_total", // ✔️
		Help: "Anzahl HTTP-Requests nach Method und Status",
	}, []string{"method", "path", "status"})

	/*
		wichtig für SLI 4: API-Latenz p95 -> 95% Quantil
	*/
	HTTPRequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "licklib_http_request_duration_seconds",
		Help:    "HTTP Request Latenz", // ✔️ der wird benutzt nicht der andere
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path", "status"})
)

func Init() {
	prometheus.MustRegister(
		TrackUploadsTotal,
		TrackUploadDuration,
		HTTPRequestsTotal,
		HTTPRequestDuration,
	)
}
