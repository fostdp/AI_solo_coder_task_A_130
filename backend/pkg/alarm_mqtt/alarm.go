package alarm_mqtt

import (
	"context"
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"
	"dujiangyan-system/pkg/models"
	"dujiangyan-system/pkg/msg"
)

type AlertLevel string

const (
	LevelCritical AlertLevel = "CRITICAL"
	LevelWarning  AlertLevel = "WARNING"
	LevelNotice   AlertLevel = "NOTICE"
)

type AlertConfig struct {
	CriticalThreshold float64 `json:"critical_threshold"`
	WarningThreshold  float64 `json:"warning_threshold"`
	PublishInterval   int     `json:"publish_interval_seconds"`
	MQTTPrefix        string  `json:"mqtt_prefix"`
}

type WolongIronRef struct {
	Name          string
	Location      string
	Elevation     float64
	StationID     string
}

type AlertEvaluator struct {
	wolongIronMap map[string][]WolongIronRef
	config        AlertConfig
}

func NewAlertEvaluator() *AlertEvaluator {
	return &AlertEvaluator{
		wolongIronMap: map[string][]WolongIronRef{
			"NEIJ-001": {{Name: "卧铁1", Location: "内江河床", Elevation: 726.24, StationID: "NEIJ-001"}},
			"NEIJ-002": {{Name: "卧铁2", Location: "内江河床", Elevation: 726.18, StationID: "NEIJ-002"}},
			"NEIJ-003": {{Name: "卧铁3", Location: "内江河床", Elevation: 726.12, StationID: "NEIJ-003"}},
			"WAIJ-001": {{Name: "卧铁4", Location: "外江河床", Elevation: 726.06, StationID: "WAIJ-001"}},
			"WAIJ-002": {{Name: "卧铁4", Location: "外江河床", Elevation: 726.06, StationID: "WAIJ-002"}},
			"FSSY-001": {{Name: "卧铁1", Location: "飞沙堰", Elevation: 726.24, StationID: "FSSY-001"}},
			"FSSY-002": {{Name: "卧铁2", Location: "飞沙堰", Elevation: 726.18, StationID: "FSSY-002"}},
			"RJK-001":  {{Name: "卧铁3", Location: "人字堤", Elevation: 726.12, StationID: "RJK-001"}},
		},
		config: AlertConfig{
			CriticalThreshold: 1.0,
			WarningThreshold:  0.5,
			PublishInterval:   30,
			MQTTPrefix:        "dujiangyan/alerts",
		},
	}
}

func (ae *AlertEvaluator) Evaluate(data *models.HydrologyData) *models.Alert {
	irons, ok := ae.wolongIronMap[data.StationID]
	if !ok || len(irons) == 0 {
		return nil
	}

	var closestIron WolongIronRef
	minDist := math.MaxFloat64
	for _, iron := range irons {
		dist := math.Abs(data.BedElevation - iron.Elevation)
		if dist < minDist {
			minDist = dist
			closestIron = iron
		}
	}

	if data.BedElevation <= closestIron.Elevation {
		return nil
	}

	exceedance := data.BedElevation - closestIron.Elevation
	var level string
	switch {
	case exceedance > ae.config.CriticalThreshold:
		level = string(LevelCritical)
	case exceedance > ae.config.WarningThreshold:
		level = string(LevelWarning)
	default:
		level = string(LevelNotice)
	}

	return &models.Alert{
		AlertTime:           time.Now(),
		AlertType:           "bed_elevation_exceeded",
		AlertLevel:          level,
		StationID:           data.StationID,
		Message:             fmt.Sprintf("河床高程 %.3fm 超过卧铁 %s 高程 %.3fm (超出 %.3fm)", data.BedElevation, closestIron.Name, closestIron.Elevation, exceedance),
		BedElevation:        data.BedElevation,
		WolongIronElevation: closestIron.Elevation,
		ExceededValue:       exceedance,
		Acknowledged:        false,
		MqttPublished:       false,
	}
}

type MQTTClient struct {
	client pahomqtt.Client
	prefix string
	ctx    context.Context
}

func NewMQTTClient(ctx context.Context, broker, clientID, prefix string) *MQTTClient {
	if prefix == "" {
		prefix = "dujiangyan/alerts"
	}

	opts := pahomqtt.NewClientOptions()
	opts.AddBroker(broker)
	opts.SetClientID(clientID)
	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(5 * time.Second)
	opts.SetMaxReconnectInterval(30 * time.Second)

	client := pahomqtt.NewClient(opts)
	token := client.Connect()
	token.Wait()
	if token.Error() != nil {
		log.Printf("MQTT connect warning: %v (will retry)", token.Error())
	}

	return &MQTTClient{client: client, prefix: prefix, ctx: ctx}
}

