// backend/internal/adapters/out/cloudtasks/list_save_operation_queue.go
package cloudtasks

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	usecase "narratives/internal/application/usecase"
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
)

const (
	listSaveOperationQueueProjectIDEnv           = "LIST_SAVE_OPERATION_QUEUE_PROJECT_ID"
	listSaveOperationQueueLocationEnv            = "LIST_SAVE_OPERATION_QUEUE_LOCATION"
	listSaveOperationQueueIDEnv                  = "LIST_SAVE_OPERATION_QUEUE_ID"
	listSaveOperationQueueTargetBaseURLEnv       = "LIST_SAVE_OPERATION_QUEUE_TARGET_BASE_URL"
	listSaveOperationQueueServiceAccountEmailEnv = "LIST_SAVE_OPERATION_QUEUE_SERVICE_ACCOUNT_EMAIL"
	listSaveOperationQueueOIDCAudienceEnv        = "LIST_SAVE_OPERATION_QUEUE_OIDC_AUDIENCE"
	listSaveOperationRetryPathPrefix             = "/internal/list/save-operations/"
	listSaveOperationRetryPathSuffix             = "/retry"
)

type ListSaveOperationQueueConfig struct {
	ProjectID           string
	Location            string
	QueueID             string
	TargetBaseURL       string
	ServiceAccountEmail string
	OIDCAudience        string
}
type ListSaveOperationQueue struct {
	Client              *cloudtasksv2.Client
	ProjectID           string
	Location            string
	QueueID             string
	TargetBaseURL       string
	ServiceAccountEmail string
	OIDCAudience        string
	ownsClient          bool
}
type listSaveOperationRetryTaskPayload struct {
	OperationID string `json:"operationId"`
}

func NewListSaveOperationQueue(client *cloudtasksv2.Client, config ListSaveOperationQueueConfig) (*ListSaveOperationQueue, error) {
	if client == nil {
		return nil, errors.New("cloud tasks client is nil")
	}
	normalized, err := normalizeListSaveOperationQueueConfig(config)
	if err != nil {
		return nil, err
	}
	return &ListSaveOperationQueue{
		Client:              client,
		ProjectID:           normalized.ProjectID,
		Location:            normalized.Location,
		QueueID:             normalized.QueueID,
		TargetBaseURL:       normalized.TargetBaseURL,
		ServiceAccountEmail: normalized.ServiceAccountEmail,
		OIDCAudience:        normalized.OIDCAudience,
		ownsClient:          false,
	}, nil
}
func NewListSaveOperationQueueFromEnv(ctx context.Context) (*ListSaveOperationQueue, error) {
	if ctx == nil {
		return nil, errors.New("context is nil")
	}
	projectID := firstNonEmptyEnvironmentValue(
		listSaveOperationQueueProjectIDEnv,
		"GOOGLE_CLOUD_PROJECT",
		"GCP_PROJECT",
		"PROJECT_ID",
	)
	config := ListSaveOperationQueueConfig{
		ProjectID:           projectID,
		Location:            strings.TrimSpace(os.Getenv(listSaveOperationQueueLocationEnv)),
		QueueID:             strings.TrimSpace(os.Getenv(listSaveOperationQueueIDEnv)),
		TargetBaseURL:       strings.TrimSpace(os.Getenv(listSaveOperationQueueTargetBaseURLEnv)),
		ServiceAccountEmail: strings.TrimSpace(os.Getenv(listSaveOperationQueueServiceAccountEmailEnv)),
		OIDCAudience:        strings.TrimSpace(os.Getenv(listSaveOperationQueueOIDCAudienceEnv)),
	}
	normalized, err := normalizeListSaveOperationQueueConfig(config)
	if err != nil {
		return nil, err
	}
	client, err := cloudtasksv2.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("create cloud tasks client: %w", err)
	}
	queue := &ListSaveOperationQueue{
		Client:              client,
		ProjectID:           normalized.ProjectID,
		Location:            normalized.Location,
		QueueID:             normalized.QueueID,
		TargetBaseURL:       normalized.TargetBaseURL,
		ServiceAccountEmail: normalized.ServiceAccountEmail,
		OIDCAudience:        normalized.OIDCAudience,
		ownsClient:          true,
	}
	return queue, nil
}

var _ usecase.ListSaveOperationRetryQueue = (*ListSaveOperationQueue)(nil)

