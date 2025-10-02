package model

type ServiceMonitor struct {
	ID        string `json:"id" bson:"_id"`
	Name      string `json:"name" bson:"name"`
	IsRunning bool   `json:"is_running" bson:"is_running"`
}