// backend/internal/adapters/out/cloudtasks/settlement_queue.go
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

	settlementdom "narratives/internal/domain/settlement"
)

const (
	envSettlementCloudTasksProjectID = "CLOUD_TASKS_PROJECT_ID"

	envSettlementCloudTasksLocation = "CLOUD_TASKS_LOCATION"

	envSettlementCloudTasksQueueID = "CLOUD_TASKS_QUEUE_ID"

	envSettlementInternalBaseURL = "INTERNAL_BASE_URL"

	envSettlementCloudTasksServiceAccount = "CLOUD_TASKS_SERVICE_ACCOUNT"

	envSettlementCloudTasksAudience = "CLOUD_TASKS_AUDIENCE"

	defaultSettlementTaskPath = "/internal/settlements/process"

	defaultSettlementDispatchDeadline = 10 * time.Minute
)

// ============================================================
// Config
// ============================================================

type SettlementQueueConfig struct {
	ProjectID string
	Location  string
	QueueID   string

	InternalBaseURL string

	ServiceAccountEmail string
	Audience            string
}

// ============================================================
// Payload
// ============================================================

// settlementTaskPayload intentionally contains only SettlementID.
//
// Amount, destination Stripe Account, Charge ID, TransferGroup, and other
// financial information must always be loaded from the authoritative
// Settlement document by the worker.
type settlementTaskPayload struct {
	SettlementID string `json:"settlementId"`
}

// ============================================================
// SettlementQueue
// ============================================================

// SettlementQueue enqueues Stripe Connect Settlement processing.
//
// Cloud Tasks delivery is at-least-once.
//
// Duplicate Stripe money movement is prevented by:
//
//  1. SettlementRepositoryFS.ClaimForTransfer
//  2. deterministic Stripe Idempotency-Key
//
// Therefore the Cloud Task itself contains only SettlementID.
type SettlementQueue struct {
	client *cloudtasksv2.Client

	queuePath string
	targetURL string

	serviceAccountEmail string
	audience            string

	dispatchDeadline time.Duration
}

// NewSettlementQueueFromEnv creates SettlementQueue from the shared
// Cloud Tasks environment configuration.
//
// Required:
//
//	CLOUD_TASKS_PROJECT_ID
//	CLOUD_TASKS_LOCATION
//	CLOUD_TASKS_QUEUE_ID
//	INTERNAL_BASE_URL
//	CLOUD_TASKS_SERVICE_ACCOUNT
//
// Optional:
//
//	CLOUD_TASKS_AUDIENCE
//
// When CLOUD_TASKS_AUDIENCE is empty, INTERNAL_BASE_URL is used.
func NewSettlementQueueFromEnv(
	ctx context.Context,
) (*SettlementQueue, error) {
	config := SettlementQueueConfig{
		ProjectID: firstNonEmptySettlementEnv(
			envSettlementCloudTasksProjectID,
			"GCP_PROJECT_ID",
			"GOOGLE_CLOUD_PROJECT",
		),
		Location: firstNonEmptySettlementEnv(
			envSettlementCloudTasksLocation,
		),
		QueueID: firstNonEmptySettlementEnv(
			envSettlementCloudTasksQueueID,
		),
		InternalBaseURL: firstNonEmptySettlementEnv(
			envSettlementInternalBaseURL,
		),
		ServiceAccountEmail: firstNonEmptySettlementEnv(
			envSettlementCloudTasksServiceAccount,
		),
		Audience: firstNonEmptySettlementEnv(
			envSettlementCloudTasksAudience,
		),
	}

	return NewSettlementQueue(
		ctx,
		config,
	)
}

