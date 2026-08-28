// backend/internal/adapters/out/cloudtasks/refund_completion_notification_queue.go
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

	applicationport "narratives/internal/application/port"
	refunddom "narratives/internal/domain/refund"
)

const (
	envRefundCompletionCloudTasksProjectID      = "CLOUD_TASKS_PROJECT_ID"
	envRefundCompletionCloudTasksLocation       = "CLOUD_TASKS_LOCATION"
	envRefundCompletionCloudTasksQueueID        = "CLOUD_TASKS_QUEUE_ID"
	envRefundCompletionInternalBaseURL          = "INTERNAL_BASE_URL"
	envRefundCompletionCloudTasksServiceAccount = "CLOUD_TASKS_SERVICE_ACCOUNT"
	envRefundCompletionCloudTasksAudience       = "CLOUD_TASKS_AUDIENCE"

	defaultRefundCompletionNotificationTaskPath = "/internal/refund-completion-notifications/process"

	defaultRefundCompletionNotificationDispatchDeadline = 10 * time.Minute
)

type RefundCompletionNotificationQueueConfig struct {
	ProjectID           string
	Location            string
	QueueID             string
	InternalBaseURL     string
	ServiceAccountEmail string
	Audience            string
}

type refundCompletionNotificationTaskPayload struct {
	DeliveryID string `json:"deliveryId"`
}

type RefundCompletionNotificationQueue struct {
	client *cloudtasksv2.Client

	queuePath string
	targetURL string

	serviceAccountEmail string
	audience            string

	dispatchDeadline time.Duration
	now              func() time.Time
}

var _ applicationport.RefundCompletionNotificationQueuePort = (*RefundCompletionNotificationQueue)(nil)

func NewRefundCompletionNotificationQueueFromEnv(
	ctx context.Context,
) (*RefundCompletionNotificationQueue, error) {
	config := RefundCompletionNotificationQueueConfig{
		ProjectID: firstNonEmptyRefundCompletionNotificationEnv(
			envRefundCompletionCloudTasksProjectID,
			"GCP_PROJECT_ID",
			"GOOGLE_CLOUD_PROJECT",
		),
		Location: firstNonEmptyRefundCompletionNotificationEnv(
			envRefundCompletionCloudTasksLocation,
		),
		QueueID: firstNonEmptyRefundCompletionNotificationEnv(
			envRefundCompletionCloudTasksQueueID,
		),
		InternalBaseURL: firstNonEmptyRefundCompletionNotificationEnv(
			envRefundCompletionInternalBaseURL,
		),
		ServiceAccountEmail: firstNonEmptyRefundCompletionNotificationEnv(
			envRefundCompletionCloudTasksServiceAccount,
		),
		Audience: firstNonEmptyRefundCompletionNotificationEnv(
			envRefundCompletionCloudTasksAudience,
		),
	}

	return NewRefundCompletionNotificationQueue(
		ctx,
		config,
	)
}

func NewRefundCompletionNotificationQueue(
	ctx context.Context,
	config RefundCompletionNotificationQueueConfig,
) (*RefundCompletionNotificationQueue, error) {
	normalizedConfig, err := normalizeRefundCompletionNotificationQueueConfig(
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
			"create refund completion notification Cloud Tasks client: %w",
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
	) + defaultRefundCompletionNotificationTaskPath

	return &RefundCompletionNotificationQueue{
		client:              client,
		queuePath:           queuePath,
		targetURL:           targetURL,
		serviceAccountEmail: normalizedConfig.ServiceAccountEmail,
		audience:            normalizedConfig.Audience,
		dispatchDeadline:    defaultRefundCompletionNotificationDispatchDeadline,
		now:                 time.Now,
	}, nil
}

