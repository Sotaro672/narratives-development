// backend/internal/adapters/out/cloudtasks/mint_task_queue.go
package cloudtasks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	cloudtasksv2 "cloud.google.com/go/cloudtasks/apiv2"
	taskspb "cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	mintTaskQueueProjectIDEnv            = "CLOUD_TASKS_PROJECT_ID"
	mintTaskQueueLocationEnv             = "CLOUD_TASKS_LOCATION"
	mintTaskQueueIDEnv                   = "CLOUD_TASKS_QUEUE_ID"
	mintTaskQueueInternalBaseURLEnv      = "INTERNAL_BASE_URL"
	mintTaskQueueServiceAccountEnv       = "CLOUD_TASKS_SERVICE_ACCOUNT"
	mintTaskQueueAudienceEnv             = "CLOUD_TASKS_AUDIENCE"
	mintTaskQueueDispatchDelaySecondsEnv = "MINT_TASK_DISPATCH_DELAY_SECONDS"
)

type MintTaskQueue struct {
	Client *cloudtasksv2.Client

	ProjectID string
	Location  string
	QueueID   string

	InternalBaseURL string

	// OIDC token に使う service account。
	// Cloud Run Invoker 権限を持つ service account を指定する。
	ServiceAccountEmail string

	// 空の場合は InternalBaseURL を audience として使用する。
	Audience string

	// 0 の場合は即時実行。
	DispatchDelay time.Duration
}

type mintTaskPayload struct {
	MintID string `json:"mintId"`
}

// NewMintTaskQueueFromEnv は環境変数から MintTaskQueue を生成する。
//
// 必須:
//   - CLOUD_TASKS_PROJECT_ID
//   - CLOUD_TASKS_LOCATION
//   - CLOUD_TASKS_QUEUE_ID
//   - INTERNAL_BASE_URL
//   - CLOUD_TASKS_SERVICE_ACCOUNT
//
// 任意:
//   - CLOUD_TASKS_AUDIENCE
//   - MINT_TASK_DISPATCH_DELAY_SECONDS
//
// CLOUD_TASKS_PROJECT_ID が空の場合は、
// GCP_PROJECT_ID / GOOGLE_CLOUD_PROJECT を順に fallback する。
func NewMintTaskQueueFromEnv(ctx context.Context) (*MintTaskQueue, error) {
	if ctx == nil {
		return nil, errors.New("context is nil")
	}

	projectID := mintTaskQueueProjectIDFromEnv()
	if projectID == "" {
		return nil, fmt.Errorf("%s is required", mintTaskQueueProjectIDEnv)
	}

	location := strings.TrimSpace(os.Getenv(mintTaskQueueLocationEnv))
	if location == "" {
		return nil, fmt.Errorf("%s is required", mintTaskQueueLocationEnv)
	}

	queueID := strings.TrimSpace(os.Getenv(mintTaskQueueIDEnv))
	if queueID == "" {
		return nil, fmt.Errorf("%s is required", mintTaskQueueIDEnv)
	}

	internalBaseURL := strings.TrimRight(
		strings.TrimSpace(os.Getenv(mintTaskQueueInternalBaseURLEnv)),
		"/",
	)
	if internalBaseURL == "" {
		return nil, fmt.Errorf("%s is required", mintTaskQueueInternalBaseURLEnv)
	}

	serviceAccountEmail := strings.TrimSpace(
		os.Getenv(mintTaskQueueServiceAccountEnv),
	)
	if serviceAccountEmail == "" {
		return nil, fmt.Errorf("%s is required", mintTaskQueueServiceAccountEnv)
	}

	audience := strings.TrimSpace(os.Getenv(mintTaskQueueAudienceEnv))
	if audience == "" {
		audience = internalBaseURL
	}

	dispatchDelay, err := mintTaskDispatchDelayFromEnv()
	if err != nil {
		return nil, err
	}

	client, err := cloudtasksv2.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("create cloud tasks client: %w", err)
	}

	return &MintTaskQueue{
		Client:              client,
		ProjectID:           projectID,
		Location:            location,
		QueueID:             queueID,
		InternalBaseURL:     internalBaseURL,
		ServiceAccountEmail: serviceAccountEmail,
		Audience:            audience,
		DispatchDelay:       dispatchDelay,
	}, nil
}

// Close closes the underlying Cloud Tasks client.
func (q *MintTaskQueue) Close() error {
	if q == nil || q.Client == nil {
		return nil
	}

	if err := q.Client.Close(); err != nil {
		return fmt.Errorf("close cloud tasks client: %w", err)
	}

	q.Client = nil
	return nil
}

