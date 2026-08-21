// backend/internal/application/usecase/shipping_quote_usecase.go
package usecase

import (
	"context"

	inventorydom "narratives/internal/domain/inventory"
	listdom "narratives/internal/domain/list"
	modeldom "narratives/internal/domain/model"
	shippingaddressdom "narratives/internal/domain/shippingAddress"
	transportationdom "narratives/internal/domain/transportation"
)

type ShippingQuoteUsecase struct {
	listRepo            listdom.Repository
	inventoryRepo       inventorydom.RepositoryPort
	modelRepo           modeldom.RepositoryPort
	shippingAddressRepo shippingaddressdom.RepositoryPort
	transportationSvc   *transportationdom.Service
}

type ShippingQuoteInput struct {
	UserID                       string
	ListID                       string
	ModelID                      string
	DestinationShippingAddressID string
}

type ShippingQuoteResult struct {
	ListID                       string
	InventoryID                  string
	ModelID                      string
	OriginShippingAddressID      string
	DestinationShippingAddressID string

	TransportationOption listdom.TransportationOption
	TransportationID     string

	Size     int
	Amount   int64
	Currency string
}

func NewShippingQuoteUsecase(
	listRepo listdom.Repository,
	inventoryRepo inventorydom.RepositoryPort,
	modelRepo modeldom.RepositoryPort,
	shippingAddressRepo shippingaddressdom.RepositoryPort,
	transportationSvc *transportationdom.Service,
) *ShippingQuoteUsecase {
	return &ShippingQuoteUsecase{
		listRepo:            listRepo,
		inventoryRepo:       inventoryRepo,
		modelRepo:           modelRepo,
		shippingAddressRepo: shippingAddressRepo,
		transportationSvc:   transportationSvc,
	}
}

