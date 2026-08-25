// backend/internal/adapters/out/cloudtasks/token_blueprint_create_operation_queue.go
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
	"google.golang.org/api/googleapi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	usecase "narratives/internal/application/usecase"
)

const (
	tokenBlueprintCreateOperationQueueProjectIDEnv = "TOKEN_BLUEPRINT_CREATE_OPERATION_QUEUE_PROJECT_ID"

	tokenBlueprintCreateOperationQueueLocationEnv = "TOKEN_BLUEPRINT_CREATE_OPERATION_QUEUE_LOCATION"

	tokenBlueprintCreateOperationQueueIDEnv = "TOKEN_BLUEPRINT_CREATE_OPERATION_QUEUE_ID"

	tokenBlueprintCreateOperationQueueTargetBaseURLEnv = "TOKEN_BLUEPRINT_CREATE_OPERATION_QUEUE_TARGET_BASE_URL"

	tokenBlueprintCreateOperationQueueServiceAccountEmailEnv = "TOKEN_BLUEPRINT_CREATE_OPERATION_QUEUE_SERVICE_ACCOUNT_EMAIL"

	tokenBlueprintCreateOperationQueueOIDCAudienceEnv = "TOKEN_BLUEPRINT_CREATE_OPERATION_QUEUE_OIDC_AUDIENCE"

	tokenBlueprintCreateOperationExecutePathPrefix = "/internal/token-blueprint/create-operations/"

	tokenBlueprintCreateOperationExecutePathSuffix = "/execute"
)

type TokenBlueprintCreateOperationQueueConfig struct {
	ProjectID           string
	Location            string
	QueueID             string
	TargetBaseURL       string
	ServiceAccountEmail string
	OIDCAudience        string
}

type TokenBlueprintCreateOperationQueue struct {
	Client *cloudtasksv2.Client

	ProjectID           string
	Location            string
	QueueID             string
	TargetBaseURL       string
	ServiceAccountEmail string
	OIDCAudience        string

	ownsClient bool
}

type tokenBlueprintCreateOperationTaskPayload struct {
	OperationID string `json:"operationId"`
}

func NewTokenBlueprintCreateOperationQueue(
	client *cloudtasksv2.Client,
	config TokenBlueprintCreateOperationQueueConfig,
) (*TokenBlueprintCreateOperationQueue, error) {
	if client == nil {
		return nil, errors.New(
			"cloud tasks client is nil",
		)
	}

	normalized, err :=
		normalizeTokenBlueprintCreateOperationQueueConfig(
			config,
		)
	if err != nil {
		return nil, err
	}

	return &TokenBlueprintCreateOperationQueue{
		Client: client,

		ProjectID: normalized.ProjectID,
		Location:  normalized.Location,
		QueueID:   normalized.QueueID,

		TargetBaseURL: normalized.TargetBaseURL,

		ServiceAccountEmail: normalized.ServiceAccountEmail,

		OIDCAudience: normalized.OIDCAudience,

		ownsClient: false,
	}, nil
}

func NewTokenBlueprintCreateOperationQueueFromEnv(
	ctx context.Context,
) (*TokenBlueprintCreateOperationQueue, error) {
	if ctx == nil {
		return nil, errors.New(
			"context is nil",
		)
	}

	projectID :=
		firstNonEmptyEnvironmentValue(
			tokenBlueprintCreateOperationQueueProjectIDEnv,
			"GOOGLE_CLOUD_PROJECT",
			"GCP_PROJECT",
			"PROJECT_ID",
		)

	config :=
		TokenBlueprintCreateOperationQueueConfig{
			ProjectID: projectID,

			Location: strings.TrimSpace(
				os.Getenv(
					tokenBlueprintCreateOperationQueueLocationEnv,
				),
			),

			QueueID: strings.TrimSpace(
				os.Getenv(
					tokenBlueprintCreateOperationQueueIDEnv,
				),
			),

			TargetBaseURL: strings.TrimSpace(
				os.Getenv(
					tokenBlueprintCreateOperationQueueTargetBaseURLEnv,
				),
			),

			ServiceAccountEmail: strings.TrimSpace(
				os.Getenv(
					tokenBlueprintCreateOperationQueueServiceAccountEmailEnv,
				),
			),

			OIDCAudience: strings.TrimSpace(
				os.Getenv(
					tokenBlueprintCreateOperationQueueOIDCAudienceEnv,
				),
			),
		}

	normalized, err :=
		normalizeTokenBlueprintCreateOperationQueueConfig(
			config,
		)
	if err != nil {
		return nil, err
	}

	client, err :=
		cloudtasksv2.NewClient(
			ctx,
		)
	if err != nil {
		return nil, fmt.Errorf(
			"create cloud tasks client: %w",
			err,
		)
	}

	queue :=
		&TokenBlueprintCreateOperationQueue{
			Client: client,

			ProjectID: normalized.ProjectID,
			Location:  normalized.Location,
			QueueID:   normalized.QueueID,

			TargetBaseURL: normalized.TargetBaseURL,

			ServiceAccountEmail: normalized.ServiceAccountEmail,

			OIDCAudience: normalized.OIDCAudience,

			ownsClient: true,
		}

	return queue, nil
}