func (q *ListSaveOperationQueue) EnqueueRetry(ctx context.Context, operationID string, scheduledAt time.Time) error {
	if ctx == nil {
		return errors.New("context is nil")
	}
	config, err := q.resolveConfig()
	if err != nil {
		return err
	}
	operationID, err = normalizeListSaveOperationOperationID(operationID)
	if err != nil {
		return err
	}
	if scheduledAt.IsZero() {
		return errors.New("scheduledAt is required")
	}
	scheduledAt = scheduledAt.UTC()
	scheduleTimestamp := timestamppb.New(scheduledAt)
	if err := scheduleTimestamp.CheckValid(); err != nil {
		return fmt.Errorf("invalid scheduledAt: %w", err)
	}
	payload, err := json.Marshal(listSaveOperationRetryTaskPayload{
		OperationID: operationID,
	})
	if err != nil {
		return fmt.Errorf("encode list save operation retry task: %w", err)
	}
	parent := fmt.Sprintf(
		"projects/%s/locations/%s/queues/%s",
		config.ProjectID,
		config.Location,
		config.QueueID,
	)
	targetURL := config.TargetBaseURL +
		listSaveOperationRetryPathPrefix +
		url.PathEscape(operationID) +
		listSaveOperationRetryPathSuffix
	taskID := buildListSaveOperationRetryTaskID(operationID, scheduledAt)
	taskName := parent + "/tasks/" + taskID
	request := &taskspb.CreateTaskRequest{
		Parent: parent,
		Task: &taskspb.Task{
			Name: taskName,
			MessageType: &taskspb.Task_HttpRequest{
				HttpRequest: &taskspb.HttpRequest{
					HttpMethod: taskspb.HttpMethod_POST,
					Url:        targetURL,
					Headers: map[string]string{
						"Content-Type": "application/json",
					},
					Body: payload,
					AuthorizationHeader: &taskspb.HttpRequest_OidcToken{
						OidcToken: &taskspb.OidcToken{
							ServiceAccountEmail: config.ServiceAccountEmail,
							Audience:            config.OIDCAudience,
						},
					},
				},
			},
			ScheduleTime: scheduleTimestamp,
		},
	}
	_, err = q.Client.CreateTask(ctx, request)
	if err == nil {
		return nil
	}
	if status.Code(err) == codes.AlreadyExists {
		return nil
	}
	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) && apiErr.Code == 409 {
		return nil
	}
	return fmt.Errorf(
		"create list save operation retry task operationId=%q queue=%q scheduledAt=%s: %w",
		operationID,
		parent,
		scheduledAt.Format(time.RFC3339Nano),
		err,
	)
}
func (q *ListSaveOperationQueue) Close() error {
	if q == nil || q.Client == nil || !q.ownsClient {
		return nil
	}
	if err := q.Client.Close(); err != nil {
		return fmt.Errorf("close cloud tasks client: %w", err)
	}
	q.Client = nil
	q.ownsClient = false
	return nil
}
func (q *ListSaveOperationQueue) resolveConfig() (ListSaveOperationQueueConfig, error) {
	if q == nil {
		return ListSaveOperationQueueConfig{}, errors.New("list save operation queue is nil")
	}
	if q.Client == nil {
		return ListSaveOperationQueueConfig{}, errors.New("cloud tasks client is nil")
	}
	return normalizeListSaveOperationQueueConfig(ListSaveOperationQueueConfig{
		ProjectID:           q.ProjectID,
		Location:            q.Location,
		QueueID:             q.QueueID,
		TargetBaseURL:       q.TargetBaseURL,
		ServiceAccountEmail: q.ServiceAccountEmail,
		OIDCAudience:        q.OIDCAudience,
	})
}
func normalizeListSaveOperationQueueConfig(config ListSaveOperationQueueConfig) (ListSaveOperationQueueConfig, error) {
	config.ProjectID = strings.TrimSpace(config.ProjectID)
	config.Location = strings.TrimSpace(config.Location)
	config.QueueID = strings.TrimSpace(config.QueueID)
	config.TargetBaseURL = strings.TrimRight(strings.TrimSpace(config.TargetBaseURL), "/")
	config.ServiceAccountEmail = strings.TrimSpace(config.ServiceAccountEmail)
	config.OIDCAudience = strings.TrimRight(strings.TrimSpace(config.OIDCAudience), "/")
	if config.ProjectID == "" {
		return ListSaveOperationQueueConfig{}, fmt.Errorf("%s is required", listSaveOperationQueueProjectIDEnv)
	}
	if err := validateListSaveOperationQueueIdentifier("projectId", config.ProjectID); err != nil {
		return ListSaveOperationQueueConfig{}, err
	}
	if config.Location == "" {
		return ListSaveOperationQueueConfig{}, fmt.Errorf("%s is required", listSaveOperationQueueLocationEnv)
	}
	if err := validateListSaveOperationQueueIdentifier("location", config.Location); err != nil {
		return ListSaveOperationQueueConfig{}, err
	}
	if config.QueueID == "" {
		return ListSaveOperationQueueConfig{}, fmt.Errorf("%s is required", listSaveOperationQueueIDEnv)
	}
	if err := validateListSaveOperationQueueIdentifier("queueId", config.QueueID); err != nil {
		return ListSaveOperationQueueConfig{}, err
	}
	if config.TargetBaseURL == "" {
		return ListSaveOperationQueueConfig{}, fmt.Errorf("%s is required", listSaveOperationQueueTargetBaseURLEnv)
	}
	parsedTargetURL, err := url.Parse(config.TargetBaseURL)
	if err != nil {
		return ListSaveOperationQueueConfig{}, fmt.Errorf("parse target base URL: %w", err)
	}
	if parsedTargetURL.Scheme != "https" && parsedTargetURL.Scheme != "http" {
		return ListSaveOperationQueueConfig{}, errors.New("target base URL must use http or https")
	}
	if parsedTargetURL.Host == "" {
		return ListSaveOperationQueueConfig{}, errors.New("target base URL host is required")
	}
	if parsedTargetURL.RawQuery != "" || parsedTargetURL.Fragment != "" {
		return ListSaveOperationQueueConfig{}, errors.New("target base URL must not contain a query or fragment")
	}
	if config.ServiceAccountEmail == "" {
		return ListSaveOperationQueueConfig{}, fmt.Errorf("%s is required", listSaveOperationQueueServiceAccountEmailEnv)
	}
	if strings.ContainsAny(config.ServiceAccountEmail, "\r\n\x00") ||
		!strings.Contains(config.ServiceAccountEmail, "@") {
		return ListSaveOperationQueueConfig{}, errors.New("service account email is invalid")
	}
	if config.OIDCAudience == "" {
		config.OIDCAudience = config.TargetBaseURL
	}
	parsedAudience, err := url.Parse(config.OIDCAudience)
	if err != nil {
		return ListSaveOperationQueueConfig{}, fmt.Errorf("parse OIDC audience: %w", err)
	}
	if parsedAudience.Scheme != "https" && parsedAudience.Scheme != "http" {
		return ListSaveOperationQueueConfig{}, errors.New("OIDC audience must use http or https")
	}
	if parsedAudience.Host == "" {
		return ListSaveOperationQueueConfig{}, errors.New("OIDC audience host is required")
	}
	return config, nil
}
func normalizeListSaveOperationOperationID(operationID string) (string, error) {
	operationID = strings.TrimSpace(operationID)
	if operationID == "" {
		return "", errors.New("operationId is required")
	}
	if len(operationID) > 512 {
		return "", errors.New("operationId must not exceed 512 characters")
	}
	if strings.Contains(operationID, "/") ||
		strings.Contains(operationID, "://") ||
		strings.ContainsAny(operationID, "\r\n\x00") {
		return "", errors.New("operationId is invalid")
	}
	return operationID, nil
}
func validateListSaveOperationQueueIdentifier(fieldName string, value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("%s is required", fieldName)
	}
	for _, character := range value {
		switch {
		case character >= 'a' && character <= 'z':
		case character >= 'A' && character <= 'Z':
		case character >= '0' && character <= '9':
		case character == '-':
		case character == '_':
		case character == '.':
		default:
			return fmt.Errorf("%s contains an invalid character", fieldName)
		}
	}
	return nil
}
func buildListSaveOperationRetryTaskID(operationID string, scheduledAt time.Time) string {
	source := operationID + "\x00" + scheduledAt.UTC().Format(time.RFC3339Nano)
	digest := sha256.Sum256([]byte(source))
	return "list-save-retry-" + hex.EncodeToString(digest[:16])
}
func firstNonEmptyEnvironmentValue(names ...string) string {
	for _, name := range names {
		value := strings.TrimSpace(os.Getenv(name))
		if value != "" {
			return value
		}
	}
	return ""
}
