package types

const (
	// Info level actions
	ActionServiceStarted    = "service_started"
	ActionDBConnected       = "db_connected"
	ActionRabbitMQConnected = "rabbitmq_connected"
	ActionWorkerRegistered  = "worker_registered"
	ActionGracefulShutdown  = "graceful_shutdown"

	// Debug level actions
	ActionWorkerStop              = "worker_stop"
	ActionHeartbeatSent           = "heartbeat_sent"
	ActionRequestReceived         = "request_received"
	ActionOrderReceived           = "order_received"
	ActionOrderPublished          = "order_published"
	ActionOrderProcessingStarted  = "order_processing_started"
	ActionOrderCompleted          = "order_completed"
	ActionNotificationReceived    = "notification_received"
	ActionRabbitConnectionClosed  = "rabbitmq_connection_closed"
	ActionRabbitConnectionClosing = "rabbitmq_connection_closing"
	ActionRabbitReconnect         = "rabbitmq_reconnect"

	// Error level actions
	ActionValidationFailed         = "validation_failed"
	ActionDBTransactionFailed      = "db_transaction_failed"
	ActionDBQueryFailed            = "db_query_failed"
	ActionRabbitMQPublishFailed    = "rabbitmq_publish_failed"
	ActionWorkerRegistrationFailed = "worker_registration_failed"
	ActionMessageProcessingFailed  = "message_processing_failed"
	ActionDBConnectionFailed       = "db_connection_failed"
	ActionRabbitConnectionFailed   = "rabbitmq_connection_failed"
	ActionOrderProccessingFailed   = "order_proccess_failed"
)