var _ usecase.TokenBlueprintCreateOperationQueue = (*TokenBlueprintCreateOperationQueue)(nil)

// Enqueue creates a Cloud Tasks task for TokenBlueprint CreateOperation.
//
// The task body contains operationId only.
// All durable state required for execution is loaded from Firestore.
//
// scheduledAt is also incorporated into the deterministic task ID:
//
//	operationID + scheduledAt
//
// Therefore:
// - repeating Commit with the same queued operation is idempotent;
// - repeating the same retry enqueue is idempotent;
// - a later retry with a different scheduledAt receives a new task ID.
func (
	q *TokenBlueprintCreateOperationQueue,
) Enqueue(
	ctx context.Context,
	operationID string,
	scheduledAt time.Time,
) error {
	if ctx == nil {
		return errors.New(
			"context is nil",
		)
	}

	config, err :=
		q.resolveConfig()
	if err != nil {
		return err
	}

	operationID, err =
		normalizeTokenBlueprintCreateOperationOperationID(
			operationID,
		)
	if err != nil {
		return err
	}

	if scheduledAt.IsZero() {
		return errors.New(
			"scheduledAt is required",
		)
	}

	scheduledAt =
		scheduledAt.UTC()

	scheduleTimestamp :=
		timestamppb.New(
			scheduledAt,
		)

	if err :=
		scheduleTimestamp.CheckValid(); err != nil {
		return fmt.Errorf(
			"invalid scheduledAt: %w",
			err,
		)
	}

	payload, err :=
		json.Marshal(
			tokenBlueprintCreateOperationTaskPayload{
				OperationID: operationID,
			},
		)
	if err != nil {
		return fmt.Errorf(
			"encode token blueprint create operation task: %w",
			err,
		)
	}

	parent := fmt.Sprintf(
		"projects/%s/locations/%s/queues/%s",
		config.ProjectID,
		config.Location,
		config.QueueID,
	)

	targetURL :=
		config.TargetBaseURL +
			tokenBlueprintCreateOperationExecutePathPrefix +
			url.PathEscape(
				operationID,
			) +
			tokenBlueprintCreateOperationExecutePathSuffix

	taskID :=
		buildTokenBlueprintCreateOperationTaskID(
			operationID,
			scheduledAt,
		)

	taskName :=
		parent +
			"/tasks/" +
			taskID

	request :=
		&taskspb.CreateTaskRequest{
			Parent: parent,

			Task: &taskspb.Task{
				Name: taskName,

				MessageType: &taskspb.Task_HttpRequest{
					HttpRequest: &taskspb.HttpRequest{
						HttpMethod: taskspb.HttpMethod_POST,

						Url: targetURL,

						Headers: map[string]string{
							"Content-Type": "application/json",
						},

						Body: payload,

						AuthorizationHeader: &taskspb.HttpRequest_OidcToken{
							OidcToken: &taskspb.OidcToken{
								ServiceAccountEmail: config.ServiceAccountEmail,

								Audience: config.OIDCAudience,
							},
						},
					},
				},

				ScheduleTime: scheduleTimestamp,
			},
		}

	_, err =
		q.Client.CreateTask(
			ctx,
			request,
		)
	if err == nil {
		return nil
	}

	// Deterministic task IDにより、同じenqueueが再送された場合は
	// AlreadyExistsを成功として扱う。
	if status.Code(err) ==
		codes.AlreadyExists {
		return nil
	}

	// Cloud Tasks APIがHTTP 409として返す場合も
	// 同様に冪等成功として扱う。
	var apiErr *googleapi.Error

	if errors.As(
		err,
		&apiErr,
	) &&
		apiErr.Code == 409 {
		return nil
	}

	return fmt.Errorf(
		"create token blueprint create operation task operationId=%q queue=%q scheduledAt=%s: %w",
		operationID,
		parent,
		scheduledAt.Format(
			time.RFC3339Nano,
		),
		err,
	)
}