func (mc *MQTTClient) PublishAlert(alert *models.Alert) error {
	if mc == nil || mc.client == nil {
		return fmt.Errorf("MQTT client not initialized")
	}

	if !mc.client.IsConnected() {
		return fmt.Errorf("MQTT client not connected")
	}

	topic := fmt.Sprintf("%s/%s/%s", mc.prefix, alert.AlertLevel, alert.StationID)

	payload := fmt.Sprintf(
		`{"time":"%s","station_id":"%s","level":"%s","type":"%s","message":"%s","elevation":%.3f,"threshold":%.3f,"exceedance":%.3f}`,
		alert.Time.Format(time.RFC3339), alert.StationID, alert.AlertLevel,
		alert.AlertType, alert.Message, alert.CurrentElevation,
		alert.Threshold, alert.Exceedance,
	)

	token := mc.client.Publish(topic, 1, false, payload)
	token.Wait()
	return token.Error()
}

func (mc *MQTTClient) PublishHydrologyData(data *models.HydrologyData) error {
	if mc == nil || mc.client == nil || !mc.client.IsConnected() {
		return fmt.Errorf("MQTT not available")
	}

	topic := fmt.Sprintf("dujiangyan/hydrology/%s", data.StationID)
	payload := fmt.Sprintf(
		`{"time":"%s","station_id":"%s","water_level":%.3f,"flow_rate":%.2f,"sediment":%.4f,"bed_elevation":%.3f}`,
		data.Time.Format(time.RFC3339), data.StationID,
		data.WaterLevel, data.FlowRate, data.SedimentConcentration, data.BedElevation,
	)

	token := mc.client.Publish(topic, 0, false, payload)
	token.Wait()
	return token.Error()
}

type AlarmService struct {
	ctx       context.Context
	bus       <-chan msg.BusMessage
	evaluator *AlertEvaluator
	mqtt      *MQTTClient
	wsClients map[string]interface{}
	wsMu      sync.RWMutex
}

func NewAlarmService(
	ctx context.Context,
	bus <-chan msg.BusMessage,
	mqttBroker string,
) *AlarmService {
	var mqttClient *MQTTClient
	if mqttBroker != "" {
		mqttClient = NewMQTTClient(ctx, mqttBroker, "dujiangyan-alarm-service", "dujiangyan/alerts")
	}

	return &AlarmService{
		ctx:       ctx,
		bus:       bus,
		evaluator: NewAlertEvaluator(),
		mqtt:      mqttClient,
		wsClients: make(map[string]interface{}),
	}
}

func (as *AlarmService) Start() {
	go as.listenBus()
	go as.startUnpublishedAlertPublisher()
}

func (as *AlarmService) listenBus() {
	for {
		select {
		case <-as.ctx.Done():
			return
		case message := <-as.bus:
			switch message.Type {
			case msg.TypeHydrologyData:
				if payload, ok := message.Payload.(msg.HydrologyPayload); ok {
					as.processHydrologyData(payload.Data)
				}
			case msg.TypeAlert:
				if payload, ok := message.Payload.(msg.AlertPayload); ok {
					as.publishAlert(payload.Alert)
				}
			}
		}
	}
}

func (as *AlarmService) processHydrologyData(data *models.HydrologyData) {
	if as.mqtt != nil {
		if err := as.mqtt.PublishHydrologyData(data); err != nil {
			log.Printf("MQTT hydrology publish failed: %v", err)
		}
	}

	alert := as.evaluator.Evaluate(data)
	if alert != nil {
		as.publishAlert(alert)
	}
}

func (as *AlarmService) publishAlert(alert *models.Alert) {
	if as.mqtt != nil {
		if err := as.mqtt.PublishAlert(alert); err != nil {
			log.Printf("MQTT alert publish failed: %v", err)
		}
	}
}

func (as *AlarmService) startUnpublishedAlertPublisher() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-as.ctx.Done():
			return
		case <-ticker.C:
			if as.mqtt == nil {
				continue
			}
			alerts, err := models.GetUnpublishedAlerts(as.ctx)
			if err != nil {
				continue
			}
			for _, alert := range alerts {
				if err := as.mqtt.PublishAlert(&alert); err != nil {
					log.Printf("Failed to republish alert %d: %v", alert.ID, err)
					continue
				}
				models.MarkAlertAsPublished(as.ctx, alert.ID)
			}
		}
	}
}

func (as *AlarmService) HandleAcknowledgeAlert(alertID int64) error {
	return models.AcknowledgeAlert(as.ctx, int(alertID), "system")
}
