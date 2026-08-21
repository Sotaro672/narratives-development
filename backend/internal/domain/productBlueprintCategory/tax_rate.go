// backend/internal/domain/productBlueprintCategory/tax_rate.go
package productBlueprintCategory

import (
	"errors"
	"strings"
)

type ConsumptionTaxRate int

const (
	ConsumptionTaxRateReduced  ConsumptionTaxRate = 8
	ConsumptionTaxRateStandard ConsumptionTaxRate = 10
)

var (
	ErrConsumptionTaxRateNotFound = errors.New(
		"productBlueprintCategory: consumption tax rate not found",
	)
)

func GetConsumptionTaxRate(
	productBlueprintCategoryPath []string,
) (
	ConsumptionTaxRate,
	error,
) {
	if len(
		productBlueprintCategoryPath,
	) == 0 {
		return 0,
			ErrConsumptionTaxRateNotFound
	}

	categoryCode :=
		strings.Join(
			productBlueprintCategoryPath,
			".",
		)

	switch categoryCode {
	case CategoryCodeHealthcareSupplement:
		return ConsumptionTaxRateReduced,
			nil

	case CategoryCodeAlcoholBeer,
		CategoryCodeAlcoholSake,
		CategoryCodeAlcoholShochu,
		CategoryCodeAlcoholSpirits,
		CategoryCodeAlcoholWhisky,
		CategoryCodeAlcoholWine,

		CategoryCodeApparelAccessory,
		CategoryCodeApparelBag,
		CategoryCodeApparelBottoms,
		CategoryCodeApparelDress,
		CategoryCodeApparelOuterwear,
		CategoryCodeApparelShoes,
		CategoryCodeApparelTops,

		CategoryCodeCosmeticsBodycare,
		CategoryCodeCosmeticsFragrance,
		CategoryCodeCosmeticsHaircare,
		CategoryCodeCosmeticsMakeup,
		CategoryCodeCosmeticsSkincare,

		CategoryCodeHealthcareMedicalDevice,
		CategoryCodeHealthcareWellness,

		CategoryCodeOtherGeneral:
		return ConsumptionTaxRateStandard,
			nil

	default:
		return 0,
			ErrConsumptionTaxRateNotFound
	}
}
