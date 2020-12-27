package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"go.opentelemetry.io/otel/propagation"

	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/otel/semconv"

	"go.opentelemetry.io/otel"

	"go.opentelemetry.io/otel/exporters/trace/jaeger"

	"go.opentelemetry.io/otel/label"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Book is the example entity
type Book struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

var inMemBookStore = make(map[string]Book)
var mtx = sync.RWMutex{}

// It's recommended to create a tracer for each package. The name of the tracer will be set as span attribute: "otel.instrumentation_library.name"
// If you set the name e.g. as the package name it's easier to debug if smth. goes wrong.
var rootTrace = otel.Tracer("github.com/fwiedmann/open-telemetry/server")

func main() {
	for i := 0; i < 10; i++ {
		inMemBookStore[fmt.Sprintf("%d", i)] = Book{
			ID:   fmt.Sprintf("%d", i),
			Name: fmt.Sprintf("book-%d", i),
		}
	}

	flush := initTracer()
	defer flush()

	// otelhttp.NewHandler wraps a OpenTelemetry HTTP middleware to the user specific handler.
	// The handler will add basic tracing meta information such as the resp status.
	http.Handle("/books", otelhttp.NewHandler(addHostnameToTrace(listBooks), "list-books"))
	http.Handle("/error", otelhttp.NewHandler(addHostnameToTrace(errorTrace), "get-error"))
	fmt.Print("started server\n")
	panic(http.ListenAndServe(":8080", nil))
}

// listBooks http.HandlerFunc which returns a json list of the current stored books
func listBooks(w http.ResponseWriter, r *http.Request) {
	mtx.RLock()
	defer mtx.RUnlock()

	var books []Book
	for _, b := range inMemBookStore {
		books = append(books, b)
	}

	jsonResp, err := json.Marshal(books)
	if err != nil {
		fmt.Print(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}

	ctx := r.Context()
	span := trace.SpanFromContext(ctx)
	// The event is context based for the span of the current request. The event can be viewed in the Jaeger UI
	// which will contain all the exact body which will be send to the client.
	span.AddEvent("current listed books", trace.WithAttributes(label.KeyValue{
		Key:   "response body",
		Value: label.StringValue(string(jsonResp)),
	}))

	_, err = w.Write(jsonResp)
	if err != nil {
		fmt.Print(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
	sleep(ctx)
}

func sleep(ctx context.Context) {
	ctx, sleepSpan := rootTrace.Start(ctx, "sleep()")
	defer sleepSpan.End()
	time.Sleep(time.Millisecond * 100)
}

// errorTrace http.HandlerFunc which returns a http.StatusInternalServerError after sleeping for a period of time
func errorTrace(w http.ResponseWriter, r *http.Request) {
	_, sleepSpan := rootTrace.Start(r.Context(), "sleepWithError()")

	// Add an error to the current span. This will be visualized in the jaeger UI for the
	sleepSpan.RecordError(errors.New("could not list books"))
	defer sleepSpan.End()
	time.Sleep(time.Millisecond * 100)
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

// addHostnameToTrace is a HTTP middleware which will set the hostname to the current span
func addHostnameToTrace(handler http.HandlerFunc) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		trace.SpanFromContext(request.Context()).SetAttributes(label.KeyValue{
			Key:   semconv.HostNameKey,
			Value: label.StringValue("book-server-service@localhost"),
		})
		handler.ServeHTTP(writer, request)
	}
}

// initTracer creates a new trace provider instance and registers it as global trace provider.
func initTracer() func() {
	// This enables the propagation of tracing and baggage headers for ingress/egress traffic to services
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	// Create and install Jaeger export pipeline.
	flush, err := jaeger.InstallNewPipeline(
		jaeger.WithCollectorEndpoint("http://jaeger:14268/api/traces"),
		jaeger.WithProcess(jaeger.Process{
			ServiceName: "book-service-server",
			Tags: []label.KeyValue{
				label.String("exporter", "jaeger"),
			},
		}),
		jaeger.WithSDK(&sdktrace.Config{}),
	)
	if err != nil {
		log.Fatal(err)
	}
	return flush
}
