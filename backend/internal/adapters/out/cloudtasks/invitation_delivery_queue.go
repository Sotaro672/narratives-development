// backend/internal/adapters/out/cloudtasks/invitation_delivery_queue.go
package cloudtasks

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
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
	invdom "narratives/internal/domain/invitation"
)

const (
	envInvitationCloudTasksProjectID = "CLOUD_TASKS_PROJECT_ID"

	envInvitationCloudTasksLocation = "CLOUD_TASKS_LOCATION"

	envInvitationCloudTasksQueueID = "CLOUD_TASKS_QUEUE_ID"

	envInvitationInternalBaseURL = "INTERNAL_BASE_URL"

	envInvitationCloudTasksServiceAccount = "CLOUD_TASKS_SERVICE_ACCOUNT"

	envInvitationCloudTasksAudience = "CLOUD_TASKS_AUDIENCE"

	defaultInvitationDeliveryTaskPath = "/internal/invitations/deliveries/process"

	defaultInvitationDeliveryDispatchDeadline = 10 * time.Minute
)

type InvitationDeliveryQueueConfig struct {
	ProjectID           string
	Location            string
	QueueID             string
	InternalBaseURL     string
	ServiceAccountEmail string
	Audience            string
}

type invitationDeliveryTaskPayload struct {
	DeliveryID string `json:"deliveryId"`
}

// InvitationDeliveryQueueは、招待メールdeliveryを処理するための
// Cloud Tasks taskを作成します。
//
// task payloadにはdelivery IDだけを含めます。
// token、メールアドレス、権限情報などは含めません。
//
// Handler側はdelivery IDを使ってFirestoreから最新のoutboxを取得します。
type InvitationDeliveryQueue struct {
	client *cloudtasksv2.Client

	queuePath string
	targetURL string

	serviceAccountEmail string
	audience            string

	dispatchDeadline time.Duration
	now              func() time.Time
}

var _ uc.InvitationDeliveryQueuePort = (*InvitationDeliveryQueue)(nil)

// NewInvitationDeliveryQueueFromEnvは、現在の共通Cloud Tasks設定から
// InvitationDeliveryQueueを生成します。
//
// 必須環境変数:
//
//   - CLOUD_TASKS_PROJECT_ID
//   - CLOUD_TASKS_LOCATION
//   - CLOUD_TASKS_QUEUE_ID
//   - INTERNAL_BASE_URL
//   - CLOUD_TASKS_SERVICE_ACCOUNT
//
// 任意環境変数:
//
//   - CLOUD_TASKS_AUDIENCE
//
// CLOUD_TASKS_AUDIENCEが空の場合は、INTERNAL_BASE_URLを使用します。
func NewInvitationDeliveryQueueFromEnv(
	ctx context.Context,
) (*InvitationDeliveryQueue, error) {
	config := InvitationDeliveryQueueConfig{
		ProjectID: firstNonEmptyInvitationDeliveryEnv(
			envInvitationCloudTasksProjectID,
			"GCP_PROJECT_ID",
			"GOOGLE_CLOUD_PROJECT",
		),
		Location: firstNonEmptyInvitationDeliveryEnv(
			envInvitationCloudTasksLocation,
		),
		QueueID: firstNonEmptyInvitationDeliveryEnv(
			envInvitationCloudTasksQueueID,
		),
		InternalBaseURL: firstNonEmptyInvitationDeliveryEnv(
			envInvitationInternalBaseURL,
		),
		ServiceAccountEmail: firstNonEmptyInvitationDeliveryEnv(
			envInvitationCloudTasksServiceAccount,
		),
		Audience: firstNonEmptyInvitationDeliveryEnv(
			envInvitationCloudTasksAudience,
		),
	}

	return NewInvitationDeliveryQueue(
		ctx,
		config,
	)
}