func (uc *ShippingQuoteUsecase) Quote(
	ctx context.Context,
	input ShippingQuoteInput,
) (ShippingQuoteResult, error) {
	if uc == nil {
		return ShippingQuoteResult{},
			ErrNotSupported(
				"ShippingQuote.Quote",
			)
	}

	if uc.listRepo == nil {
		return ShippingQuoteResult{},
			ErrNotSupported(
				"ShippingQuote.ListRepo",
			)
	}

	if uc.inventoryRepo == nil {
		return ShippingQuoteResult{},
			ErrNotSupported(
				"ShippingQuote.InventoryRepo",
			)
	}

	if uc.modelRepo == nil {
		return ShippingQuoteResult{},
			ErrNotSupported(
				"ShippingQuote.ModelRepo",
			)
	}

	if uc.shippingAddressRepo == nil {
		return ShippingQuoteResult{},
			ErrNotSupported(
				"ShippingQuote.ShippingAddressRepo",
			)
	}

	if uc.transportationSvc == nil {
		return ShippingQuoteResult{},
			ErrNotSupported(
				"ShippingQuote.TransportationService",
			)
	}

	if input.UserID == "" {
		return ShippingQuoteResult{},
			ErrInvalidArgument(
				"user_id_required",
			)
	}

	if input.ListID == "" {
		return ShippingQuoteResult{},
			ErrInvalidArgument(
				"list_id_required",
			)
	}

	if input.ModelID == "" {
		return ShippingQuoteResult{},
			ErrInvalidArgument(
				"model_id_required",
			)
	}

	if input.DestinationShippingAddressID == "" {
		return ShippingQuoteResult{},
			ErrInvalidArgument(
				"destination_shipping_address_id_required",
			)
	}

	listItem, err :=
		uc.listRepo.GetByID(
			ctx,
			input.ListID,
		)
	if err != nil {
		return ShippingQuoteResult{}, err
	}

	if listItem.InventoryID == "" {
		return ShippingQuoteResult{},
			ErrInvalidArgument(
				"list_inventory_id_required",
			)
	}

	if !listContainsModel(
		listItem.Prices,
		input.ModelID,
	) {
		return ShippingQuoteResult{},
			ErrInvalidArgument(
				"model_not_available_in_list",
			)
	}

	if !listdom.IsValidTransportationOption(
		listItem.TransportationOption,
	) {
		return ShippingQuoteResult{},
			listdom.ErrInvalidTransportationOption
	}

	inventoryItem, err :=
		uc.inventoryRepo.GetByID(
			ctx,
			listItem.InventoryID,
		)
	if err != nil {
		return ShippingQuoteResult{}, err
	}

	if inventoryItem.ShippingAddressID == "" {
		return ShippingQuoteResult{},
			ErrInvalidArgument(
				"inventory_shipping_address_id_required",
			)
	}

	modelItem, err :=
		uc.modelRepo.GetByID(
			ctx,
			input.ModelID,
		)
	if err != nil {
		return ShippingQuoteResult{}, err
	}

	if modelItem == nil {
		return ShippingQuoteResult{},
			modeldom.ErrNotFound
	}

	if modelItem.GetProductBlueprintID() !=
		inventoryItem.ProductBlueprintID {
		return ShippingQuoteResult{},
			ErrInvalidArgument(
				"model_inventory_mismatch",
			)
	}

	shippingPackage :=
		modelItem.GetShippingPackage()

	if err :=
		shippingPackage.Validate(); err != nil {
		return ShippingQuoteResult{}, err
	}

	originAddress, err :=
		uc.shippingAddressRepo.GetByID(
			ctx,
			inventoryItem.ShippingAddressID,
		)
	if err != nil {
		return ShippingQuoteResult{}, err
	}

	if originAddress == nil {
		return ShippingQuoteResult{},
			shippingaddressdom.ErrNotFound
	}

	destinationAddress, err :=
		uc.shippingAddressRepo.GetByUser(
			ctx,
			input.DestinationShippingAddressID,
			input.UserID,
		)
	if err != nil {
		return ShippingQuoteResult{}, err
	}

	if destinationAddress == nil {
		return ShippingQuoteResult{},
			shippingaddressdom.ErrNotFound
	}

	if originAddress.Country !=
		shippingaddressdom.DefaultCountry {
		return ShippingQuoteResult{},
			ErrInvalidArgument(
				"unsupported_origin_country",
			)
	}

	if destinationAddress.Country !=
		shippingaddressdom.DefaultCountry {
		return ShippingQuoteResult{},
			ErrInvalidArgument(
				"unsupported_destination_country",
			)
	}

	quote, err :=
		uc.transportationSvc.Calculate(
			ctx,
			transportationdom.CalculateInput{
				Carrier: transportationdom.Carrier(
					listItem.TransportationOption,
				),

				Package: transportationdom.Package{
					WeightGrams: shippingPackage.WeightGrams,
					WidthMM:     shippingPackage.WidthMM,
					LengthMM:    shippingPackage.LengthMM,
					HeightMM:    shippingPackage.HeightMM,
				},

				Origin: transportationdom.Address{
					Country: originAddress.Country,
					ZipCode: originAddress.ZipCode,
					State:   originAddress.State,
					City:    originAddress.City,
				},

				Destination: transportationdom.Address{
					Country: destinationAddress.Country,
					ZipCode: destinationAddress.ZipCode,
					State:   destinationAddress.State,
					City:    destinationAddress.City,
				},

				CompanyID: originAddress.CompanyID,

				TransportationID: listItem.TransportationID,
			},
		)
	if err != nil {
		return ShippingQuoteResult{}, err
	}

	return ShippingQuoteResult{
		ListID:                       listItem.ID,
		InventoryID:                  inventoryItem.ID,
		ModelID:                      input.ModelID,
		OriginShippingAddressID:      originAddress.ID,
		DestinationShippingAddressID: destinationAddress.ID,

		TransportationOption: listItem.TransportationOption,
		TransportationID:     listItem.TransportationID,

		Size:     quote.Size,
		Amount:   quote.Amount,
		Currency: "JPY",
	}, nil
}

func listContainsModel(
	prices []listdom.ListPriceRow,
	modelID string,
) bool {
	for _, price := range prices {
		if price.ModelID == modelID {
			return true
		}
	}

	return false
}