func (
	q *TokenBlueprintCreateOperationQueue,
) Close() error {
	if q == nil ||
		q.Client == nil ||
		!q.ownsClient {
		return nil
	}

	if err :=
		q.Client.Close(); err != nil {
		return fmt.Errorf(
			"close cloud tasks client: %w",
			err,
		)
	}

	q.Client = nil
	q.ownsClient = false

	return nil
}

func (
	q *TokenBlueprintCreateOperationQueue,
) resolveConfig() (
	TokenBlueprintCreateOperationQueueConfig,
	error,
) {
	if q == nil {
		return TokenBlueprintCreateOperationQueueConfig{},
			errors.New(
				"token blueprint create operation queue is nil",
			)
	}

	if q.Client == nil {
		return TokenBlueprintCreateOperationQueueConfig{},
			errors.New(
				"cloud tasks client is nil",
			)
	}

	return normalizeTokenBlueprintCreateOperationQueueConfig(
		TokenBlueprintCreateOperationQueueConfig{
			ProjectID: q.ProjectID,

			Location: q.Location,

			QueueID: q.QueueID,

			TargetBaseURL: q.TargetBaseURL,

			ServiceAccountEmail: q.ServiceAccountEmail,

			OIDCAudience: q.OIDCAudience,
		},
	)
}

