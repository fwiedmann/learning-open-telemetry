package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"go.opentelemetry.io/otel/exporters/trace/jaeger"
	"go.opentelemetry.io/otel/label"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

var rootTrace = otel.Tracer("client-trace")

func main() {
	flush := initTracer()
	defer flush()

	ctx, span := rootTrace.Start(context.Background(), "book-service-client-handler")
	defer span.End()
	wg := sync.WaitGroup{}

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(iter int) {
			time.Sleep(time.Millisecond * time.Duration(iter))
			makeRequest(ctx)
			wg.Done()
		}(i)
	}
	wg.Wait()
}

func makeRequest(ctx context.Context) {
	c := http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost:8080/books", nil)

	res, err := c.Do(req)
	if err != nil {
		panic(err)
	}

	content, err := ioutil.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}
	res.Body.Close()

	fmt.Print(string(content))
}

// initTracer creates a new trace provider instance and registers it as global trace provider.
func initTracer() func() {
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	// Create and install Jaeger export pipeline.
	flush, err := jaeger.InstallNewPipeline(
		jaeger.WithCollectorEndpoint("http://localhost:14268/api/traces"),
		jaeger.WithProcess(jaeger.Process{
			ServiceName: "book-service-client",
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
