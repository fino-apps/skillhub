package main

import (
	"cmp"
	"context"
	"net"
	"os"

	"github.com/fino-io/finokit/config"
	"github.com/fino-io/finokit/logs"
	gokitmetrics "github.com/fino-io/gokit/metrics"
	nserver "github.com/fino-io/gokit/server"
	"github.com/fino-io/gokit/tracing"
	"github.com/fino-io/skillhub/backend/infra/nacos"
	"github.com/fino-io/skillhub/backend/internal/skillhub/api"
)

func loadConfig() (nserver.Config, error) {
	if err := nacos.LoadServiceConfig("skillhub-server"); err != nil {
		return nserver.Config{}, err
	}

	var cfg nserver.Config
	if err := config.ScanFrom(&cfg, "skillhubServer", "server"); err != nil {
		return nserver.Config{}, err
	}

	for _, address := range []struct {
		value       *string
		host        string
		defaultPort string
	}{
		{&cfg.HttpAddr, os.Getenv("SERVER_HTTP_HOST"), "31101"},
		{&cfg.GrpcAddr, "", "31102"},
		{&cfg.DebugAddr, "", "31103"},
	} {
		host, port, err := net.SplitHostPort(*address.value)
		if err != nil {
			return nserver.Config{}, err
		}
		*address.value = net.JoinHostPort(cmp.Or(address.host, host), cmp.Or(port, address.defaultPort))
	}

	if err := logs.NewConfig(); err != nil {
		return nserver.Config{}, err
	}

	return cfg, nil
}

func serve(ctx context.Context, cfg nserver.Config) error {
	tracer, shutdownTrace, err := tracing.New("skillhub-server")
	if err != nil {
		logs.Warnw("failed to initialize tracing", "error", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), nserver.DefaultShutdownTimeout)
		defer cancel()
		if err := shutdownTrace(shutdownCtx); err != nil {
			logs.Warnw("failed to shut down tracing", "error", err)
		}
	}()

	instrumentation, err := gokitmetrics.New("skillhub")
	if err != nil {
		logs.Warnw("failed to initialize metrics", "error", err)
		instrumentation = gokitmetrics.Disabled()
	}

	server, closeCatalog, err := openCatalog(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err := closeCatalog(); err != nil {
			logs.Errorw("failed to close infrastructure", "error", err)
		}
	}()

	return api.RunContext(ctx, cfg, server, tracer, instrumentation)
}
