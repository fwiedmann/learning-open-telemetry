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

var rootTrace = otel.Tracer("book-service-demo-server")

func main() {
	for i := 0; i < 10; i++ {
		inMemBookStore[fmt.Sprintf("%d", i)] = Book{
			ID:   fmt.Sprintf("%d", i),
			Name: fmt.Sprintf("book-%d", i),
		}
	}

	flush := initTracer()
	defer flush()

	http.Handle("/books", otelhttp.NewHandler(http.HandlerFunc(listBooks), "list-books"))
	panic(http.ListenAndServe(":8080", nil))
}

func listBooks(w http.ResponseWriter, r *http.Request) {
	fmt.Print(r.Header)

	ctx := r.Context()
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(label.KeyValue{Key: semconv.HostNameKey, Value: label.StringValue("test-server-s01")})
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

	span.AddEvent("current listed books", trace.WithAttributes(label.KeyValue{
		Key:   "response body",
		Value: label.StringValue(string(jsonResp)),
	}))

	_, err = w.Write(jsonResp)
	if err != nil {
		fmt.Print(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
	time.Sleep(time.Millisecond * 100)
	span.End()
	sleep(ctx)
	sleepWithError(ctx)
}

func sleep(ctx context.Context) {
	ctx, sleepSpan := rootTrace.Start(ctx, "sleep()")
	defer sleepSpan.End()
	time.Sleep(time.Millisecond * 100)
}

func sleepWithError(ctx context.Context) {
	ctx, sleepSpan := rootTrace.Start(ctx, "sleepWithError()")
	sleepSpan.RecordError(errors.New("error"))
	defer sleepSpan.End()
	time.Sleep(time.Millisecond * 100)
}

// initTracer creates a new trace provider instance and registers it as global trace provider.
func initTracer() func() {
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	// Create and install Jaeger export pipeline.
	flush, err := jaeger.InstallNewPipeline(
		jaeger.WithCollectorEndpoint("http://localhost:14268/api/traces"),
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
