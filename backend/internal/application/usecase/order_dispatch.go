// backend/internal/application/usecase/order_dispatch.go
package usecase

import (
	"context"

	orderdom "narratives/internal/domain/order"
)

type DispatchOrderItemsInput struct {
	ID string

	AllowedInventoryIDs map[string]struct{}
}

type DispatchOrderItemsResult struct {
	Order orderdom.Order

	TargetItems []orderdom.OrderItemSnapshot
	Changed     bool
}

func (u *OrderUsecase) PrepareDispatchItems(
	ctx context.Context,
	in DispatchOrderItemsInput,
) (DispatchOrderItemsResult, error) {
	if in.ID == "" {
		return DispatchOrderItemsResult{},
			orderdom.ErrInvalidID
	}

	order, err := u.repo.GetByID(
		ctx,
		in.ID,
	)
	if err != nil {
		return DispatchOrderItemsResult{}, err
	}

	targetItems := make(
		[]orderdom.OrderItemSnapshot,
		0,
		len(order.Items),
	)

	for _, item := range order.Items {
		if item.Type != orderdom.OrderItemTypeList {
			continue
		}

		if _, ok :=
			in.AllowedInventoryIDs[item.InventoryID]; !ok {
			continue
		}

		if item.IsCancelled || item.IsReturnRequested {
			continue
		}

		targetItems = append(
			targetItems,
			item,
		)
	}

	if len(targetItems) == 0 {
		return DispatchOrderItemsResult{},
			orderdom.ErrNotFound
	}

	return DispatchOrderItemsResult{
		Order:       order,
		TargetItems: targetItems,
		Changed:     false,
	}, nil
}

func (u *OrderUsecase) DispatchItems(
	ctx context.Context,
	in DispatchOrderItemsInput,
) (DispatchOrderItemsResult, error) {
	if in.ID == "" {
		return DispatchOrderItemsResult{},
			orderdom.ErrInvalidID
	}

	order, err := u.repo.GetByID(
		ctx,
		in.ID,
	)
	if err != nil {
		return DispatchOrderItemsResult{}, err
	}

	if !order.Paid {
		return DispatchOrderItemsResult{},
			orderdom.ErrConflict
	}

	targetItems := make(
		[]orderdom.OrderItemSnapshot,
		0,
		len(order.Items),
	)
	changed := false

	for index := range order.Items {
		item := order.Items[index]

		if item.Type != orderdom.OrderItemTypeList {
			continue
		}

		if _, ok :=
			in.AllowedInventoryIDs[item.InventoryID]; !ok {
			continue
		}

		if item.IsCancelled || item.IsReturnRequested {
			continue
		}

		if !item.IsDispatched {
			if err := order.UpdateItemDispatched(
				index,
				true,
			); err != nil {
				return DispatchOrderItemsResult{}, err
			}

			changed = true
		}

		targetItems = append(
			targetItems,
			order.Items[index],
		)
	}

	if len(targetItems) == 0 {
		return DispatchOrderItemsResult{},
			orderdom.ErrNotFound
	}

	if changed {
		updated, err := u.repo.Update(
			ctx,
			order,
			nil,
		)
		if err != nil {
			return DispatchOrderItemsResult{}, err
		}

		order = updated
	}

	return DispatchOrderItemsResult{
		Order:       order,
		TargetItems: targetItems,
		Changed:     changed,
	}, nil
}
