// backend/internal/adapters/out/cloudtasks/order_dispatch_notification_queue.go
package cloudtasks

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	cloudtasksv2 "cloud.google.com/go/cloudtasks/apiv2"
	taskspb "cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	uc "narratives/internal/application/usecase"
	orderdom "narratives/internal/domain/order"
)

const (
	envOrderDispatchCloudTasksProjectID      = "CLOUD_TASKS_PROJECT_ID"
	envOrderDispatchCloudTasksLocation       = "CLOUD_TASKS_LOCATION"
	envOrderDispatchCloudTasksQueueID        = "CLOUD_TASKS_QUEUE_ID"
	envOrderDispatchInternalBaseURL          = "INTERNAL_BASE_URL"
	envOrderDispatchCloudTasksServiceAccount = "CLOUD_TASKS_SERVICE_ACCOUNT"
	envOrderDispatchCloudTasksAudience       = "CLOUD_TASKS_AUDIENCE"

	defaultOrderDispatchNotificationTaskPath = "/internal/order-dispatch-notifications/process"

	defaultOrderDispatchNotificationDispatchDeadline = 10 * time.Minute
)

type OrderDispatchNotificationQueueConfig struct {
	ProjectID           string
	Location            string
	QueueID             string
	InternalBaseURL     string
	ServiceAccountEmail string
	Audience            string
}

type orderDispatchNotificationTaskPayload struct {
	DeliveryID string `json:"deliveryId"`
}

type OrderDispatchNotificationQueue struct {
	client *cloudtasksv2.Client

	queuePath string
	targetURL string

	serviceAccountEmail string
	audience            string

	dispatchDeadline time.Duration
	now              func() time.Time
}

var _ uc.OrderDispatchNotificationQueuePort = (*OrderDispatchNotificationQueue)(nil)

func NewOrderDispatchNotificationQueueFromEnv(
	ctx context.Context,
) (*OrderDispatchNotificationQueue, error) {
	config := OrderDispatchNotificationQueueConfig{
		ProjectID: firstNonEmptyOrderDispatchNotificationEnv(
			envOrderDispatchCloudTasksProjectID,
			"GCP_PROJECT_ID",
			"GOOGLE_CLOUD_PROJECT",
		),
		Location: firstNonEmptyOrderDispatchNotificationEnv(
			envOrderDispatchCloudTasksLocation,
		),
		QueueID: firstNonEmptyOrderDispatchNotificationEnv(
			envOrderDispatchCloudTasksQueueID,
		),
		InternalBaseURL: firstNonEmptyOrderDispatchNotificationEnv(
			envOrderDispatchInternalBaseURL,
		),
		ServiceAccountEmail: firstNonEmptyOrderDispatchNotificationEnv(
			envOrderDispatchCloudTasksServiceAccount,
		),
		Audience: firstNonEmptyOrderDispatchNotificationEnv(
			envOrderDispatchCloudTasksAudience,
		),
	}

	return NewOrderDispatchNotificationQueue(
		ctx,
		config,
	)
}

func NewOrderDispatchNotificationQueue(
	ctx context.Context,
	config OrderDispatchNotificationQueueConfig,
) (*OrderDispatchNotificationQueue, error) {
	normalizedConfig, err := normalizeOrderDispatchNotificationQueueConfig(
		config,
	)
	if err != nil {
		return nil, err
	}

	if ctx == nil {
		ctx = context.Background()
	}

	client, err := cloudtasksv2.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"create order dispatch notification Cloud Tasks client: %w",
			err,
		)
	}

	queuePath := fmt.Sprintf(
		"projects/%s/locations/%s/queues/%s",
		normalizedConfig.ProjectID,
		normalizedConfig.Location,
		normalizedConfig.QueueID,
	)

	targetURL := strings.TrimRight(
		normalizedConfig.InternalBaseURL,
		"/",
	) + defaultOrderDispatchNotificationTaskPath

	return &OrderDispatchNotificationQueue{
		client:              client,
		queuePath:           queuePath,
		targetURL:           targetURL,
		serviceAccountEmail: normalizedConfig.ServiceAccountEmail,
		audience:            normalizedConfig.Audience,
		dispatchDeadline:    defaultOrderDispatchNotificationDispatchDeadline,
		now:                 time.Now,
	}, nil
}

