// backend/internal/adapters/out/cloudtasks/resale_payout_notification_queue.go
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
	bankpayoutdom "narratives/internal/domain/bankPayout"
)

const (
	envResalePayoutCloudTasksProjectID      = "CLOUD_TASKS_PROJECT_ID"
	envResalePayoutCloudTasksLocation       = "CLOUD_TASKS_LOCATION"
	envResalePayoutCloudTasksQueueID        = "CLOUD_TASKS_QUEUE_ID"
	envResalePayoutInternalBaseURL          = "INTERNAL_BASE_URL"
	envResalePayoutCloudTasksServiceAccount = "CLOUD_TASKS_SERVICE_ACCOUNT"
	envResalePayoutCloudTasksAudience       = "CLOUD_TASKS_AUDIENCE"

	defaultResalePayoutNotificationTaskPath = "/internal/resale-payout-notifications/process"

	defaultResalePayoutNotificationDispatchDeadline = 10 * time.Minute
)

type ResalePayoutNotificationQueueConfig struct {
	ProjectID           string
	Location            string
	QueueID             string
	InternalBaseURL     string
	ServiceAccountEmail string
	Audience            string
}

type resalePayoutNotificationTaskPayload struct {
	DeliveryID string `json:"deliveryId"`
}

type ResalePayoutNotificationQueue struct {
	client *cloudtasksv2.Client

	queuePath string
	targetURL string

	serviceAccountEmail string
	audience            string

	dispatchDeadline time.Duration
	now              func() time.Time
}

var _ applicationport.ResalePayoutNotificationQueuePort = (*ResalePayoutNotificationQueue)(nil)

func NewResalePayoutNotificationQueueFromEnv(
	ctx context.Context,
) (*ResalePayoutNotificationQueue, error) {
	config := ResalePayoutNotificationQueueConfig{
		ProjectID: firstNonEmptyResalePayoutNotificationEnv(
			envResalePayoutCloudTasksProjectID,
			"GCP_PROJECT_ID",
			"GOOGLE_CLOUD_PROJECT",
		),
		Location: firstNonEmptyResalePayoutNotificationEnv(
			envResalePayoutCloudTasksLocation,
		),
		QueueID: firstNonEmptyResalePayoutNotificationEnv(
			envResalePayoutCloudTasksQueueID,
		),
		InternalBaseURL: firstNonEmptyResalePayoutNotificationEnv(
			envResalePayoutInternalBaseURL,
		),
		ServiceAccountEmail: firstNonEmptyResalePayoutNotificationEnv(
			envResalePayoutCloudTasksServiceAccount,
		),
		Audience: firstNonEmptyResalePayoutNotificationEnv(
			envResalePayoutCloudTasksAudience,
		),
	}

	return NewResalePayoutNotificationQueue(
		ctx,
		config,
	)
}

func NewResalePayoutNotificationQueue(
	ctx context.Context,
	config ResalePayoutNotificationQueueConfig,
) (*ResalePayoutNotificationQueue, error) {
	normalizedConfig, err := normalizeResalePayoutNotificationQueueConfig(
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
			"create resale payout notification Cloud Tasks client: %w",
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
	) + defaultResalePayoutNotificationTaskPath

	return &ResalePayoutNotificationQueue{
		client:              client,
		queuePath:           queuePath,
		targetURL:           targetURL,
		serviceAccountEmail: normalizedConfig.ServiceAccountEmail,
		audience:            normalizedConfig.Audience,
		dispatchDeadline:    defaultResalePayoutNotificationDispatchDeadline,
		now:                 time.Now,
	}, nil
}