// NewInvitationDeliveryQueueは、明示的な設定からqueueを生成します。
func NewInvitationDeliveryQueue(
	ctx context.Context,
	config InvitationDeliveryQueueConfig,
) (*InvitationDeliveryQueue, error) {
	normalizedConfig, err :=
		normalizeInvitationDeliveryQueueConfig(config)
	if err != nil {
		return nil, err
	}

	if ctx == nil {
		ctx = context.Background()
	}

	client, err := cloudtasksv2.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"create invitation delivery Cloud Tasks client: %w",
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
	) + defaultInvitationDeliveryTaskPath

	queue := &InvitationDeliveryQueue{
		client: client,

		queuePath: queuePath,
		targetURL: targetURL,

		serviceAccountEmail: normalizedConfig.ServiceAccountEmail,
		audience:            normalizedConfig.Audience,

		dispatchDeadline: defaultInvitationDeliveryDispatchDeadline,
		now:              time.Now,
	}

	log.Printf(
		"[invitation-delivery-queue] initialized queue=%s target=%s audience=%s serviceAccount=%s",
		queue.queuePath,
		queue.targetURL,
		queue.audience,
		queue.serviceAccountEmail,
	)

	return queue, nil
}

// EnqueueInvitationDeliveryは、deliveryの次回処理taskを作成します。
//
// task名は、delivery IDと次回試行番号から決定します。
// 同じdelivery・同じ試行番号が重複投入された場合、Cloud Tasksの
// AlreadyExistsを正常終了として扱います。
func (q *InvitationDeliveryQueue) EnqueueInvitationDelivery(
	ctx context.Context,
	delivery invdom.InvitationDelivery,
) error {
	if q == nil {
		return errors.New(
			"invitation delivery queue is nil",
		)
	}

	if q.client == nil {
		return errors.New(
			"invitation delivery Cloud Tasks client is nil",
		)
	}

	if ctx == nil {
		ctx = context.Background()
	}

	normalizedDelivery, err := delivery.Normalize()
	if err != nil {
		return fmt.Errorf(
			"normalize invitation delivery before enqueue: %w",
			err,
		)
	}

	if normalizedDelivery.IsTerminal() {
		return invdom.ErrInvitationDeliveryNotClaimable
	}

	if normalizedDelivery.AttemptCount >=
		normalizedDelivery.MaxAttempts {
		return invdom.ErrInvitationDeliveryAttemptLimit
	}

	deliveryID := strings.TrimSpace(
		normalizedDelivery.ID,
	)
	if deliveryID == "" {
		return invdom.ErrInvitationDeliveryIDRequired
	}

	payload, err := json.Marshal(
		invitationDeliveryTaskPayload{
			DeliveryID: deliveryID,
		},
	)
	if err != nil {
		return fmt.Errorf(
			"marshal invitation delivery task payload: %w",
			err,
		)
	}

	nextAttemptNumber :=
		normalizedDelivery.AttemptCount + 1

	taskID := buildInvitationDeliveryTaskID(
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

	scheduleAt :=
		invitationDeliveryScheduleTime(
			normalizedDelivery,
		)

	if !scheduleAt.IsZero() &&
		scheduleAt.After(now) {
		task.ScheduleTime = timestamppb.New(
			scheduleAt,
		)

		if err := task.ScheduleTime.CheckValid(); err != nil {
			return fmt.Errorf(
				"invalid invitation delivery task schedule time: %w",
				err,
			)
		}
	}

	createdTask, err := q.client.CreateTask(
		ctx,
		&taskspb.CreateTaskRequest{
			Parent: q.queuePath,
			Task:   task,
		},
	)
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			log.Printf(
				"[invitation-delivery-queue] task already exists deliveryId=%s attempt=%d task=%s",
				deliveryID,
				nextAttemptNumber,
				taskName,
			)

			return nil
		}

		return fmt.Errorf(
			"create invitation delivery task deliveryId=%q attempt=%d: %w",
			deliveryID,
			nextAttemptNumber,
			err,
		)
	}

	createdTaskName := taskName
	if createdTask != nil &&
		strings.TrimSpace(createdTask.Name) != "" {
		createdTaskName =
			strings.TrimSpace(createdTask.Name)
	}

	log.Printf(
		"[invitation-delivery-queue] task created deliveryId=%s attempt=%d task=%s scheduleAt=%s",
		deliveryID,
		nextAttemptNumber,
		createdTaskName,
		formatInvitationDeliveryTaskTime(
			scheduleAt,
		),
	)

	return nil
}

