// backend/internal/domain/transportation/resale_flat_rate.go
package transportation

import "errors"

var (
	ErrInvalidResaleBoxSize = errors.New(
		"transportation: invalid resale box size",
	)
)

const (
	ResaleBoxSize60  = 60
	ResaleBoxSize80  = 80
	ResaleBoxSize100 = 100
	ResaleBoxSize120 = 120
	ResaleBoxSize140 = 140
	ResaleBoxSize160 = 160
)

type ResaleFlatRateOption struct {
	Size   int   `json:"size"`
	Amount int64 `json:"amount"`
}

var resaleShippingCarriers = []Carrier{
	CarrierPost,
	CarrierYamato,
}

var resaleFlatRateOptions = []ResaleFlatRateOption{
	{
		Size:   ResaleBoxSize60,
		Amount: 750,
	},
	{
		Size:   ResaleBoxSize80,
		Amount: 850,
	},
	{
		Size:   ResaleBoxSize100,
		Amount: 1050,
	},
	{
		Size:   ResaleBoxSize120,
		Amount: 1200,
	},
	{
		Size:   ResaleBoxSize140,
		Amount: 1450,
	},
	{
		Size:   ResaleBoxSize160,
		Amount: 1700,
	},
}

func ResaleShippingCarriers() []Carrier {
	result := make(
		[]Carrier,
		len(resaleShippingCarriers),
	)

	copy(
		result,
		resaleShippingCarriers,
	)

	return result
}

func ResaleFlatRateOptions() []ResaleFlatRateOption {
	result := make(
		[]ResaleFlatRateOption,
		len(resaleFlatRateOptions),
	)

	copy(
		result,
		resaleFlatRateOptions,
	)

	return result
}

func IsValidResaleShippingCarrier(
	carrier Carrier,
) bool {
	switch carrier {
	case CarrierPost,
		CarrierYamato:
		return true
	default:
		return false
	}
}

func IsValidResaleBoxSize(
	boxSize int,
) bool {
	switch boxSize {
	case ResaleBoxSize60,
		ResaleBoxSize80,
		ResaleBoxSize100,
		ResaleBoxSize120,
		ResaleBoxSize140,
		ResaleBoxSize160:
		return true
	default:
		return false
	}
}

func CalculateResaleFlatRate(
	carrier Carrier,
	boxSize int,
) (Quote, error) {
	if !IsValidResaleShippingCarrier(
		carrier,
	) {
		return Quote{},
			ErrInvalidCarrier
	}

	if !IsValidResaleBoxSize(
		boxSize,
	) {
		return Quote{},
			ErrInvalidResaleBoxSize
	}

	for _, option := range resaleFlatRateOptions {

		if option.Size != boxSize {
			continue
		}

		if option.Amount <
			MinRateAmount {
			return Quote{},
				ErrInvalidRateAmount
		}

		return Quote{
			Carrier: carrier,
			Size:    option.Size,
			Amount:  option.Amount,
		}, nil
	}

	return Quote{},
		ErrInvalidResaleBoxSize
}
