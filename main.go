package main

import (
	"mongo_fetch/config"
	"mongo_fetch/service"
	"os"
	"os/signal"
	"syscall"
)


func main() {
    config := config.LoadConfig()
    
    // Start service monitor which will launch non-running services in goroutines
    serviceMonitor := service.NewServiceMonitorService(config)
    
    // Run the service monitor in an infinite loop, exit gracefully on Ctrl+C (SIGINT)
    c := make(chan struct{})
    sigs := make(chan os.Signal, 1)
    signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
    go func() {
        <-sigs
        close(c)
    }()
    for {
        select {
        case <-c:
            return
        default:
            serviceMonitor.Start()
        }
    }
}