func (q *ResalePayoutNotificationQueue) EnqueueResalePayoutNotification(
	ctx context.Context,
	delivery bankpayoutdom.PayoutNotificationDelivery,
) error {
	if q == nil {
		return errors.New(
			"resale payout notification queue is nil",
		)
	}

	if q.client == nil {
		return errors.New(
			"resale payout notification Cloud Tasks client is nil",
		)
	}

	if ctx == nil {
		ctx = context.Background()
	}

	normalizedDelivery, err := delivery.Normalize()
	if err != nil {
		return fmt.Errorf(
			"normalize resale payout notification before enqueue: %w",
			err,
		)
	}

	if normalizedDelivery.IsTerminal() {
		return bankpayoutdom.ErrPayoutNotificationNotClaimable
	}

	if normalizedDelivery.AttemptCount >= normalizedDelivery.MaxAttempts {
		return bankpayoutdom.ErrPayoutNotificationAttemptLimit
	}

	deliveryID := strings.TrimSpace(normalizedDelivery.ID)
	if deliveryID == "" {
		return bankpayoutdom.ErrPayoutNotificationDeliveryIDRequired
	}

	payload, err := json.Marshal(
		resalePayoutNotificationTaskPayload{
			DeliveryID: deliveryID,
		},
	)
	if err != nil {
		return fmt.Errorf(
			"marshal resale payout notification task payload: %w",
			err,
		)
	}

	nextAttemptNumber := normalizedDelivery.AttemptCount + 1

	taskID := buildResalePayoutNotificationTaskID(
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
	scheduleAt := resalePayoutNotificationScheduleTime(
		normalizedDelivery,
	)

	if !scheduleAt.IsZero() && scheduleAt.After(now) {
		task.ScheduleTime = timestamppb.New(scheduleAt)

		if err := task.ScheduleTime.CheckValid(); err != nil {
			return fmt.Errorf(
				"invalid resale payout notification task schedule time: %w",
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
			"create resale payout notification task deliveryId=%q attempt=%d: %w",
			deliveryID,
			nextAttemptNumber,
			err,
		)
	}

	return nil
}

func (q *ResalePayoutNotificationQueue) Close() error {
	if q == nil || q.client == nil {
		return nil
	}

	if err := q.client.Close(); err != nil {
		return fmt.Errorf(
			"close resale payout notification Cloud Tasks client: %w",
			err,
		)
	}

	q.client = nil
	return nil
}

func normalizeResalePayoutNotificationQueueConfig(
	config ResalePayoutNotificationQueueConfig,
) (ResalePayoutNotificationQueueConfig, error) {
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
		return ResalePayoutNotificationQueueConfig{},
			errors.New(
				"CLOUD_TASKS_PROJECT_ID is empty",
			)
	}

	if config.Location == "" {
		return ResalePayoutNotificationQueueConfig{},
			errors.New(
				"CLOUD_TASKS_LOCATION is empty",
			)
	}

	if config.QueueID == "" {
		return ResalePayoutNotificationQueueConfig{},
			errors.New(
				"CLOUD_TASKS_QUEUE_ID is empty",
			)
	}

	if config.InternalBaseURL == "" {
		return ResalePayoutNotificationQueueConfig{},
			errors.New(
				"INTERNAL_BASE_URL is empty",
			)
	}

	if err := validateResalePayoutNotificationURL(
		config.InternalBaseURL,
	); err != nil {
		return ResalePayoutNotificationQueueConfig{},
			fmt.Errorf(
				"invalid INTERNAL_BASE_URL: %w",
				err,
			)
	}

	if config.ServiceAccountEmail == "" {
		return ResalePayoutNotificationQueueConfig{},
			errors.New(
				"CLOUD_TASKS_SERVICE_ACCOUNT is empty",
			)
	}

	if config.Audience == "" {
		config.Audience = config.InternalBaseURL
	}

	if err := validateResalePayoutNotificationURL(
		config.Audience,
	); err != nil {
		return ResalePayoutNotificationQueueConfig{},
			fmt.Errorf(
				"invalid CLOUD_TASKS_AUDIENCE: %w",
				err,
			)
	}

	return config, nil
}

func validateResalePayoutNotificationURL(
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

func resalePayoutNotificationScheduleTime(
	delivery bankpayoutdom.PayoutNotificationDelivery,
) time.Time {
	switch delivery.Status {
	case bankpayoutdom.PayoutNotificationStatusPending,
		bankpayoutdom.PayoutNotificationStatusRetryableFailed:
		if delivery.NextAttemptAt != nil &&
			!delivery.NextAttemptAt.IsZero() {
			return delivery.NextAttemptAt.UTC()
		}

	case bankpayoutdom.PayoutNotificationStatusProcessing:
		if delivery.ProcessingUntil != nil &&
			!delivery.ProcessingUntil.IsZero() {
			return delivery.ProcessingUntil.UTC()
		}
	}

	return time.Time{}
}

func buildResalePayoutNotificationTaskID(
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
		"resale-payout-%s-attempt-%d",
		deliveryHash,
		attemptNumber,
	)
}

func firstNonEmptyResalePayoutNotificationEnv(
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

func (q *ResalePayoutNotificationQueue) currentTime() time.Time {
	if q != nil && q.now != nil {
		return q.now().UTC()
	}

	return time.Now().UTC()
}

func (q *ResalePayoutNotificationQueue) normalizedDispatchDeadline() time.Duration {
	if q == nil || q.dispatchDeadline <= 0 {
		return defaultResalePayoutNotificationDispatchDeadline
	}

	return q.dispatchDeadline
}
