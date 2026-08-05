// backend/internal/adapters/out/firestore/cloudtasks/mint_task_queue.go
package cloudtasks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	cloudtasks "cloud.google.com/go/cloudtasks/apiv2"
	taskspb "cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// MintTaskQueue は MintUsecase から見た MintTaskEnqueuer の Cloud Tasks 実装です。
//
// 役割:
// - mints/{mintID}/products のうち、次の1件だけを処理する internal endpoint を Cloud Tasks に投入する。
// - 1 task = 1 product mint ではなく、1 task = 「mintID の次の実行可能 product を1件処理」です。
// - ExecuteNextMintTask 側が成功後に次 task を enqueue するため、全件が順次処理されます。
type MintTaskQueue struct {
	Client *cloudtasks.Client

	ProjectID string
	Location  string
	QueueID   string

	InternalBaseURL string

	// OIDC token に使う service account。
	// Cloud Run Invoker 権限を持つ service account を指定してください。
	ServiceAccountEmail string

	// 任意。
	// 空なら InternalBaseURL を audience として使います。
	Audience string

	// 任意。
	// 0 の場合は即時実行です。
	DispatchDelay time.Duration
}

type mintTaskPayload struct {
	MintID string `json:"mintId"`
}

// NewMintTaskQueueFromEnv は環境変数から MintTaskQueue を生成します。
//
// 必須:
// - CLOUD_TASKS_PROJECT_ID
// - CLOUD_TASKS_LOCATION
// - CLOUD_TASKS_QUEUE_ID
// - INTERNAL_BASE_URL
// - CLOUD_TASKS_SERVICE_ACCOUNT
//
// 任意:
// - CLOUD_TASKS_AUDIENCE
// - MINT_TASK_DISPATCH_DELAY_SECONDS
func NewMintTaskQueueFromEnv(ctx context.Context) (*MintTaskQueue, error) {
	projectID := strings.TrimSpace(os.Getenv("CLOUD_TASKS_PROJECT_ID"))
	if projectID == "" {
		projectID = strings.TrimSpace(os.Getenv("GCP_PROJECT_ID"))
	}
	if projectID == "" {
		projectID = strings.TrimSpace(os.Getenv("GOOGLE_CLOUD_PROJECT"))
	}
	if projectID == "" {
		return nil, errors.New("CLOUD_TASKS_PROJECT_ID is required")
	}

	location := strings.TrimSpace(os.Getenv("CLOUD_TASKS_LOCATION"))
	if location == "" {
		return nil, errors.New("CLOUD_TASKS_LOCATION is required")
	}

	queueID := strings.TrimSpace(os.Getenv("CLOUD_TASKS_QUEUE_ID"))
	if queueID == "" {
		return nil, errors.New("CLOUD_TASKS_QUEUE_ID is required")
	}

	internalBaseURL := strings.TrimRight(
		strings.TrimSpace(os.Getenv("INTERNAL_BASE_URL")),
		"/",
	)
	if internalBaseURL == "" {
		return nil, errors.New("INTERNAL_BASE_URL is required")
	}

	serviceAccountEmail := strings.TrimSpace(
		os.Getenv("CLOUD_TASKS_SERVICE_ACCOUNT"),
	)
	if serviceAccountEmail == "" {
		return nil, errors.New("CLOUD_TASKS_SERVICE_ACCOUNT is required")
	}

	audience := strings.TrimSpace(os.Getenv("CLOUD_TASKS_AUDIENCE"))
	if audience == "" {
		audience = internalBaseURL
	}

	var dispatchDelay time.Duration

	rawDispatchDelay := strings.TrimSpace(
		os.Getenv("MINT_TASK_DISPATCH_DELAY_SECONDS"),
	)
	if rawDispatchDelay != "" {
		parsed, err := time.ParseDuration(rawDispatchDelay + "s")
		if err != nil {
			return nil, fmt.Errorf(
				"invalid MINT_TASK_DISPATCH_DELAY_SECONDS: %w",
				err,
			)
		}

		if parsed < 0 {
			return nil, errors.New(
				"MINT_TASK_DISPATCH_DELAY_SECONDS must be >= 0",
			)
		}

		dispatchDelay = parsed
	}

	client, err := cloudtasks.NewClient(ctx)
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

	return q.Client.Close()
}

// EnqueueMintTask は mintID の次の product mint を1件進める worker を enqueue します。
//
// 呼び出し先:
//
//	POST {INTERNAL_BASE_URL}/internal/mint/tasks/{mintID}/execute
func (q *MintTaskQueue) EnqueueMintTask(
	ctx context.Context,
	mintID string,
) error {
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

	if q.ProjectID == "" {
		return errors.New("projectID is empty")
	}
	if q.Location == "" {
		return errors.New("location is empty")
	}
	if q.QueueID == "" {
		return errors.New("queueID is empty")
	}
	if q.InternalBaseURL == "" {
		return errors.New("internalBaseURL is empty")
	}
	if q.ServiceAccountEmail == "" {
		return errors.New("serviceAccountEmail is empty")
	}

	parent := fmt.Sprintf(
		"projects/%s/locations/%s/queues/%s",
		q.ProjectID,
		q.Location,
		q.QueueID,
	)

	url := fmt.Sprintf(
		"%s/internal/mint/tasks/%s/execute",
		strings.TrimRight(q.InternalBaseURL, "/"),
		urlPathEscape(id),
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
				Url:        url,
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
				Body: body,
				AuthorizationHeader: &taskspb.HttpRequest_OidcToken{
					OidcToken: &taskspb.OidcToken{
						ServiceAccountEmail: q.ServiceAccountEmail,
						Audience:            q.Audience,
					},
				},
			},
		},
	}

	if q.DispatchDelay > 0 {
		task.ScheduleTime = timestamppb.New(
			time.Now().UTC().Add(q.DispatchDelay),
		)
	}

	req := &taskspb.CreateTaskRequest{
		Parent: parent,
		Task:   task,
	}

	if _, err := q.Client.CreateTask(ctx, req); err != nil {
		return fmt.Errorf(
			"create mint cloud task mintID=%s: %w",
			id,
			err,
		)
	}

	return nil
}

func urlPathEscape(s string) string {
	// mintID / productionID は通常 Firestore docId なので
	// "/" を含めない想定です。
	// 念のため path segment として安全にします。
	return strings.ReplaceAll(strings.TrimSpace(s), "/", "%2F")
}