func normalizeTokenBlueprintCreateOperationQueueConfig(
	config TokenBlueprintCreateOperationQueueConfig,
) (
	TokenBlueprintCreateOperationQueueConfig,
	error,
) {
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

	config.TargetBaseURL =
		strings.TrimRight(
			strings.TrimSpace(
				config.TargetBaseURL,
			),
			"/",
		)

	config.ServiceAccountEmail =
		strings.TrimSpace(
			config.ServiceAccountEmail,
		)

	config.OIDCAudience =
		strings.TrimRight(
			strings.TrimSpace(
				config.OIDCAudience,
			),
			"/",
		)

	if config.ProjectID == "" {
		return TokenBlueprintCreateOperationQueueConfig{},
			fmt.Errorf(
				"%s is required",
				tokenBlueprintCreateOperationQueueProjectIDEnv,
			)
	}

	if err :=
		validateTokenBlueprintCreateOperationQueueIdentifier(
			"projectId",
			config.ProjectID,
		); err != nil {
		return TokenBlueprintCreateOperationQueueConfig{},
			err
	}

	if config.Location == "" {
		return TokenBlueprintCreateOperationQueueConfig{},
			fmt.Errorf(
				"%s is required",
				tokenBlueprintCreateOperationQueueLocationEnv,
			)
	}

	if err :=
		validateTokenBlueprintCreateOperationQueueIdentifier(
			"location",
			config.Location,
		); err != nil {
		return TokenBlueprintCreateOperationQueueConfig{},
			err
	}

	if config.QueueID == "" {
		return TokenBlueprintCreateOperationQueueConfig{},
			fmt.Errorf(
				"%s is required",
				tokenBlueprintCreateOperationQueueIDEnv,
			)
	}

	if err :=
		validateTokenBlueprintCreateOperationQueueIdentifier(
			"queueId",
			config.QueueID,
		); err != nil {
		return TokenBlueprintCreateOperationQueueConfig{},
			err
	}

	if config.TargetBaseURL == "" {
		return TokenBlueprintCreateOperationQueueConfig{},
			fmt.Errorf(
				"%s is required",
				tokenBlueprintCreateOperationQueueTargetBaseURLEnv,
			)
	}

	parsedTargetURL, err :=
		url.Parse(
			config.TargetBaseURL,
		)
	if err != nil {
		return TokenBlueprintCreateOperationQueueConfig{},
			fmt.Errorf(
				"parse target base URL: %w",
				err,
			)
	}

	if parsedTargetURL.Scheme != "https" &&
		parsedTargetURL.Scheme != "http" {
		return TokenBlueprintCreateOperationQueueConfig{},
			errors.New(
				"target base URL must use http or https",
			)
	}

	if parsedTargetURL.Host == "" {
		return TokenBlueprintCreateOperationQueueConfig{},
			errors.New(
				"target base URL host is required",
			)
	}

	if parsedTargetURL.RawQuery != "" ||
		parsedTargetURL.Fragment != "" {
		return TokenBlueprintCreateOperationQueueConfig{},
			errors.New(
				"target base URL must not contain a query or fragment",
			)
	}

	if config.ServiceAccountEmail == "" {
		return TokenBlueprintCreateOperationQueueConfig{},
			fmt.Errorf(
				"%s is required",
				tokenBlueprintCreateOperationQueueServiceAccountEmailEnv,
			)
	}

	if strings.ContainsAny(
		config.ServiceAccountEmail,
		"\r\n\x00",
	) ||
		!strings.Contains(
			config.ServiceAccountEmail,
			"@",
		) {
		return TokenBlueprintCreateOperationQueueConfig{},
			errors.New(
				"service account email is invalid",
			)
	}

	if config.OIDCAudience == "" {
		config.OIDCAudience =
			config.TargetBaseURL
	}

	parsedAudience, err :=
		url.Parse(
			config.OIDCAudience,
		)
	if err != nil {
		return TokenBlueprintCreateOperationQueueConfig{},
			fmt.Errorf(
				"parse OIDC audience: %w",
				err,
			)
	}

	if parsedAudience.Scheme != "https" &&
		parsedAudience.Scheme != "http" {
		return TokenBlueprintCreateOperationQueueConfig{},
			errors.New(
				"OIDC audience must use http or https",
			)
	}

	if parsedAudience.Host == "" {
		return TokenBlueprintCreateOperationQueueConfig{},
			errors.New(
				"OIDC audience host is required",
			)
	}

	return config, nil
}

func normalizeTokenBlueprintCreateOperationOperationID(
	operationID string,
) (string, error) {
	operationID =
		strings.TrimSpace(
			operationID,
		)

	if operationID == "" {
		return "",
			errors.New(
				"operationId is required",
			)
	}

	if len(operationID) > 512 {
		return "",
			errors.New(
				"operationId must not exceed 512 characters",
			)
	}

	if strings.Contains(
		operationID,
		"/",
	) ||
		strings.Contains(
			operationID,
			"://",
		) ||
		strings.ContainsAny(
			operationID,
			"\r\n\x00",
		) {
		return "",
			errors.New(
				"operationId is invalid",
			)
	}

	return operationID, nil
}

func validateTokenBlueprintCreateOperationQueueIdentifier(
	fieldName string,
	value string,
) error {
	value =
		strings.TrimSpace(
			value,
		)

	if value == "" {
		return fmt.Errorf(
			"%s is required",
			fieldName,
		)
	}

	for _, character := range value {
		switch {
		case character >= 'a' &&
			character <= 'z':

		case character >= 'A' &&
			character <= 'Z':

		case character >= '0' &&
			character <= '9':

		case character == '-':

		case character == '_':

		case character == '.':

		default:
			return fmt.Errorf(
				"%s contains an invalid character",
				fieldName,
			)
		}
	}

	return nil
}

func buildTokenBlueprintCreateOperationTaskID(
	operationID string,
	scheduledAt time.Time,
) string {
	source :=
		operationID +
			"\x00" +
			scheduledAt.
				UTC().
				Format(
					time.RFC3339Nano,
				)

	digest :=
		sha256.Sum256(
			[]byte(source),
		)

	return "token-blueprint-create-" +
		hex.EncodeToString(
			digest[:16],
		)
}
