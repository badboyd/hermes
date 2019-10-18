package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/sideshow/apns2"
	"github.com/sideshow/apns2/payload"
	"github.com/sideshow/apns2/token"
)

const (
	// GOOGLE stands for google
	GOOGLE = "google"
	// APPLE is apple
	APPLE = "apple"
	// WEB is web LOL
	WEB = "web"
	// SUCCESS is success, really ?
	SUCCESS = "success"
	// FAIL means failure
	FAIL = "fail"
	// INTERNALERROR means error that comes from network, etc
	INTERNALERROR = "INTERNAL_SERVER_ERROR"
	// HTTPCODE order
	HTTPCODE = 0
	// REASON order
	REASON = 1
	// CODE order (firebase code, or apns code)
	CODE = 2
	// DETAILS of push notification error
	DETAILS = 3

	// For getting the data from firebase error message
	// "http error status: 404; reason: app instance has been unregistered; code: registration-token-not-registered; details: Requested entity was not found."
	lenFirebaseErrPart1 = len("http error status: ")
	lenFirebaseErrPart2 = len("code: ")
	lenFirebaseErrPart3 = len("reason: ")
)

var (
	apns2Client *apns2.Client
)

type (
	// Notification stands for each push notification event
	Notification struct {
		Channel     string `json:"channel"` // google, apple
		ProviderID  string `json:"provider_id"`
		DeviceToken string `json:"device_token"`
		Data        struct {
			Message string                 `json:"message"`
			Data    map[string]interface{} `json:"data"`
		} `json:"data"`
	}

	pushNotificationError struct {
		Reason  string `mapstructure:"reason"`
		Message string `mapstructure:"message"`
		Status  string `mapstructure:"reason"`
		Code    string `mapstructure:"code"`
	}

	TopicPublish struct {
		Data      PubSubMessage `json:"data"`
		EventID   string        `json:"eventID"`
		Timestamp string        `json:"timestamp"`
		EventType string        `json:"eventType"`
		Resource  string        `json:"resource"`
	}

	PubSubMessage struct {
		Attributes map[string]string `json:"attributes"`
		Data       string            `json:"data"`
	}
)

func (e *pushNotificationError) Error() string {
	return e.Message
}

// PushNotification ....
func PushNotification(ctx context.Context, data TopicPublish) error {
	notification := Notification{}
	err := json.Unmarshal([]byte(data.Data.Data), &notification)
	if err != nil {
		log.Printf("Cannot unmarshal data [%s]: %v\n", string(data.Data.Data), err)
		return nil
	}

	pnErr := &pushNotificationError{}

	switch notification.Channel {
	case APPLE:
		pnErr = pushAppleNotificationMessage(ctx, &notification)
	case GOOGLE:
	case WEB:
	default:
		log.Printf("Unknown device: %+v\n", notification)
	}

	if pnErr != nil {
		log.Printf("Cannot push data to %s-%s: %+v\n", notification.Channel, notification.DeviceToken, pnErr)
	} else {
		log.Printf("Push data successful to %s-%s\n", notification.Channel, notification.DeviceToken)
	}

	return nil
}

func getPushNotificationSource(n *Notification) string {
	for k, v := range n.Data.Data {
		if k == "type" {
			return fmt.Sprint(v)
		}
	}
	return "unknown"
}

func formatApns2Error(r *apns2.Response, apnsErr error) *pushNotificationError {
	if apnsErr != nil {
		return &pushNotificationError{
			Code:    "500",
			Status:  "fail",
			Reason:  INTERNALERROR,
			Message: apnsErr.Error(),
		}
	}

	if r.Sent() {
		return nil
	}

	return &pushNotificationError{
		Status:  "fail",
		Code:    fmt.Sprint(r.StatusCode),
		Reason:  r.Reason,
		Message: r.ApnsID,
	}
}

func pushAppleNotificationMessage(ctx context.Context, n *Notification) *pushNotificationError {
	notificationPayload := payload.NewPayload().
		Alert(n.Data.Message).
		Badge(1).Sound("default")

	notificationPayload.Custom("data", n.Data.Data)

	notification := &apns2.Notification{
		DeviceToken: n.DeviceToken,
		Topic:       "vn.chotot.iosapp",
		Payload:     notificationPayload,
	}

	log.Printf("Apple message: %+v\n", *notification)

	return formatApns2Error(apns2Client.PushWithContext(ctx, notification))
}

func initApns2Client() {
	iosCertPath := "./chotot-apns.p8"

	authKey, err := token.AuthKeyFromFile(iosCertPath)
	if err != nil {
		log.Fatalf("error read IOS JWT: %v\n", err)
	}

	token := &token.Token{
		AuthKey: authKey,
		KeyID:   "JQSPP5N79K",
		TeamID:  "MJ7856Z5Y6",
	}

	apns2Client = apns2.NewTokenClient(token)
	apns2Client = apns2Client.Production()
}

func init() {
	initApns2Client()
}
