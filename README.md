# Learning about OpenTelemetry

This repository contains a [OpenTelemetry](https://opentelemetry.io/) example about how to instrument Go based services with tracing.
The example consists of a HTTP server and HTTP client. The server provides an API with the following URLS:

 - `/books`: returns a json response / 200
 - `/error`: returns a internal server error / 500

The client periodically calls booth API endpoints. Currently Jaeger is used as a tracing exporter to visualize the created traces by the example service.opentelemetry


## Quick Start

If you want to see the example service in action you can run it via docker-compose:
```bash
docker-compose up --build
```

After successfully building booth of the services you can visit the Jaeger UI on `http://localhost:16686/`

## Resources

- [opentelemetry.io](https://opentelemetry.io/)
- [GopherCon 2020: Ted Young - The Fundamentals of OpenTelemetry](https://youtu.be/X8w4yCmAMos)
- [github.com/open-telemetry/opentelemetry-go](https://github.com/open-telemetry/opentelemetry-go)
- [github.com/open-telemetry/opentelemetry-go-contrib](https://github.com/open-telemetry/opentelemetry-go-contrib)
- [github.com/jaegertracing/jaeger](https://github.com/jaegertracing/jaeger)