// CloseはCloud Tasks clientを終了します。
func (q *InvitationDeliveryQueue) Close() error {
	if q == nil || q.client == nil {
		return nil
	}

	if err := q.client.Close(); err != nil {
		return fmt.Errorf(
			"close invitation delivery Cloud Tasks client: %w",
			err,
		)
	}

	q.client = nil

	return nil
}

func normalizeInvitationDeliveryQueueConfig(
	config InvitationDeliveryQueueConfig,
) (InvitationDeliveryQueueConfig, error) {
	config.ProjectID = strings.TrimSpace(
		config.ProjectID,
	)

	config.Location = strings.TrimSpace(
		config.Location,
	)

	config.QueueID = strings.TrimSpace(
		config.QueueID,
	)

	config.InternalBaseURL = strings.TrimRight(
		strings.TrimSpace(
			config.InternalBaseURL,
		),
		"/",
	)

	config.ServiceAccountEmail = strings.TrimSpace(
		config.ServiceAccountEmail,
	)

	config.Audience = strings.TrimRight(
		strings.TrimSpace(
			config.Audience,
		),
		"/",
	)

	if config.ProjectID == "" {
		return InvitationDeliveryQueueConfig{},
			errors.New(
				"CLOUD_TASKS_PROJECT_ID is empty",
			)
	}

	if config.Location == "" {
		return InvitationDeliveryQueueConfig{},
			errors.New(
				"CLOUD_TASKS_LOCATION is empty",
			)
	}

	if config.QueueID == "" {
		return InvitationDeliveryQueueConfig{},
			errors.New(
				"CLOUD_TASKS_QUEUE_ID is empty",
			)
	}

	if config.InternalBaseURL == "" {
		return InvitationDeliveryQueueConfig{},
			errors.New(
				"INTERNAL_BASE_URL is empty",
			)
	}

	if err := validateInvitationDeliveryURL(
		config.InternalBaseURL,
	); err != nil {
		return InvitationDeliveryQueueConfig{},
			fmt.Errorf(
				"invalid INTERNAL_BASE_URL: %w",
				err,
			)
	}

	if config.ServiceAccountEmail == "" {
		return InvitationDeliveryQueueConfig{},
			errors.New(
				"CLOUD_TASKS_SERVICE_ACCOUNT is empty",
			)
	}

	if config.Audience == "" {
		config.Audience =
			config.InternalBaseURL
	}

	if err := validateInvitationDeliveryURL(
		config.Audience,
	); err != nil {
		return InvitationDeliveryQueueConfig{},
			fmt.Errorf(
				"invalid CLOUD_TASKS_AUDIENCE: %w",
				err,
			)
	}

	return config, nil
}

func validateInvitationDeliveryURL(
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

func invitationDeliveryScheduleTime(
	delivery invdom.InvitationDelivery,
) time.Time {
	switch delivery.Status {
	case invdom.InvitationDeliveryStatusPending,
		invdom.InvitationDeliveryStatusRetryableFailed:
		if delivery.NextAttemptAt != nil &&
			!delivery.NextAttemptAt.IsZero() {
			return delivery.NextAttemptAt.UTC()
		}

	case invdom.InvitationDeliveryStatusProcessing:
		if delivery.ProcessingUntil != nil &&
			!delivery.ProcessingUntil.IsZero() {
			return delivery.ProcessingUntil.UTC()
		}
	}

	return time.Time{}
}

func buildInvitationDeliveryTaskID(
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
		"invitation-delivery-%s-attempt-%d",
		deliveryHash,
		attemptNumber,
	)
}

func firstNonEmptyInvitationDeliveryEnv(
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

func (q *InvitationDeliveryQueue) currentTime() time.Time {
	if q != nil && q.now != nil {
		return q.now().UTC()
	}

	return time.Now().UTC()
}

func (q *InvitationDeliveryQueue) normalizedDispatchDeadline() time.Duration {
	if q == nil ||
		q.dispatchDeadline <= 0 {
		return defaultInvitationDeliveryDispatchDeadline
	}

	return q.dispatchDeadline
}

func formatInvitationDeliveryTaskTime(
	value time.Time,
) string {
	if value.IsZero() {
		return ""
	}

	return value.UTC().Format(
		time.RFC3339Nano,
	)
}
