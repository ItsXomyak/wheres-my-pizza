package types

type ServiceMode string

const (
	ModeOrder                  ServiceMode = "order-service"
	ModeKitchenWorker          ServiceMode = "kitchen-worker"
	ModeTracking               ServiceMode = "tracking-service"
	ModeNotificationSubscriber ServiceMode = "notification-subscriber"
)
