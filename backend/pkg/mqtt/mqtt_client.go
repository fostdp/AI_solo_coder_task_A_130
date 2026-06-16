package mqtt

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"dujiangyan-system/pkg/models"
)

var Client mqtt.Client
var topicPrefix string

type AlertMessage struct {
	ID                  int       `json:"id"`
	AlertTime           time.Time `json:"alert_time"`
	AlertType           string    `json:"alert_type"`
	AlertLevel          string    `json:"alert_level"`
	StationID           string    `json:"station_id"`
	Message             string    `json:"message"`
	BedElevation        float64   `json:"bed_elevation"`
	WolongIronElevation float64   `json:"wolong_iron_elevation"`
	ExceededValue       float64   `json:"exceeded_value"`
	Timestamp           time.Time `json:"timestamp"`
}

func InitMQTT() error {
	broker := os.Getenv("MQTT_BROKER")
	clientID := os.Getenv("MQTT_CLIENT_ID")
	username := os.Getenv("MQTT_USERNAME")
	password := os.Getenv("MQTT_PASSWORD")
	topicPrefix = os.Getenv("MQTT_TOPIC_PREFIX")

	opts := mqtt.NewClientOptions()
	opts.AddBroker(broker)
	opts.SetClientID(clientID)
	opts.SetUsername(username)
	opts.SetPassword(password)
	opts.SetAutoReconnect(true)
	opts.SetMaxReconnectInterval(10 * time.Second)
	opts.SetConnectionLostHandler(func(c mqtt.Client, err error) {
		log.Printf("MQTT connection lost: %v", err)
	})
	opts.SetOnConnectHandler(func(c mqtt.Client) {
		log.Println("MQTT connected successfully")
	})

	Client = mqtt.NewClient(opts)

	token := Client.Connect()
	token.WaitTimeout(10 * time.Second)
	if token.Error() != nil {
		return fmt.Errorf("failed to connect to MQTT broker: %w", token.Error())
	}

	log.Printf("MQTT client initialized, broker: %s", broker)
	return nil
}

func CloseMQTT() {
	if Client != nil && Client.IsConnected() {
		Client.Disconnect(250)
		log.Println("MQTT connection closed")
	}
}

func PublishAlert(alert *models.Alert) error {
	if Client == nil || !Client.IsConnected() {
		return fmt.Errorf("MQTT client not connected")
	}

	msg := AlertMessage{
		ID:                  alert.ID,
		AlertTime:           alert.AlertTime,
		AlertType:           alert.AlertType,
		AlertLevel:          alert.AlertLevel,
		StationID:           alert.StationID,
		Message:             alert.Message,
		BedElevation:        alert.BedElevation,
		WolongIronElevation: alert.WolongIronElevation,
		ExceededValue:       alert.ExceededValue,
		Timestamp:           time.Now(),
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal alert message: %w", err)
	}

	topic := fmt.Sprintf("%s/%s/%s", topicPrefix, alert.AlertLevel, alert.StationID)

	token := Client.Publish(topic, 1, false, payload)
	token.WaitTimeout(5 * time.Second)
	if token.Error() != nil {
		return fmt.Errorf("failed to publish alert: %w", token.Error())
	}

	log.Printf("Alert published to MQTT topic: %s", topic)
	return nil
}

func StartAlertPublisher(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Alert publisher stopped")
			return
		case <-ticker.C:
			alerts, err := models.GetUnpublishedAlerts(ctx)
			if err != nil {
				log.Printf("Failed to get unpublished alerts: %v", err)
				continue
			}

			for _, alert := range alerts {
				if err := PublishAlert(&alert); err != nil {
					log.Printf("Failed to publish alert %d: %v", alert.ID, err)
					continue
				}

				if err := models.MarkAlertAsPublished(ctx, alert.ID); err != nil {
					log.Printf("Failed to mark alert %d as published: %v", alert.ID, err)
				}
			}
		}
	}
}

func PublishHydrologyData(data *models.HydrologyData) error {
	if Client == nil || !Client.IsConnected() {
		return fmt.Errorf("MQTT client not connected")
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal hydrology data: %w", err)
	}

	topic := fmt.Sprintf("dujiangyan/hydrology/%s", data.StationID)

	token := Client.Publish(topic, 0, false, payload)
	token.WaitTimeout(5 * time.Second)
	if token.Error() != nil {
		return fmt.Errorf("failed to publish hydrology data: %w", token.Error())
	}

	return nil
}
