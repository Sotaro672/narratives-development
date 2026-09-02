// backend/internal/application/usecase/bank_payout_usecase.go
package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	applicationport "narratives/internal/application/port"
	bankpayoutdom "narratives/internal/domain/bankPayout"
	payoutdom "narratives/internal/domain/payoutAccount"
	salesreceivabledom "narratives/internal/domain/salesReceivable"
)

var (
	ErrBankPayoutRepositoryMissing = errors.New(
		"bankPayout: repository is not configured",
	)
	ErrBankPayoutSalesReceivableRepositoryMissing = errors.New(
		"bankPayout: sales receivable repository is not configured",
	)
	ErrBankPayoutAccountServiceMissing = errors.New(
		"bankPayout: payout account service is not configured",
	)
	ErrBankPayoutGatewayMissing = errors.New(
		"bankPayout: gateway is not configured",
	)
	ErrBankPayoutClockMissing = errors.New(
		"bankPayout: clock is not configured",
	)
	ErrBankPayoutReceivableNotAvailable = errors.New(
		"bankPayout: sales receivable is not available for payout",
	)
	ErrBankPayoutReceivableMismatch = errors.New(
		"bankPayout: sales receivable does not match bank payout",
	)
	ErrBankPayoutAccountMismatch = errors.New(
		"bankPayout: payout account does not match sales receivable",
	)
	ErrBankPayoutExistingMismatch = errors.New(
		"bankPayout: existing bank payout does not match sales receivable",
	)
	ErrBankPayoutGatewayResultInvalid = errors.New(
		"bankPayout: gateway returned an invalid result",
	)
	ErrBankPayoutTerminalFailed = errors.New(
		"bankPayout: payout has failed permanently",
	)
)

// BankPayoutAccountService contains only the payout-account operations required
// by BankPayoutUsecase.
//
// DecryptAccountNumber must return plaintext only for the immediate gateway
// request. The plaintext value must never be persisted, logged, or returned to
// an HTTP client.
type BankPayoutAccountService interface {
	GetByUserID(
		ctx context.Context,
		userID string,
	) (*payoutdom.PayoutAccount, error)

	DecryptAccountNumber(
		ctx context.Context,
		userID string,
		accountNumberCiphertext string,
	) (string, error)
}

// BankPayoutUsecase creates and executes one seller bank payout for one
// item-level resale SalesReceivable.
//
// Current Fake-payout flow:
//
//	SalesReceivable available
//		-> BankPayout pending
//		-> SalesReceivable reserved
//		-> BankPayout processing
//		-> BankPayoutGateway.Execute
//		-> BankPayout paid
//		-> SalesReceivable paid
//
// Idempotency:
//   - BankPayout ID is deterministic from SalesReceivableID.
//   - one SalesReceivable therefore creates at most one BankPayout.
//   - the same BankPayout ID is also used as the gateway idempotency key.
//   - retries always reuse the already-snapshotted bank destination.
//   - a paid BankPayout never executes the gateway again.
//   - if BankPayout becomes paid before SalesReceivable is marked paid, a retry
//     reconciles the SalesReceivable without re-executing the gateway.
//
// For the current Fake gateway, BankPayout processing is synchronous. A future
// asynchronous real-bank provider should introduce an explicit provider
// submitted/pending state instead of treating provider acceptance as paid.
type BankPayoutUsecase struct {
	payoutRepo     bankpayoutdom.Repository
	receivableRepo salesreceivabledom.Repository
	accountService BankPayoutAccountService
	gateway        applicationport.BankPayoutGateway
	now            func() time.Time
}

func NewBankPayoutUsecase(
	payoutRepo bankpayoutdom.Repository,
	receivableRepo salesreceivabledom.Repository,
	accountService BankPayoutAccountService,
	gateway applicationport.BankPayoutGateway,
) *BankPayoutUsecase {
	return &BankPayoutUsecase{
		payoutRepo:     payoutRepo,
		receivableRepo: receivableRepo,
		accountService: accountService,
		gateway:        gateway,
		now:            time.Now,
	}
}

