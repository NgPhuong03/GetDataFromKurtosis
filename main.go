package main

import (
	"context"
	"mongo_fetch/config"
	"mongo_fetch/service"
	"os"
	"os/signal"
	"syscall"
	"time"
)


func main() {
    config := config.LoadConfig()
    
    // Start service monitor which will launch non-running services in goroutines
    serviceMonitor := service.NewServiceMonitorService(config)
    serviceMonitor.SetAllFalse()

    // Run the service monitor in an infinite loop, exit gracefully on Ctrl+C (SIGINT)
    c := make(chan struct{})
    sigs := make(chan os.Signal, 1)
    signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
    go func() {
        <-sigs
        // graceful shutdown mongo singleton with a short timeout
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()
        _ = service.DisconnectMongo(ctx)
        close(c)
    }()
    for {
        select {
		case <-c:
			serviceMonitor.SetAllFalse()
            return
        default:
            serviceMonitor.Start()
        }
    }
}


