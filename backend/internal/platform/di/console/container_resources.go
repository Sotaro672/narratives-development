// backend/internal/platform/di/console/container_resources.go
package console

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"sync"
)

// containerResource は Container が所有する closeable resource を表します。
//
// name は Close 失敗時のエラー識別に使用します。
type containerResource struct {
	name   string
	closer io.Closer
}

// containerResources は Console Container が所有する runtime resource の
// lifecycle を一元管理します。
//
// Resource は生成順に Add し、Close 時には依存関係を考慮して
// 逆順に Close します。
//
// Close は idempotent です。同じ containerResources に対して複数回
// Close を呼んでも、各 resource の Close は一度だけ実行されます。
type containerResources struct {
	mu sync.Mutex

	resources []containerResource
	closed    bool
}

func newContainerResources() *containerResources {
	return &containerResources{
		resources: make(
			[]containerResource,
			0,
		),
	}
}

// Add は Container が所有する resource を登録します。
//
// nil resource は無視します。
//
// 既に containerResources が Close 済みの場合は、resource leak を
// 防止するため追加された resource を即時 Close します。
func (r *containerResources) Add(
	name string,
	closer io.Closer,
) {
	if r == nil ||
		isNilContainerCloser(closer) {
		return
	}

	name = strings.TrimSpace(name)
	if name == "" {
		name = "unnamed resource"
	}

	r.mu.Lock()

	if r.closed {
		r.mu.Unlock()

		_ = closer.Close()

		return
	}

	r.resources = append(
		r.resources,
		containerResource{
			name:   name,
			closer: closer,
		},
	)

	r.mu.Unlock()
}

// Close は登録済み resource を登録順の逆順で Close します。
//
// すべての resource に対して Close を試行し、一部が失敗しても
// 後続 resource の Close を継続します。
//
// 複数の Close error は errors.Join で集約して返します。
func (r *containerResources) Close() error {
	if r == nil {
		return nil
	}

	r.mu.Lock()

	if r.closed {
		r.mu.Unlock()

		return nil
	}

	r.closed = true

	resources := append(
		[]containerResource(nil),
		r.resources...,
	)

	r.resources = nil

	r.mu.Unlock()

	var closeErrors []error

	for i := len(resources) - 1; i >= 0; i-- {
		resource := resources[i]

		if isNilContainerCloser(
			resource.closer,
		) {
			continue
		}

		if err :=
			resource.closer.Close(); err != nil {
			closeErrors = append(
				closeErrors,
				fmt.Errorf(
					"di.console: close %s: %w",
					resource.name,
					err,
				),
			)
		}
	}

	return errors.Join(
		closeErrors...,
	)
}

// CloseWithError は元の error と resource cleanup error を結合します。
//
// Container 構築途中で失敗した場合:
//
//	return nil, resources.CloseWithError(err)
//
// のように使用することで、元の初期化 error を失わずに
// cleanup error も返せます。
func (r *containerResources) CloseWithError(
	err error,
) error {
	if r == nil {
		return err
	}

	return errors.Join(
		err,
		r.Close(),
	)
}

// isNilContainerCloser は interface に格納された typed nil pointer も
// nil resource として扱います。
//
// 例えば:
//
//	var queue *SomeQueue = nil
//	var closer io.Closer = queue
//
// の場合、closer != nil ですが実体は nil のため、そのまま Close を
// 呼ぶと実装によっては panic する可能性があります。
func isNilContainerCloser(
	closer io.Closer,
) bool {
	if closer == nil {
		return true
	}

	value := reflect.ValueOf(
		closer,
	)

	switch value.Kind() {
	case reflect.Chan,
		reflect.Func,
		reflect.Interface,
		reflect.Map,
		reflect.Ptr,
		reflect.Slice:
		return value.IsNil()

	default:
		return false
	}
}