// GetByID returns one persisted BankPayout.
func (u *BankPayoutUsecase) GetByID(
	ctx context.Context,
	payoutID string,
) (*bankpayoutdom.BankPayout, error) {
	if u == nil || u.payoutRepo == nil {
		return nil, ErrBankPayoutRepositoryMissing
	}
	if payoutID == "" || strings.Contains(payoutID, "/") {
		return nil, bankpayoutdom.ErrInvalidID
	}

	payout, err := u.payoutRepo.GetByID(ctx, payoutID)
	if err != nil {
		return nil, err
	}
	if payout.ID != payoutID {
		return nil, bankpayoutdom.ErrInvalidID
	}
	if err := payout.Validate(); err != nil {
		return nil, err
	}

	return &payout, nil
}

// ExecuteForSalesReceivable ensures and executes the BankPayout belonging to one
// payout-eligible SalesReceivable.
//
// The method is safe to call repeatedly for the same receivable.
//
// Status handling:
//   - available: create/ensure BankPayout, reserve receivable, execute payout
//   - reserved: resume the existing BankPayout
//   - paid: return the already-paid BankPayout
//   - pending/canceled: reject payout execution
func (u *BankPayoutUsecase) ExecuteForSalesReceivable(
	ctx context.Context,
	receivableID string,
) (*bankpayoutdom.BankPayout, error) {
	if err := u.validateReady(); err != nil {
		return nil, err
	}
	if receivableID == "" || strings.Contains(receivableID, "/") {
		return nil, salesreceivabledom.ErrInvalidID
	}

	receivable, err := u.receivableRepo.GetByID(ctx, receivableID)
	if err != nil {
		return nil, err
	}
	if receivable.ID != receivableID {
		return nil, salesreceivabledom.ErrInvalidID
	}
	if err := receivable.Validate(); err != nil {
		return nil, err
	}

	switch receivable.Status {
	case salesreceivabledom.StatusAvailable:
		payout, err := u.ensureBankPayout(ctx, receivable)
		if err != nil {
			return nil, err
		}

		reserved, err := u.ensureReceivableReserved(
			ctx,
			receivable,
			payout.ID,
		)
		if err != nil {
			return nil, err
		}

		return u.executeOrResume(
			ctx,
			reserved,
			payout,
		)

	case salesreceivabledom.StatusReserved:
		if receivable.BankPayoutID == "" {
			return nil, ErrBankPayoutReceivableMismatch
		}

		payout, err := u.payoutRepo.GetByID(
			ctx,
			receivable.BankPayoutID,
		)
		if err != nil {
			return nil, err
		}
		if err := validateBankPayoutReceivable(
			payout,
			receivable,
		); err != nil {
			return nil, err
		}

		return u.executeOrResume(
			ctx,
			receivable,
			payout,
		)

	case salesreceivabledom.StatusPaid:
		if receivable.BankPayoutID == "" {
			return nil, ErrBankPayoutReceivableMismatch
		}

		payout, err := u.payoutRepo.GetByID(
			ctx,
			receivable.BankPayoutID,
		)
		if err != nil {
			return nil, err
		}
		if err := validateBankPayoutReceivable(
			payout,
			receivable,
		); err != nil {
			return nil, err
		}
		if payout.Status != bankpayoutdom.StatusPaid {
			return nil, ErrBankPayoutReceivableMismatch
		}

		return &payout, nil

	case salesreceivabledom.StatusPending,
		salesreceivabledom.StatusCanceled:
		return nil, ErrBankPayoutReceivableNotAvailable

	default:
		return nil, salesreceivabledom.ErrInvalidStatus
	}
}

