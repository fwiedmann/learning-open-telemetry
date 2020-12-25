package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"go.opentelemetry.io/otel/exporters/trace/jaeger"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
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

	time.Sleep(10 * time.Millisecond)

	_, err = w.Write(jsonResp)
	if err != nil {
		fmt.Print(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func main() {
	for i := 0; i < 10; i++ {
		inMemBookStore[fmt.Sprintf("%d", i)] = Book{
			ID:   fmt.Sprintf("%d", i),
			Name: fmt.Sprintf("book-%d", i),
		}
	}

	flush := initTracer()
	defer flush()

	http.Handle("/books", otelhttp.NewHandler(http.HandlerFunc(listBooks), "list books"))
	panic(http.ListenAndServe(":8080", nil))
}

// initTracer creates a new trace provider instance and registers it as global trace provider.
func initTracer() func() {
	// Create and install Jaeger export pipeline.
	flush, err := jaeger.InstallNewPipeline(
		jaeger.WithCollectorEndpoint("http://localhost:14268/api/traces"),
		jaeger.WithProcess(jaeger.Process{
			ServiceName: "book-service",
			Tags: []label.KeyValue{
				label.String("exporter", "jaeger"),
				label.Float64("float", 312.23),
			},
		}),
		jaeger.WithSDK(&sdktrace.Config{DefaultSampler: sdktrace.AlwaysSample()}),
	)
	if err != nil {
		log.Fatal(err)
	}
	return flush
}