func (q *OrderDispatchNotificationQueue) EnqueueOrderDispatchNotification(
	ctx context.Context,
	delivery orderdom.DispatchNotificationDelivery,
) error {
	if q == nil {
		return errors.New(
			"order dispatch notification queue is nil",
		)
	}

	if q.client == nil {
		return errors.New(
			"order dispatch notification Cloud Tasks client is nil",
		)
	}

	if ctx == nil {
		ctx = context.Background()
	}

	normalizedDelivery, err := delivery.Normalize()
	if err != nil {
		return fmt.Errorf(
			"normalize order dispatch notification before enqueue: %w",
			err,
		)
	}

	if normalizedDelivery.IsTerminal() {
		return orderdom.ErrDispatchNotificationNotClaimable
	}

	if normalizedDelivery.AttemptCount >= normalizedDelivery.MaxAttempts {
		return orderdom.ErrDispatchNotificationAttemptLimit
	}

	deliveryID := strings.TrimSpace(normalizedDelivery.ID)
	if deliveryID == "" {
		return orderdom.ErrDispatchNotificationDeliveryIDRequired
	}

	payload, err := json.Marshal(
		orderDispatchNotificationTaskPayload{
			DeliveryID: deliveryID,
		},
	)
	if err != nil {
		return fmt.Errorf(
			"marshal order dispatch notification task payload: %w",
			err,
		)
	}

	nextAttemptNumber := normalizedDelivery.AttemptCount + 1

	taskID := buildOrderDispatchNotificationTaskID(
		deliveryID,
		nextAttemptNumber,
	)

	taskName := fmt.Sprintf(
		"%s/tasks/%s",
		q.queuePath,
		taskID,
	)

	httpRequest := &taskspb.HttpRequest{
		HttpMethod: taskspb.HttpMethod_POST,
		Url:        q.targetURL,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: payload,
		AuthorizationHeader: &taskspb.HttpRequest_OidcToken{
			OidcToken: &taskspb.OidcToken{
				ServiceAccountEmail: q.serviceAccountEmail,
				Audience:            q.audience,
			},
		},
	}

	task := &taskspb.Task{
		Name: taskName,
		MessageType: &taskspb.Task_HttpRequest{
			HttpRequest: httpRequest,
		},
		DispatchDeadline: durationpb.New(
			q.normalizedDispatchDeadline(),
		),
	}

	now := q.currentTime()
	scheduleAt := orderDispatchNotificationScheduleTime(
		normalizedDelivery,
	)

	if !scheduleAt.IsZero() && scheduleAt.After(now) {
		task.ScheduleTime = timestamppb.New(scheduleAt)

		if err := task.ScheduleTime.CheckValid(); err != nil {
			return fmt.Errorf(
				"invalid order dispatch notification task schedule time: %w",
				err,
			)
		}
	}

	_, err = q.client.CreateTask(
		ctx,
		&taskspb.CreateTaskRequest{
			Parent: q.queuePath,
			Task:   task,
		},
	)
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return nil
		}

		return fmt.Errorf(
			"create order dispatch notification task deliveryId=%q attempt=%d: %w",
			deliveryID,
			nextAttemptNumber,
			err,
		)
	}

	return nil
}

func (q *OrderDispatchNotificationQueue) Close() error {
	if q == nil || q.client == nil {
		return nil
	}

	if err := q.client.Close(); err != nil {
		return fmt.Errorf(
			"close order dispatch notification Cloud Tasks client: %w",
			err,
		)
	}

	q.client = nil
	return nil
}