// ensureBankPayout returns the deterministic BankPayout for an available
// SalesReceivable.
//
// Existing BankPayouts are always reused. The current PayoutAccount is resolved
// only when the BankPayout does not exist yet. Once created, all retries use the
// immutable BankDestination snapshot stored in the BankPayout.
func (u *BankPayoutUsecase) ensureBankPayout(
	ctx context.Context,
	receivable salesreceivabledom.SalesReceivable,
) (bankpayoutdom.BankPayout, error) {
	if receivable.Status != salesreceivabledom.StatusAvailable {
		return bankpayoutdom.BankPayout{},
			ErrBankPayoutReceivableNotAvailable
	}
	if err := receivable.Validate(); err != nil {
		return bankpayoutdom.BankPayout{}, err
	}

	existing, err := u.payoutRepo.GetBySalesReceivableID(
		ctx,
		receivable.ID,
	)
	if err == nil {
		if err := validateBankPayoutReceivable(
			existing,
			receivable,
		); err != nil {
			return bankpayoutdom.BankPayout{}, err
		}

		return existing, nil
	}
	if !errors.Is(err, bankpayoutdom.ErrNotFound) {
		return bankpayoutdom.BankPayout{}, err
	}

	account, err := u.accountService.GetByUserID(
		ctx,
		receivable.UserID,
	)
	if err != nil {
		return bankpayoutdom.BankPayout{}, err
	}
	if account == nil {
		return bankpayoutdom.BankPayout{},
			ErrBankPayoutAccountMismatch
	}
	if err := account.Validate(); err != nil {
		return bankpayoutdom.BankPayout{}, err
	}

	// PayoutAccount currently uses userId as its canonical document identity.
	// SalesReceivable enforces PayoutAccountID == UserID, so the account resolved
	// here must represent that same payout-account identity.
	if account.UserID != receivable.UserID ||
		receivable.PayoutAccountID != account.UserID {
		return bankpayoutdom.BankPayout{},
			ErrBankPayoutAccountMismatch
	}

	payoutID, err := bankpayoutdom.NewID(receivable.ID)
	if err != nil {
		return bankpayoutdom.BankPayout{}, err
	}

	destination := bankpayoutdom.BankDestinationSnapshot{
		BankCode:   account.BankCode,
		BankName:   account.BankName,
		BranchCode: account.BranchCode,
		BranchName: account.BranchName,

		AccountType:             account.AccountType,
		AccountNumberCiphertext: account.AccountNumberCiphertext,
		BankLast4:               account.BankLast4,
		AccountHolderName:       account.AccountHolderName,
	}

	// AvailableAt is deterministic for the receivable and occurs immediately
	// before payout creation in the current flow. Reusing it here also ensures
	// concurrent CreateIfAbsent candidates carry the same immutable CreatedAt.
	if receivable.AvailableAt == nil ||
		receivable.AvailableAt.IsZero() {
		return bankpayoutdom.BankPayout{},
			ErrBankPayoutReceivableNotAvailable
	}
	createdAt := receivable.AvailableAt.UTC()

	candidate, err := bankpayoutdom.New(
		payoutID,
		receivable.ID,
		receivable.OrderID,
		receivable.PaymentID,
		receivable.OrderItemIndex,
		receivable.ResaleID,
		receivable.UserID,
		receivable.PayoutAccountID,
		destination,
		receivable.ReceivableAmount,
		receivable.Currency,
		createdAt,
	)
	if err != nil {
		return bankpayoutdom.BankPayout{}, err
	}

	persisted, _, err := u.payoutRepo.CreateIfAbsent(
		ctx,
		candidate,
	)
	if err != nil {
		return bankpayoutdom.BankPayout{}, err
	}

	if err := validateBankPayoutReceivable(
		persisted,
		receivable,
	); err != nil {
		return bankpayoutdom.BankPayout{}, err
	}
	if err := validateBankPayoutCreationIdentity(
		persisted,
		candidate,
	); err != nil {
		return bankpayoutdom.BankPayout{}, err
	}

	return persisted, nil
}

// ensureReceivableReserved assigns the available SalesReceivable to the
// deterministic BankPayout.
//
// If a concurrent invocation already reserved the same receivable for the same
// BankPayout, the persisted reserved state is accepted.
func (u *BankPayoutUsecase) ensureReceivableReserved(
	ctx context.Context,
	receivable salesreceivabledom.SalesReceivable,
	payoutID string,
) (salesreceivabledom.SalesReceivable, error) {
	switch receivable.Status {
	case salesreceivabledom.StatusAvailable:
		next := receivable

		if err := next.Reserve(
			payoutID,
			u.now().UTC(),
		); err != nil {
			return salesreceivabledom.SalesReceivable{}, err
		}

		updated, err := u.receivableRepo.Update(ctx, next)
		if err == nil {
			return updated, nil
		}

		// A concurrent retry may have completed the same transition between our
		// read and update. Re-read and accept only the same payout assignment.
		current, getErr := u.receivableRepo.GetByID(
			ctx,
			receivable.ID,
		)
		if getErr != nil {
			return salesreceivabledom.SalesReceivable{}, err
		}

		switch current.Status {
		case salesreceivabledom.StatusReserved,
			salesreceivabledom.StatusPaid:
			if current.BankPayoutID == payoutID {
				return current, nil
			}
		}

		return salesreceivabledom.SalesReceivable{}, err

	case salesreceivabledom.StatusReserved,
		salesreceivabledom.StatusPaid:
		if receivable.BankPayoutID != payoutID {
			return salesreceivabledom.SalesReceivable{},
				ErrBankPayoutReceivableMismatch
		}

		return receivable, nil

	default:
		return salesreceivabledom.SalesReceivable{},
			ErrBankPayoutReceivableNotAvailable
	}
}