func (q *RefundCompletionNotificationQueue) EnqueueRefundCompletionNotification(
	ctx context.Context,
	delivery refunddom.CompletionNotificationDelivery,
) error {
	if q == nil {
		return errors.New(
			"refund completion notification queue is nil",
		)
	}

	if q.client == nil {
		return errors.New(
			"refund completion notification Cloud Tasks client is nil",
		)
	}

	if ctx == nil {
		ctx = context.Background()
	}

	normalizedDelivery, err := delivery.Normalize()
	if err != nil {
		return fmt.Errorf(
			"normalize refund completion notification before enqueue: %w",
			err,
		)
	}

	if normalizedDelivery.IsTerminal() {
		return refunddom.ErrCompletionNotificationNotClaimable
	}

	if normalizedDelivery.AttemptCount >= normalizedDelivery.MaxAttempts {
		return refunddom.ErrCompletionNotificationAttemptLimit
	}

	deliveryID := strings.TrimSpace(normalizedDelivery.ID)
	if deliveryID == "" {
		return refunddom.ErrCompletionNotificationDeliveryIDRequired
	}

	payload, err := json.Marshal(
		refundCompletionNotificationTaskPayload{
			DeliveryID: deliveryID,
		},
	)
	if err != nil {
		return fmt.Errorf(
			"marshal refund completion notification task payload: %w",
			err,
		)
	}

	nextAttemptNumber := normalizedDelivery.AttemptCount + 1

	taskID := buildRefundCompletionNotificationTaskID(
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
	scheduleAt := refundCompletionNotificationScheduleTime(
		normalizedDelivery,
	)

	if !scheduleAt.IsZero() && scheduleAt.After(now) {
		task.ScheduleTime = timestamppb.New(scheduleAt)

		if err := task.ScheduleTime.CheckValid(); err != nil {
			return fmt.Errorf(
				"invalid refund completion notification task schedule time: %w",
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
			"create refund completion notification task deliveryId=%q attempt=%d: %w",
			deliveryID,
			nextAttemptNumber,
			err,
		)
	}

	return nil
}

func (q *RefundCompletionNotificationQueue) Close() error {
	if q == nil || q.client == nil {
		return nil
	}

	if err := q.client.Close(); err != nil {
		return fmt.Errorf(
			"close refund completion notification Cloud Tasks client: %w",
			err,
		)
	}

	q.client = nil
	return nil
}

func normalizeRefundCompletionNotificationQueueConfig(
	config RefundCompletionNotificationQueueConfig,
) (RefundCompletionNotificationQueueConfig, error) {
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
		return RefundCompletionNotificationQueueConfig{},
			errors.New(
				"CLOUD_TASKS_PROJECT_ID is empty",
			)
	}

	if config.Location == "" {
		return RefundCompletionNotificationQueueConfig{},
			errors.New(
				"CLOUD_TASKS_LOCATION is empty",
			)
	}

	if config.QueueID == "" {
		return RefundCompletionNotificationQueueConfig{},
			errors.New(
				"CLOUD_TASKS_QUEUE_ID is empty",
			)
	}

	if config.InternalBaseURL == "" {
		return RefundCompletionNotificationQueueConfig{},
			errors.New(
				"INTERNAL_BASE_URL is empty",
			)
	}

	if err := validateRefundCompletionNotificationURL(
		config.InternalBaseURL,
	); err != nil {
		return RefundCompletionNotificationQueueConfig{},
			fmt.Errorf(
				"invalid INTERNAL_BASE_URL: %w",
				err,
			)
	}

	if config.ServiceAccountEmail == "" {
		return RefundCompletionNotificationQueueConfig{},
			errors.New(
				"CLOUD_TASKS_SERVICE_ACCOUNT is empty",
			)
	}

	if config.Audience == "" {
		config.Audience = config.InternalBaseURL
	}

	if err := validateRefundCompletionNotificationURL(
		config.Audience,
	); err != nil {
		return RefundCompletionNotificationQueueConfig{},
			fmt.Errorf(
				"invalid CLOUD_TASKS_AUDIENCE: %w",
				err,
			)
	}

	return config, nil
}

func validateRefundCompletionNotificationURL(
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

func refundCompletionNotificationScheduleTime(
	delivery refunddom.CompletionNotificationDelivery,
) time.Time {
	switch delivery.Status {
	case refunddom.CompletionNotificationStatusPending,
		refunddom.CompletionNotificationStatusRetryableFailed:
		if delivery.NextAttemptAt != nil &&
			!delivery.NextAttemptAt.IsZero() {
			return delivery.NextAttemptAt.UTC()
		}

	case refunddom.CompletionNotificationStatusProcessing:
		if delivery.ProcessingUntil != nil &&
			!delivery.ProcessingUntil.IsZero() {
			return delivery.ProcessingUntil.UTC()
		}
	}

	return time.Time{}
}

func buildRefundCompletionNotificationTaskID(
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
		"refund-completion-%s-attempt-%d",
		deliveryHash,
		attemptNumber,
	)
}

func firstNonEmptyRefundCompletionNotificationEnv(
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

func (q *RefundCompletionNotificationQueue) currentTime() time.Time {
	if q != nil && q.now != nil {
		return q.now().UTC()
	}

	return time.Now().UTC()
}

func (q *RefundCompletionNotificationQueue) normalizedDispatchDeadline() time.Duration {
	if q == nil || q.dispatchDeadline <= 0 {
		return defaultRefundCompletionNotificationDispatchDeadline
	}

	return q.dispatchDeadline
}
