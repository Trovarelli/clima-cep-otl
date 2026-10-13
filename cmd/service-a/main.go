package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"clima-cep-otel/internal/servicea"
	"clima-cep-otel/internal/telemetry"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	serviceBURL := os.Getenv("SERVICE_B_URL")
	if serviceBURL == "" {
		serviceBURL = "http://service-b:8081"
	}

	otelCollectorAddr := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if otelCollectorAddr == "" {
		otelCollectorAddr = os.Getenv("OTEL_COLLECTOR_ADDR")
	}
	if otelCollectorAddr == "" {
		otelCollectorAddr = "otel-collector:4317"
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tracer, shutdownTracer, err := telemetry.InitTracer(ctx, "service-a", otelCollectorAddr)
	if err != nil {
		logger.Error("falha ao inicializar tracer", "erro", err)
	} else {
		defer func() {
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer shutdownCancel()
			if err := shutdownTracer(shutdownCtx); err != nil {
				logger.Error("erro ao finalizar tracer", "erro", err)
			}
		}()
	}

	handler := servicea.NewHandler(serviceBURL, tracer)
	otelHandler := otelhttp.NewHandler(handler.Routes(), "service-a-http")

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           otelHandler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		logger.Info("Serviço A iniciado", "porta", port, "service_b", serviceBURL)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("falha ao iniciar servidor HTTP do Serviço A", "erro", err)
			os.Exit(1)
		}
	}()

	<-stop
	logger.Info("desligando Serviço A graciosamente...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("erro ao desligar servidor", "erro", err)
	}
	logger.Info("Serviço A finalizado com sucesso.")
}
