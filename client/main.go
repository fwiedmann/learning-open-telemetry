package main

import (
	"context"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"go.opentelemetry.io/otel/exporters/trace/jaeger"
	"go.opentelemetry.io/otel/label"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// It's recommended to create a tracer for each package. The name of the tracer will be set as span attribute: "otel.instrumentation_library.name"
// If you set the name e.g. as the package name it's easier to debug if smth. goes wrong.
var rootTrace = otel.Tracer("github.com/fwiedmann/open-telemetry/client")

func main() {
	flush := initTracer()
	defer flush()
	for {
		ctx, span := rootTrace.Start(context.Background(), "book-service-client-handler-valid")
		makeRequest(ctx, "http://book-server:8080/books")
		span.End()
		ctx, span = rootTrace.Start(context.Background(), "book-service-client-handler-invalid")
		makeRequest(ctx, "http://book-server:8080/error")
		span.End()
		time.Sleep(time.Second * 1)
	}

}

func makeRequest(ctx context.Context, URI string) {
	c := http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, URI, nil)

	res, err := c.Do(req)
	if err != nil {
		panic(err)
	}

	content, err := ioutil.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}
	if res.StatusCode < 300 {
		trace.SpanFromContext(ctx).AddEvent("books received", trace.WithAttributes(label.KeyValue{Key: "body", Value: label.StringValue(string(content))}))
	}
	res.Body.Close()
}

// initTracer creates a new trace provider instance and registers it as global trace provider.
func initTracer() func() {
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	// Create and install Jaeger export pipeline.
	flush, err := jaeger.InstallNewPipeline(
		jaeger.WithCollectorEndpoint("http://jaeger:14268/api/traces"),
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
