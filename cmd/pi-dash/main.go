package main

import (
	"context"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/jakekeeys/pi-dash/internal/dashcamsvc"
	"github.com/jawher/mow.cli"
	"github.com/sirupsen/logrus"
	"github.com/stianeikeland/go-rpio"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

func main() {
	app := cli.App("Audi Pi", "")

	logLevel := app.String(cli.StringOpt{
		Name:   "log-Level",
		Desc:   "log level [debug|info|warn|error]",
		EnvVar: "LOG_LEVEL",
		Value:  "debug",
	})
	logFormat := app.String(cli.StringOpt{
		Name:   "log-format",
		Desc:   "log format [text|json]",
		EnvVar: "LOG_FORMAT",
		Value:  "text",
	})
	httpPort := app.Int(cli.IntOpt{
		Name:   "http-port",
		Desc:   "The port to listen on for HTTP connections",
		Value:  8080,
		EnvVar: "HTTP_PORT",
	})
	indicatorPin := app.Int(cli.IntOpt{
		Name:   "indicator-pin",
		Desc:   "The pin connected to the indicator LED",
		Value:  18,
		EnvVar: "INDICATOR_PIN",
	})
	monitorPin := app.Int(cli.IntOpt{
		Name:   "monitor-pin",
		Desc:   "The pin connected to the powerboost USB pin",
		Value:  17,
		EnvVar: "MONITOR_PIN",
	})
	recordingPath := app.String(cli.StringOpt{
		Name:   "recording-path",
		Desc:   "The path at which to store recordings",
		Value:  "./media/",
		EnvVar: "RECORDING_PATH",
	})
	diskUsageTarget := app.Int(cli.IntOpt{
		Name:   "disk-usage-target",
		Desc:   "The disk usage percentage at which footage will be rotated",
		Value:  80,
		EnvVar: "DISK_USAGE_TARGET",
	})

	app.Action = func() {
		configureLogger(*logLevel, *logFormat)

		err := rpio.Open()
		if err != nil {
			logrus.Panic(err)
		}
		defer rpio.Close()

		i := dashcamsvc.NewIndicator(uint8(*indicatorPin))
		defer i.Extinguish()
		dcsvc := dashcamsvc.NewDashCamService(i, *recordingPath)
		defer dcsvc.Quit()
		pm := dashcamsvc.NewPowerMonitor(uint8(*monitorPin), dcsvc)
		defer pm.Quit()
		dm := dashcamsvc.NewDiskMonitor(*recordingPath, float64(*diskUsageTarget))
		defer dm.Quit()

		router := mux.NewRouter()
		dashCamSubrouter := router.PathPrefix("/dashcam").Subrouter()
		dashCamSubrouter.Handle("/record", dashcamsvc.StartRecording(dcsvc))
		dashCamSubrouter.Handle("/stop", dashcamsvc.StopRecording(dcsvc))
		dashCamSubrouter.Handle("/still", dashcamsvc.GetStill(dcsvc))
		dashCamFileServer := http.StripPrefix("/dashcam/recordings/", http.FileServer(http.Dir(*recordingPath)))
		dashCamSubrouter.PathPrefix("/recordings/").Handler(dashCamFileServer)
		loggedRouter := loggingMiddleware(router)

		server := http.Server{
			Addr:    fmt.Sprintf(":%d", *httpPort),
			Handler: loggedRouter,
		}
		defer server.Shutdown(context.Background())

		go func() {
			server.ListenAndServe()
		}()

		go func() {
			dcsvc.Run()
		}()

		go func() {
			pm.Run()
		}()

		go func() {
			dm.Run()
		}()

		waitForShutdown()
	}
	if err := app.Run(os.Args); err != nil {
		logrus.Panic(err)
	}
}

func configureLogger(level string, format string) {
	l, err := logrus.ParseLevel(level)
	if err != nil {
		logrus.WithError(err).Panic("invalid log level")
	}
	logrus.SetLevel(l)

	format = strings.ToLower(format)
	if format != "text" && format != "json" {
		logrus.Panicf("invalid log format: %s", format)
	}
	if format == "json" {
		logrus.SetFormatter(&logrus.JSONFormatter{})
	}
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
		logrus.Infof("%s %s", r.Method, r.RequestURI)
	})
}

func waitForShutdown() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan
	logrus.Info("Shutting down")
}