func normalizeOrderDispatchNotificationQueueConfig(
	config OrderDispatchNotificationQueueConfig,
) (OrderDispatchNotificationQueueConfig, error) {
	config.ProjectID = strings.TrimSpace(config.ProjectID)
	config.Location = strings.TrimSpace(config.Location)
	config.QueueID = strings.TrimSpace(config.QueueID)
	config.InternalBaseURL = strings.TrimRight(
		strings.TrimSpace(config.InternalBaseURL),
		"/",
	)
	config.ServiceAccountEmail = strings.TrimSpace(
		config.ServiceAccountEmail,
	)
	config.Audience = strings.TrimRight(
		strings.TrimSpace(config.Audience),
		"/",
	)

	if config.ProjectID == "" {
		return OrderDispatchNotificationQueueConfig{},
			errors.New(
				"CLOUD_TASKS_PROJECT_ID is empty",
			)
	}

	if config.Location == "" {
		return OrderDispatchNotificationQueueConfig{},
			errors.New(
				"CLOUD_TASKS_LOCATION is empty",
			)
	}

	if config.QueueID == "" {
		return OrderDispatchNotificationQueueConfig{},
			errors.New(
				"CLOUD_TASKS_QUEUE_ID is empty",
			)
	}

	if config.InternalBaseURL == "" {
		return OrderDispatchNotificationQueueConfig{},
			errors.New(
				"INTERNAL_BASE_URL is empty",
			)
	}

	if err := validateOrderDispatchNotificationURL(
		config.InternalBaseURL,
	); err != nil {
		return OrderDispatchNotificationQueueConfig{},
			fmt.Errorf(
				"invalid INTERNAL_BASE_URL: %w",
				err,
			)
	}

	if config.ServiceAccountEmail == "" {
		return OrderDispatchNotificationQueueConfig{},
			errors.New(
				"CLOUD_TASKS_SERVICE_ACCOUNT is empty",
			)
	}

	if config.Audience == "" {
		config.Audience = config.InternalBaseURL
	}

	if err := validateOrderDispatchNotificationURL(
		config.Audience,
	); err != nil {
		return OrderDispatchNotificationQueueConfig{},
			fmt.Errorf(
				"invalid CLOUD_TASKS_AUDIENCE: %w",
				err,
			)
	}

	return config, nil
}

func validateOrderDispatchNotificationURL(
	value string,
) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return errors.New("URL is empty")
	}

	parsed, err := url.ParseRequestURI(value)
	if err != nil {
		return fmt.Errorf(
			"parse URL: %w",
			err,
		)
	}

	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
	default:
		return errors.New(
			"URL must use http or https",
		)
	}

	if strings.TrimSpace(parsed.Host) == "" {
		return errors.New(
			"URL host is empty",
		)
	}

	if parsed.RawQuery != "" {
		return errors.New(
			"URL must not contain query parameters",
		)
	}

	if parsed.Fragment != "" {
		return errors.New(
			"URL must not contain fragment",
		)
	}

	return nil
}

func orderDispatchNotificationScheduleTime(
	delivery orderdom.DispatchNotificationDelivery,
) time.Time {
	switch delivery.Status {
	case orderdom.DispatchNotificationStatusPending,
		orderdom.DispatchNotificationStatusRetryableFailed:
		if delivery.NextAttemptAt != nil &&
			!delivery.NextAttemptAt.IsZero() {
			return delivery.NextAttemptAt.UTC()
		}

	case orderdom.DispatchNotificationStatusProcessing:
		if delivery.ProcessingUntil != nil &&
			!delivery.ProcessingUntil.IsZero() {
			return delivery.ProcessingUntil.UTC()
		}
	}

	return time.Time{}
}

func buildOrderDispatchNotificationTaskID(
	deliveryID string,
	attemptNumber int,
) string {
	deliveryID = strings.TrimSpace(deliveryID)

	digest := sha256.Sum256(
		[]byte(deliveryID),
	)

	deliveryHash := hex.EncodeToString(
		digest[:16],
	)

	if attemptNumber < 1 {
		attemptNumber = 1
	}

	return fmt.Sprintf(
		"order-dispatch-%s-attempt-%d",
		deliveryHash,
		attemptNumber,
	)
}

func firstNonEmptyOrderDispatchNotificationEnv(
	keys ...string,
) string {
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}

		value := strings.TrimSpace(
			os.Getenv(key),
		)
		if value != "" {
			return value
		}
	}

	return ""
}

func (q *OrderDispatchNotificationQueue) currentTime() time.Time {
	if q != nil && q.now != nil {
		return q.now().UTC()
	}

	return time.Now().UTC()
}

func (q *OrderDispatchNotificationQueue) normalizedDispatchDeadline() time.Duration {
	if q == nil || q.dispatchDeadline <= 0 {
		return defaultOrderDispatchNotificationDispatchDeadline
	}

	return q.dispatchDeadline
}