// NewSettlementQueue creates SettlementQueue from explicit configuration.
func NewSettlementQueue(
	ctx context.Context,
	config SettlementQueueConfig,
) (*SettlementQueue, error) {
	normalizedConfig, err :=
		normalizeSettlementQueueConfig(
			config,
		)
	if err != nil {
		return nil, err
	}

	if ctx == nil {
		ctx = context.Background()
	}

	client, err :=
		cloudtasksv2.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"create settlement Cloud Tasks client: %w",
			err,
		)
	}

	queuePath := fmt.Sprintf(
		"projects/%s/locations/%s/queues/%s",
		normalizedConfig.ProjectID,
		normalizedConfig.Location,
		normalizedConfig.QueueID,
	)

	targetURL :=
		strings.TrimRight(
			normalizedConfig.InternalBaseURL,
			"/",
		) +
			defaultSettlementTaskPath

	return &SettlementQueue{
		client: client,

		queuePath: queuePath,
		targetURL: targetURL,

		serviceAccountEmail: normalizedConfig.ServiceAccountEmail,

		audience: normalizedConfig.Audience,

		dispatchDeadline: defaultSettlementDispatchDeadline,
	}, nil
}

// ============================================================
// Enqueue
// ============================================================

// EnqueueSettlementTransfer creates the Cloud Task that executes one
// Settlement's Stripe Connect Transfer.
//
// The task name is deterministic from SettlementID.
//
// Repeated enqueue calls for the same Settlement are therefore idempotent:
// Cloud Tasks AlreadyExists is treated as success.
//
// Retryable Stripe failures should cause the worker to return a non-2xx
// response. Cloud Tasks then retries this same task according to the queue's
// retry configuration.
func (q *SettlementQueue) EnqueueSettlementTransfer(
	ctx context.Context,
	settlementID string,
) error {
	if q == nil {
		return errors.New(
			"settlement queue is nil",
		)
	}

	if q.client == nil {
		return errors.New(
			"settlement Cloud Tasks client is nil",
		)
	}

	if ctx == nil {
		ctx = context.Background()
	}

	settlementID =
		strings.TrimSpace(
			settlementID,
		)

	if settlementID == "" ||
		strings.Contains(
			settlementID,
			"/",
		) {
		return settlementdom.ErrInvalidID
	}

	payload, err := json.Marshal(
		settlementTaskPayload{
			SettlementID: settlementID,
		},
	)
	if err != nil {
		return fmt.Errorf(
			"marshal settlement task payload: %w",
			err,
		)
	}

	taskID :=
		buildSettlementTaskID(
			settlementID,
		)

	taskName := fmt.Sprintf(
		"%s/tasks/%s",
		q.queuePath,
		taskID,
	)

	httpRequest :=
		&taskspb.HttpRequest{
			HttpMethod: taskspb.HttpMethod_POST,

			Url: q.targetURL,

			Headers: map[string]string{
				"Content-Type": "application/json",
			},

			Body: payload,

			AuthorizationHeader: &taskspb.HttpRequest_OidcToken{
				OidcToken: &taskspb.OidcToken{
					ServiceAccountEmail: q.serviceAccountEmail,

					Audience: q.audience,
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

	_, err = q.client.CreateTask(
		ctx,
		&taskspb.CreateTaskRequest{
			Parent: q.queuePath,

			Task: task,
		},
	)
	if err != nil {
		if status.Code(err) ==
			codes.AlreadyExists {
			return nil
		}

		return fmt.Errorf(
			"create settlement task settlementId=%q: %w",
			settlementID,
			err,
		)
	}

	return nil
}

// ============================================================
// Close
// ============================================================

func (q *SettlementQueue) Close() error {
	if q == nil ||
		q.client == nil {
		return nil
	}

	if err := q.client.Close(); err != nil {
		return fmt.Errorf(
			"close settlement Cloud Tasks client: %w",
			err,
		)
	}

	q.client = nil

	return nil
}

// ============================================================
// Config normalization
// ============================================================

func normalizeSettlementQueueConfig(
	config SettlementQueueConfig,
) (SettlementQueueConfig, error) {
	config.ProjectID =
		strings.TrimSpace(
			config.ProjectID,
		)

	config.Location =
		strings.TrimSpace(
			config.Location,
		)

	config.QueueID =
		strings.TrimSpace(
			config.QueueID,
		)

	config.InternalBaseURL =
		strings.TrimRight(
			strings.TrimSpace(
				config.InternalBaseURL,
			),
			"/",
		)

	config.ServiceAccountEmail =
		strings.TrimSpace(
			config.ServiceAccountEmail,
		)

	config.Audience =
		strings.TrimRight(
			strings.TrimSpace(
				config.Audience,
			),
			"/",
		)

	if config.ProjectID == "" {
		return SettlementQueueConfig{},
			errors.New(
				"CLOUD_TASKS_PROJECT_ID is empty",
			)
	}

	if config.Location == "" {
		return SettlementQueueConfig{},
			errors.New(
				"CLOUD_TASKS_LOCATION is empty",
			)
	}

	if config.QueueID == "" {
		return SettlementQueueConfig{},
			errors.New(
				"CLOUD_TASKS_QUEUE_ID is empty",
			)
	}

	if config.InternalBaseURL == "" {
		return SettlementQueueConfig{},
			errors.New(
				"INTERNAL_BASE_URL is empty",
			)
	}

	if err :=
		validateSettlementQueueURL(
			config.InternalBaseURL,
		); err != nil {
		return SettlementQueueConfig{},
			fmt.Errorf(
				"invalid INTERNAL_BASE_URL: %w",
				err,
			)
	}

	if config.ServiceAccountEmail == "" {
		return SettlementQueueConfig{},
			errors.New(
				"CLOUD_TASKS_SERVICE_ACCOUNT is empty",
			)
	}

	if config.Audience == "" {
		config.Audience =
			config.InternalBaseURL
	}

	if err :=
		validateSettlementQueueURL(
			config.Audience,
		); err != nil {
		return SettlementQueueConfig{},
			fmt.Errorf(
				"invalid CLOUD_TASKS_AUDIENCE: %w",
				err,
			)
	}

	return config, nil
}

// ============================================================
// URL validation
// ============================================================

func validateSettlementQueueURL(
	value string,
) error {
	value =
		strings.TrimSpace(
			value,
		)

	if value == "" {
		return errors.New(
			"URL is empty",
		)
	}

	parsed, err :=
		url.ParseRequestURI(
			value,
		)
	if err != nil {
		return fmt.Errorf(
			"parse URL: %w",
			err,
		)
	}

	switch strings.ToLower(
		parsed.Scheme,
	) {
	case "http",
		"https":

	default:
		return errors.New(
			"URL must use http or https",
		)
	}

	if strings.TrimSpace(
		parsed.Host,
	) == "" {
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

// ============================================================
// Task ID
// ============================================================

// buildSettlementTaskID creates a Cloud Tasks-safe deterministic task ID.
//
// SettlementID itself may contain characters that should not be exposed or
// depended on as a Cloud Tasks task name, therefore SHA-256 is used.
func buildSettlementTaskID(
	settlementID string,
) string {
	settlementID =
		strings.TrimSpace(
			settlementID,
		)

	digest :=
		sha256.Sum256(
			[]byte(
				settlementID,
			),
		)

	settlementHash :=
		hex.EncodeToString(
			digest[:16],
		)

	return fmt.Sprintf(
		"settlement-%s",
		settlementHash,
	)
}

// ============================================================
// Environment
// ============================================================

func firstNonEmptySettlementEnv(
	keys ...string,
) string {
	for _, key := range keys {
		key =
			strings.TrimSpace(
				key,
			)

		if key == "" {
			continue
		}

		value :=
			strings.TrimSpace(
				os.Getenv(
					key,
				),
			)

		if value != "" {
			return value
		}
	}

	return ""
}

// ============================================================
// Deadline
// ============================================================

func (q *SettlementQueue) normalizedDispatchDeadline() time.Duration {
	if q == nil ||
		q.dispatchDeadline <= 0 {
		return defaultSettlementDispatchDeadline
	}

	return q.dispatchDeadline
}
