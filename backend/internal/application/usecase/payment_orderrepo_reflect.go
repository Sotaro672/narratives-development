// backend/internal/application/usecase/payment_orderrepo_reflect.go
package usecase

/*
責任と機能:
- PaymentUsecase が orderRepo を any として受け取る設計を維持しつつ、
  "GetByID / Get / FindByID" のどれかを reflection で呼び出して order を取得する。
- orderRepo の具体型（アダプタ/実装）への依存を避け、DI の自由度を保つ。
*/

import (
	"context"
	"errors"
	"reflect"
)

func callOrderGetByIDBestEffort(orderRepo any, ctx context.Context, orderID string) (any, error) {
	if orderRepo == nil {
		return nil, errors.New("order_repo_not_initialized")
	}

	rv := reflect.ValueOf(orderRepo)
	if !rv.IsValid() {
		return nil, errors.New("order_repo_not_initialized")
	}

	// try methods in order
	methodNames := []string{"GetByID", "Get", "FindByID"}

	var m reflect.Value
	for _, name := range methodNames {
		m = rv.MethodByName(name)
		if m.IsValid() {
			break
		}
		// if value receiver not found, try addressable
		if rv.Kind() != reflect.Pointer && rv.CanAddr() {
			m = rv.Addr().MethodByName(name)
			if m.IsValid() {
				break
			}
		}
	}

	if !m.IsValid() {
		return nil, errors.New("order_repo_missing_method_GetByID_or_equivalent")
	}

	// signature: (context.Context, string) (T, error)
	if m.Type().NumIn() != 2 || m.Type().NumOut() != 2 {
		return nil, errors.New("order_repo_invalid_signature")
	}

	outs := m.Call([]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(orderID)})
	if len(outs) != 2 {
		return nil, errors.New("order_repo_invalid_signature")
	}

	var err error
	// outs[1] should be error (interface), may be nil
	if outs[1].IsValid() && outs[1].Kind() == reflect.Interface && !outs[1].IsNil() {
		if e, ok := outs[1].Interface().(error); ok {
			err = e
		} else {
			err = errors.New("order_repo_returned_non_error")
		}
	}

	return outs[0].Interface(), err
}