func (u *BankPayoutUsecase) executeOrResume(
	ctx context.Context,
	receivable salesreceivabledom.SalesReceivable,
	payout bankpayoutdom.BankPayout,
) (*bankpayoutdom.BankPayout, error) {
	if err := validateBankPayoutReceivable(
		payout,
		receivable,
	); err != nil {
		return nil, err
	}

	switch payout.Status {
	case bankpayoutdom.StatusPaid:
		if err := u.ensureReceivablePaid(
			ctx,
			receivable.ID,
			payout.ID,
			requiredPaidAt(payout),
		); err != nil {
			return &payout, err
		}

		return &payout, nil

	case bankpayoutdom.StatusFailed:
		return &payout, ErrBankPayoutTerminalFailed

	case bankpayoutdom.StatusPending,
		bankpayoutdom.StatusFailedRetryable:
		next := payout

		if err := next.StartProcessing(
			u.now().UTC(),
		); err != nil {
			return nil, err
		}

		updated, err := u.payoutRepo.Update(ctx, next)
		if err != nil {
			return nil, err
		}

		payout = updated

	case bankpayoutdom.StatusProcessing:
		// A processing payout may be retried with the same deterministic
		// idempotency key. The gateway contract guarantees that doing so must not
		// intentionally create a second payout.
		//
		// This also recovers the case where the gateway succeeded but the process
		// failed before the provider result was persisted.
	default:
		return nil, bankpayoutdom.ErrInvalidStatus
	}

	accountNumber, err := u.accountService.DecryptAccountNumber(
		ctx,
		payout.SellerUserID,
		payout.BankDestination.AccountNumberCiphertext,
	)
	if err != nil {
		failed, recordErr := u.recordProcessingFailure(
			ctx,
			payout,
			true,
			"decryption_error",
			"account_number_decryption_failed",
			err.Error(),
		)
		if recordErr != nil {
			return nil, recordErr
		}

		return &failed, err
	}

	result, err := u.gateway.Execute(
		ctx,
		applicationport.ExecuteBankPayoutInput{
			BankPayoutID:   payout.ID,
			IdempotencyKey: payout.ID,

			Amount:   payout.Amount,
			Currency: payout.Currency,

			BankCode:   payout.BankDestination.BankCode,
			BankName:   payout.BankDestination.BankName,
			BranchCode: payout.BankDestination.BranchCode,
			BranchName: payout.BankDestination.BranchName,

			AccountType:       payout.BankDestination.AccountType,
			AccountNumber:     accountNumber,
			BankLast4:         payout.BankDestination.BankLast4,
			AccountHolderName: payout.BankDestination.AccountHolderName,
		},
	)
	if err != nil {
		retryable, errorType, errorCode, errorMsg :=
			bankPayoutGatewayFailureDetails(err)

		failed, recordErr := u.recordProcessingFailure(
			ctx,
			payout,
			retryable,
			errorType,
			errorCode,
			errorMsg,
		)
		if recordErr != nil {
			return nil, recordErr
		}

		return &failed, err
	}

	if result == nil ||
		result.ProviderPayoutID == "" ||
		result.PaidAt.IsZero() {
		failed, recordErr := u.recordProcessingFailure(
			ctx,
			payout,
			true,
			"invalid_response",
			"invalid_gateway_result",
			ErrBankPayoutGatewayResultInvalid.Error(),
		)
		if recordErr != nil {
			return nil, recordErr
		}

		return &failed, ErrBankPayoutGatewayResultInvalid
	}

	paid := payout
	if err := paid.MarkPaid(
		result.ProviderPayoutID,
		result.PaidAt.UTC(),
	); err != nil {
		return nil, err
	}

	persisted, err := u.payoutRepo.Update(
		ctx,
		paid,
	)
	if err != nil {
		return nil, err
	}

	if err := u.ensureReceivablePaid(
		ctx,
		receivable.ID,
		persisted.ID,
		result.PaidAt.UTC(),
	); err != nil {
		// Returning the paid BankPayout together with the reconciliation error is
		// intentional. A later retry sees StatusPaid and only completes the
		// SalesReceivable reserved -> paid transition; the gateway is not called
		// again.
		return &persisted, err
	}

	return &persisted, nil
}

