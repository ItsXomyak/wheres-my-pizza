package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"restaurant-system/internal/logger"
	"restaurant-system/internal/messaging"
	"restaurant-system/internal/models"
)

// Subscriber handles notification messages
type Subscriber struct {
	consumer *messaging.Consumer
	logger   *logger.Logger
	
	// Graceful shutdown
	shutdown chan os.Signal
	done     chan bool
}

// NewSubscriber creates a new notification subscriber
func NewSubscriber(consumer *messaging.Consumer, logger *logger.Logger) *Subscriber {
	return &Subscriber{
		consumer: consumer,
		logger:   logger,
		shutdown: make(chan os.Signal, 1),
		done:     make(chan bool, 1),
	}
}

// Start starts the notification subscriber
func (s *Subscriber) Start(ctx context.Context) error {
	requestID := logger.GenerateRequestID()
	
	// Set up graceful shutdown
	signal.Notify(s.shutdown, syscall.SIGINT, syscall.SIGTERM)
	
	s.logger.Info("service_started", "Notification subscriber started", requestID, nil)
	
	// Start message consumption
	go func() {
		if err := s.consumer.StartConsuming(ctx, s.handleNotification); err != nil {
			s.logger.Error("consumer_failed", "Notification consumer failed", requestID, err, nil)
		}
		s.done <- true
	}()
	
	// Wait for shutdown signal or consumer to finish
	select {
	case <-s.shutdown:
		s.logger.Info("graceful_shutdown", "Received shutdown signal", requestID, nil)
		return s.gracefulShutdown(ctx, requestID)
	case <-s.done:
		return nil
	}
}

// handleNotification processes incoming status update notifications
func (s *Subscriber) handleNotification(ctx context.Context, body []byte) error {
	requestID := logger.GenerateRequestID()
	
	// Parse status update message
	var statusUpdate models.StatusUpdateMessage
	if err := json.Unmarshal(body, &statusUpdate); err != nil {
		s.logger.Error("message_parsing_failed", "Failed to parse notification message", requestID, err, nil)
		return fmt.Errorf("failed to parse notification: %w", err)
	}
	
	s.logger.Debug("notification_received", "Received status update notification", requestID, map[string]interface{}{
		"order_number": statusUpdate.OrderNumber,
		"new_status":   statusUpdate.NewStatus,
		"changed_by":   statusUpdate.ChangedBy,
	})
	
	// Display human-readable notification
	s.displayNotification(&statusUpdate)
	
	return nil
}

// displayNotification displays a human-readable notification to console
func (s *Subscriber) displayNotification(statusUpdate *models.StatusUpdateMessage) {
	notification := s.formatNotification(statusUpdate)
	
	// Print to console (stdout)
	fmt.Println(notification)
	
	// Also log as structured data
	s.logger.Info("notification_displayed", "Notification displayed to user", "", map[string]interface{}{
		"order_number": statusUpdate.OrderNumber,
		"old_status":   statusUpdate.OldStatus,
		"new_status":   statusUpdate.NewStatus,
		"changed_by":   statusUpdate.ChangedBy,
		"timestamp":    statusUpdate.Timestamp.Format("2006-01-02 15:04:05"),
	})
}

// formatNotification creates a human-readable notification message
func (s *Subscriber) formatNotification(statusUpdate *models.StatusUpdateMessage) string {
	timestamp := statusUpdate.Timestamp.Format("2006-01-02 15:04:05")
	
	var message string
	switch statusUpdate.NewStatus {
	case "cooking":
		if statusUpdate.EstimatedCompletion != nil {
			estimatedTime := statusUpdate.EstimatedCompletion.Format("15:04:05")
			message = fmt.Sprintf(
				"🍳 [%s] Order %s is now being prepared by %s. Estimated completion: %s",
				timestamp,
				statusUpdate.OrderNumber,
				statusUpdate.ChangedBy,
				estimatedTime,
			)
		} else {
			message = fmt.Sprintf(
				"🍳 [%s] Order %s is now being prepared by %s.",
				timestamp,
				statusUpdate.OrderNumber,
				statusUpdate.ChangedBy,
			)
		}
	case "ready":
		message = fmt.Sprintf(
			"✅ [%s] Order %s is ready for pickup/delivery! Prepared by %s.",
			timestamp,
			statusUpdate.OrderNumber,
			statusUpdate.ChangedBy,
		)
	case "completed":
		message = fmt.Sprintf(
			"🎉 [%s] Order %s has been completed and delivered! Thank you for your business.",
			timestamp,
			statusUpdate.OrderNumber,
		)
	case "cancelled":
		message = fmt.Sprintf(
			"❌ [%s] Order %s has been cancelled.",
			timestamp,
			statusUpdate.OrderNumber,
		)
	default:
		message = fmt.Sprintf(
			"📋 [%s] Order %s status changed from '%s' to '%s' by %s.",
			timestamp,
			statusUpdate.OrderNumber,
			statusUpdate.OldStatus,
			statusUpdate.NewStatus,
			statusUpdate.ChangedBy,
		)
	}
	
	return message
}

// gracefulShutdown handles graceful shutdown of the subscriber
func (s *Subscriber) gracefulShutdown(ctx context.Context, requestID string) error {
	s.logger.Info("graceful_shutdown", "Starting graceful shutdown", requestID, nil)
	
	// Close consumer
	if s.consumer != nil {
		s.consumer.Close()
	}
	
	s.logger.Info("graceful_shutdown", "Graceful shutdown completed", requestID, nil)
	return nil
}
