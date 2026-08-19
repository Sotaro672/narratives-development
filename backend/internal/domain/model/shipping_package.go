// backend\internal\domain\model\shipping_package.go
package model

import "errors"

var ErrInvalidShippingPackage = errors.New(
	"model: invalid shipping package",
)

type ShippingPackage struct {
	WeightGrams int
	WidthMM     int
	LengthMM    int
	HeightMM    int
}

func (p ShippingPackage) Validate() error {
	if p.WeightGrams <= 0 ||
		p.WidthMM <= 0 ||
		p.LengthMM <= 0 ||
		p.HeightMM <= 0 {
		return ErrInvalidShippingPackage
	}

	return nil
}