// EnqueueMintTask は mintID の次の product mint を1件進める worker を enqueue する。
//
// 呼び出し先:
//
//	POST {INTERNAL_BASE_URL}/internal/mint/tasks/{mintID}/execute
func (q *MintTaskQueue) EnqueueMintTask(
	ctx context.Context,
	mintID string,
) error {
	if ctx == nil {
		return errors.New("context is nil")
	}
	if q == nil {
		return errors.New("mint task queue is nil")
	}
	if q.Client == nil {
		return errors.New("cloud tasks client is nil")
	}

	id := strings.TrimSpace(mintID)
	if id == "" {
		return errors.New("mintID is empty")
	}
	if strings.ContainsAny(id, "\r\n\x00") {
		return errors.New("mintID is invalid")
	}

	projectID := strings.TrimSpace(q.ProjectID)
	if projectID == "" {
		return errors.New("projectID is empty")
	}

	location := strings.TrimSpace(q.Location)
	if location == "" {
		return errors.New("location is empty")
	}

	queueID := strings.TrimSpace(q.QueueID)
	if queueID == "" {
		return errors.New("queueID is empty")
	}

	internalBaseURL := strings.TrimRight(
		strings.TrimSpace(q.InternalBaseURL),
		"/",
	)
	if internalBaseURL == "" {
		return errors.New("internalBaseURL is empty")
	}

	serviceAccountEmail := strings.TrimSpace(q.ServiceAccountEmail)
	if serviceAccountEmail == "" {
		return errors.New("serviceAccountEmail is empty")
	}

	audience := strings.TrimSpace(q.Audience)
	if audience == "" {
		audience = internalBaseURL
	}

	parent := fmt.Sprintf(
		"projects/%s/locations/%s/queues/%s",
		projectID,
		location,
		queueID,
	)

	targetURL := fmt.Sprintf(
		"%s/internal/mint/tasks/%s/execute",
		internalBaseURL,
		url.PathEscape(id),
	)

	body, err := json.Marshal(mintTaskPayload{
		MintID: id,
	})
	if err != nil {
		return fmt.Errorf("marshal mint task payload: %w", err)
	}

	task := &taskspb.Task{
		MessageType: &taskspb.Task_HttpRequest{
			HttpRequest: &taskspb.HttpRequest{
				HttpMethod: taskspb.HttpMethod_POST,
				Url:        targetURL,
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
				Body: body,
				AuthorizationHeader: &taskspb.HttpRequest_OidcToken{
					OidcToken: &taskspb.OidcToken{
						ServiceAccountEmail: serviceAccountEmail,
						Audience:            audience,
					},
				},
			},
		},
	}

	if q.DispatchDelay > 0 {
		scheduleTime := timestamppb.New(
			time.Now().UTC().Add(q.DispatchDelay),
		)
		if err := scheduleTime.CheckValid(); err != nil {
			return fmt.Errorf("invalid mint task schedule time: %w", err)
		}

		task.ScheduleTime = scheduleTime
	}

	request := &taskspb.CreateTaskRequest{
		Parent: parent,
		Task:   task,
	}

	if _, err := q.Client.CreateTask(ctx, request); err != nil {
		return fmt.Errorf(
			"create mint cloud task mintID=%s: %w",
			id,
			err,
		)
	}

	return nil
}

func mintTaskQueueProjectIDFromEnv() string {
	for _, name := range []string{
		mintTaskQueueProjectIDEnv,
		"GCP_PROJECT_ID",
		"GOOGLE_CLOUD_PROJECT",
	} {
		value := strings.TrimSpace(os.Getenv(name))
		if value != "" {
			return value
		}
	}

	return ""
}

func mintTaskDispatchDelayFromEnv() (time.Duration, error) {
	raw := strings.TrimSpace(
		os.Getenv(mintTaskQueueDispatchDelaySecondsEnv),
	)
	if raw == "" {
		return 0, nil
	}

	dispatchDelay, err := time.ParseDuration(raw + "s")
	if err != nil {
		return 0, fmt.Errorf(
			"invalid %s: %w",
			mintTaskQueueDispatchDelaySecondsEnv,
			err,
		)
	}

	if dispatchDelay < 0 {
		return 0, fmt.Errorf(
			"%s must be >= 0",
			mintTaskQueueDispatchDelaySecondsEnv,
		)
	}

	return dispatchDelay, nil
}
