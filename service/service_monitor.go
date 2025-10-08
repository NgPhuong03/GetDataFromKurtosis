package service

import (
	"context"
	"log"
	"sync"
	"time"

	"mongo_fetch/config"
	"mongo_fetch/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type ServiceMonitorService struct {
	config *config.Config
	collName string
}

func NewServiceMonitorService(config *config.Config) *ServiceMonitorService {
	return &ServiceMonitorService{config: config, collName: "service_monitor"}
}

func (s *ServiceMonitorService) Start() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := getMongoClient(ctx)
	if err != nil {
		log.Fatalf("failed to connect mongo: %v", err)
	}

	coll := client.Database(s.config.Mongo.DBName).Collection(s.collName)

	services, err := getAllServices(ctx, coll)
	if err != nil {
		log.Fatalf("failed to get all services: %v", err)
	}
	log.Println("Services", services)

	var wg sync.WaitGroup
	runningCount := 0
	maxGoroutines := 4

	for _, svc := range services {
		if svc.IsRunning {
			log.Println("Service is running", svc.Name)
			continue
		}

		log.Println("Service is not running", svc.Name)
		
		if runningCount >= maxGoroutines {
			log.Println("Maximum goroutines reached, skipping service", svc.Name)
			continue
		}

		wg.Add(1)
		go func(service model.ServiceMonitor) {
			defer wg.Done()
			log.Printf("Starting service %s in goroutine", service.Name)
			
			// Start the appropriate service based on name
			switch service.Name {
			case "epochs":
				epochService := NewEpochService(s.config)
				service.IsRunning = true
				_, err := coll.UpdateOne(ctx, bson.M{"name": service.Name}, bson.M{"$set": bson.M{"is_running": true}})
				if err != nil {
					log.Fatalf("failed to update service: %v", err)
				}
				epochService.Start()
				service.IsRunning = false
				_, err = coll.UpdateOne(ctx, bson.M{"name": service.Name}, bson.M{"$set": bson.M{"is_running": false}})
				if err != nil {
					log.Fatalf("failed to update service: %v", err)
				}
			case "validators":
				validatorsService := NewValidatorsService(s.config)
				service.IsRunning = true
				_, err := coll.UpdateOne(ctx, bson.M{"name": service.Name}, bson.M{"$set": bson.M{"is_running": true}})
				if err != nil {
					log.Fatalf("failed to update service: %v", err)
				}
				validatorsService.Start()
				service.IsRunning = false
				_, err = coll.UpdateOne(ctx, bson.M{"name": service.Name}, bson.M{"$set": bson.M{"is_running": false}})
				if err != nil {
					log.Fatalf("failed to update service: %v", err)
				}
			case "slots":
				slotsService := NewSlotService(s.config)
				service.IsRunning = true
				_, err := coll.UpdateOne(ctx, bson.M{"name": service.Name}, bson.M{"$set": bson.M{"is_running": true}})
				if err != nil {
					log.Fatalf("failed to update service: %v", err)
				}
				slotsService.Start()
				service.IsRunning = false
				_, err = coll.UpdateOne(ctx, bson.M{"name": service.Name}, bson.M{"$set": bson.M{"is_running": false}})
				if err != nil {
					log.Fatalf("failed to update service: %v", err)
				}
			default:
				log.Printf("Unknown service type: %s", service.Name)
			}
		}(svc)
		
		runningCount++
	}

	// Wait for all goroutines to complete
	wg.Wait()
	log.Println("All non-running services have been started")
}

func getAllServices(ctx context.Context, coll *mongo.Collection) ([]model.ServiceMonitor, error) {
	cursor, err := coll.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var services []model.ServiceMonitor
	for cursor.Next(ctx) {
		var service model.ServiceMonitor
		if err := cursor.Decode(&service); err != nil {
			return nil, err
		}
		services = append(services, service)
	}
	return services, nil
}

func (s *ServiceMonitorService) SetAllFalse() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := getMongoClient(ctx)
	if err != nil {
		log.Fatalf("failed to connect mongo: %v", err)
	}

	coll := client.Database(s.config.Mongo.DBName).Collection(s.collName)
	_, err = coll.UpdateMany(ctx, bson.M{}, bson.M{"$set": bson.M{"is_running": false}})
	if err != nil {
		log.Fatalf("failed to update services: %v", err)
	}
	log.Println("All services set to false")
}