// ensureReceivablePaid reconciles the SalesReceivable after successful payout.
//
// It is idempotent for an already-paid receivable assigned to the same
// BankPayout.
func (u *BankPayoutUsecase) ensureReceivablePaid(
	ctx context.Context,
	receivableID string,
	payoutID string,
	paidAt time.Time,
) error {
	current, err := u.receivableRepo.GetByID(
		ctx,
		receivableID,
	)
	if err != nil {
		return err
	}
	if current.BankPayoutID != payoutID {
		return ErrBankPayoutReceivableMismatch
	}

	switch current.Status {
	case salesreceivabledom.StatusReserved:
		if err := current.MarkPaid(paidAt); err != nil {
			return err
		}

		_, err := u.receivableRepo.Update(
			ctx,
			current,
		)
		if err == nil {
			return nil
		}

		// If a concurrent invocation completed the same state transition, accept
		// the resulting paid state.
		latest, getErr := u.receivableRepo.GetByID(
			ctx,
			receivableID,
		)
		if getErr != nil {
			return err
		}
		if latest.Status == salesreceivabledom.StatusPaid &&
			latest.BankPayoutID == payoutID {
			return nil
		}

		return err

	case salesreceivabledom.StatusPaid:
		return nil

	default:
		return ErrBankPayoutReceivableMismatch
	}
}

func (u *BankPayoutUsecase) recordProcessingFailure(
	ctx context.Context,
	payout bankpayoutdom.BankPayout,
	retryable bool,
	errorType string,
	errorCode string,
	errorMsg string,
) (bankpayoutdom.BankPayout, error) {
	if payout.Status != bankpayoutdom.StatusProcessing {
		return bankpayoutdom.BankPayout{},
			bankpayoutdom.ErrInvalidStatusTransition
	}

	errorTypePtr := optionalFailureString(
		errorType,
		bankpayoutdom.MaxErrorTypeLength,
	)
	errorCodePtr := optionalFailureString(
		errorCode,
		bankpayoutdom.MaxErrorCodeLength,
	)
	errorMsgPtr := optionalFailureString(
		errorMsg,
		bankpayoutdom.MaxErrorMsgLength,
	)

	// Domain failure states require at least one non-empty failure reason.
	if errorTypePtr == nil &&
		errorCodePtr == nil &&
		errorMsgPtr == nil {
		errorTypePtr = stringPointer("bank_payout_error")
	}

	next := payout
	now := u.now().UTC()

	var err error
	if retryable {
		err = next.MarkFailedRetryable(
			errorTypePtr,
			errorCodePtr,
			errorMsgPtr,
			now,
		)
	} else {
		err = next.MarkFailed(
			errorTypePtr,
			errorCodePtr,
			errorMsgPtr,
			now,
		)
	}
	if err != nil {
		return bankpayoutdom.BankPayout{}, err
	}

	updated, err := u.payoutRepo.Update(
		ctx,
		next,
	)
	if err != nil {
		return bankpayoutdom.BankPayout{}, err
	}

	return updated, nil
}

