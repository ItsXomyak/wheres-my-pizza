package rabbit

import (
	"fmt"

	"wheres-my-pizza/internal/domain/models"
	"wheres-my-pizza/pkg/rabbit"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Creates queues for each type of order and binds them to the given exchange.
// Also creates Dead Letter Exchange(DLX) and alongside with DLQ for each orderType and bind each the queue to that DLQ.
func InitQueuesForOrderTypes(client *rabbit.RabbitMQ, exchange string, orderTypes []string) error {
	// > Dead Letter Queue (DLQ) is a specialized queue that stores messages that cannot be delivered or processed by
	// their intended queue. It acts as a safety net, preventing failed messages from being lost and allowing for
	// inspection, troubleshooting, and potential reprocessing.
	// Creating Dead Letter exchange
	dlxExchange := "dlx_exchange"
	err := client.Channel.ExchangeDeclare(
		dlxExchange,
		"topic",
		true,  // durable
		false, // autoDelete
		false, // internal
		false, // noWait
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to declare DLX exchange: %w", err)
	}

	for _, ot := range orderTypes {
		// main order type queue name
		queueName := getQueueByOrderType(ot)

		// DLQ name
		deadLQueue := getDLQKeyForQueue(queueName)

		// arguments for the queue
		args := amqp.Table{
			"x-dead-letter-exchange":    dlxExchange,
			"x-dead-letter-routing-key": deadLQueue,
		}

		// Creating new queue
		_, err := client.Channel.QueueDeclare(
			queueName,
			true,
			false,
			false,
			false,
			args,
		)
		if err != nil {
			return fmt.Errorf("failed to declare queue %s: %w", queueName, err)
		}

		// Binding queue to the exchange
		bindingKey := getRoutingkeyByOrderType(ot)
		if err := client.Channel.QueueBind(
			queueName,
			bindingKey,
			exchange,
			false,
			nil,
		); err != nil {
			return fmt.Errorf("failed to bind queue %s: %w", queueName, err)
		}

		// DLQ queue
		_, err = client.Channel.QueueDeclare(
			deadLQueue,
			true,  // durable
			false, // autoDelete
			false, // exclusive
			false, // noWait
			nil,
		)
		if err != nil {
			return fmt.Errorf("failed to declare DLQ: %w", err)
		}

		// binding DLQ to the DLX
		if err := client.Channel.QueueBind(
			deadLQueue,
			getDLQRoutingKeyQueueName(queueName),
			dlxExchange,
			false,
			nil,
		); err != nil {
			return fmt.Errorf("failed to bind DLQ: %w", err)
		}
	}

	return nil
}

func getQueueByOrderType(ot string) string {
	return fmt.Sprintf("kitchen_%s_queue", ot)
}

func getRoutingkeyByOrderType(ot string) string {
	return fmt.Sprintf("kitchen.%s.*", ot)
}

// Create the routing key
func createOrderPublishedKey(order *models.CreateOrder) string {
	return fmt.Sprintf("kitchen.%s.%d", order.Type, order.Priority)
}

func getDLQKeyForQueue(queueName string) string {
	return fmt.Sprintf("dlq.%s", queueName)
}

func getDLQRoutingKeyQueueName(queueName string) string {
	return fmt.Sprintf("dlq.%s", queueName)
}