func validateBankPayoutReceivable(
	payout bankpayoutdom.BankPayout,
	receivable salesreceivabledom.SalesReceivable,
) error {
	if err := payout.Validate(); err != nil {
		return err
	}
	if err := receivable.Validate(); err != nil {
		return err
	}

	expectedPayoutID, err := bankpayoutdom.NewID(
		receivable.ID,
	)
	if err != nil {
		return err
	}

	if payout.ID != expectedPayoutID ||
		payout.SalesReceivableID != receivable.ID ||
		payout.OrderID != receivable.OrderID ||
		payout.PaymentID != receivable.PaymentID ||
		payout.OrderItemIndex != receivable.OrderItemIndex ||
		payout.ResaleID != receivable.ResaleID ||
		payout.SellerUserID != receivable.UserID ||
		payout.PayoutAccountID != receivable.PayoutAccountID ||
		payout.Amount != receivable.ReceivableAmount ||
		payout.Currency != receivable.Currency {
		return ErrBankPayoutExistingMismatch
	}

	if receivable.Status == salesreceivabledom.StatusReserved ||
		receivable.Status == salesreceivabledom.StatusPaid {
		if receivable.BankPayoutID != payout.ID {
			return ErrBankPayoutReceivableMismatch
		}
	}

	return nil
}

func validateBankPayoutCreationIdentity(
	actual bankpayoutdom.BankPayout,
	expected bankpayoutdom.BankPayout,
) error {
	if actual.ID != expected.ID ||
		actual.SalesReceivableID != expected.SalesReceivableID ||
		actual.OrderID != expected.OrderID ||
		actual.PaymentID != expected.PaymentID ||
		actual.OrderItemIndex != expected.OrderItemIndex ||
		actual.ResaleID != expected.ResaleID ||
		actual.SellerUserID != expected.SellerUserID ||
		actual.PayoutAccountID != expected.PayoutAccountID ||
		actual.Amount != expected.Amount ||
		actual.Currency != expected.Currency ||
		!bankPayoutDestinationEqual(
			actual.BankDestination,
			expected.BankDestination,
		) ||
		!actual.CreatedAt.Equal(expected.CreatedAt) {
		return ErrBankPayoutExistingMismatch
	}

	return nil
}

func bankPayoutDestinationEqual(
	left bankpayoutdom.BankDestinationSnapshot,
	right bankpayoutdom.BankDestinationSnapshot,
) bool {
	return left.BankCode == right.BankCode &&
		left.BankName == right.BankName &&
		left.BranchCode == right.BranchCode &&
		left.BranchName == right.BranchName &&
		left.AccountType == right.AccountType &&
		left.AccountNumberCiphertext ==
			right.AccountNumberCiphertext &&
		left.BankLast4 == right.BankLast4 &&
		left.AccountHolderName == right.AccountHolderName
}

func requiredPaidAt(
	payout bankpayoutdom.BankPayout,
) time.Time {
	if payout.PaidAt == nil {
		return time.Time{}
	}

	return payout.PaidAt.UTC()
}

func bankPayoutGatewayFailureDetails(
	err error,
) (
	retryable bool,
	errorType string,
	errorCode string,
	errorMsg string,
) {
	if err == nil {
		return false, "", "", ""
	}

	var gatewayErr applicationport.BankPayoutGatewayError
	if errors.As(err, &gatewayErr) {
		errorType = strings.TrimSpace(
			gatewayErr.ErrorType(),
		)
		errorCode = strings.TrimSpace(
			gatewayErr.ErrorCode(),
		)
		errorMsg = strings.TrimSpace(
			gatewayErr.Error(),
		)

		if errorType == "" {
			errorType = "gateway_error"
		}

		return gatewayErr.Retryable(),
			errorType,
			errorCode,
			errorMsg
	}

	return true,
		"gateway_error",
		"",
		strings.TrimSpace(err.Error())
}

func optionalFailureString(
	value string,
	maxLength int,
) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	value = truncateRunes(
		value,
		maxLength,
	)

	return &value
}

func truncateRunes(
	value string,
	maxLength int,
) string {
	if maxLength <= 0 {
		return value
	}

	runes := []rune(value)
	if len(runes) <= maxLength {
		return value
	}

	return string(runes[:maxLength])
}

func stringPointer(
	value string,
) *string {
	return &value
}

func (u *BankPayoutUsecase) validateReady() error {
	if u == nil || u.payoutRepo == nil {
		return ErrBankPayoutRepositoryMissing
	}
	if u.receivableRepo == nil {
		return ErrBankPayoutSalesReceivableRepositoryMissing
	}
	if u.accountService == nil {
		return ErrBankPayoutAccountServiceMissing
	}
	if u.gateway == nil {
		return ErrBankPayoutGatewayMissing
	}
	if u.now == nil {
		return ErrBankPayoutClockMissing
	}

	return nil
}
