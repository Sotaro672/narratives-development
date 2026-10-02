// backend/internal/adapters/in/http/console/handler/print_pdf_builder.go
package consoleHandler

import (
	"bytes"
	"fmt"
	"math"
	"strings"

	"codeberg.org/go-pdf/fpdf"
	"github.com/skip2/go-qrcode"

	consolequery "narratives/internal/application/query/console"
)

const (
	printPDFPageWidth  = 595.28
	printPDFPageHeight = 841.89

	printPDFMarginX    = 36.0
	printPDFMarginY    = 36.0
	printPDFCols       = 5
	printPDFCellHeight = 100.0

	printPDFQRPaddingX     = 10.0
	printPDFQRReservedText = 30.0
	printPDFQRBottomGap    = 20.0

	printPDFLabelHeight      = 20.0
	printPDFLabelMaxFontSize = 14.0
	printPDFLabelMinFontSize = 6.0
	printPDFLabelPaddingX    = 8.0

	printPDFQRImageSize = 256
)

var printPDFImageOptions = fpdf.ImageOptions{
	ImageType:             "png",
	ReadDpi:               false,
	AllowNegativePosition: false,
}

// BuildPrintPDF は Console の print_log DTO から QR 印刷用 PDF を生成する。
// レイアウトは旧フロントの qrPdfBuilder.ts と同じ値を使用する。
//
// - A4 portrait
// - 左右上下 margin: 36pt
// - 5 columns
// - cell height: 100pt
// - QR: 各セル中央
// - modelNumber: QR 下部中央
func BuildPrintPDF(
	logs []consolequery.PrintLogForPrintDTO,
) ([]byte, error) {
	items := flattenPrintPDFItems(logs)
	if len(items) == 0 {
		return nil, fmt.Errorf("print pdf: no printable items")
	}

	pdf := fpdf.New(
		"P",
		"pt",
		"A4",
		"",
	)

	pdf.SetMargins(0, 0, 0)
	pdf.SetAutoPageBreak(false, 0)
	pdf.SetCompression(true)
	pdf.SetCreator("AMOL Console", true)
	pdf.SetTitle("AMOL QR Print", true)
	pdf.SetTextColor(0, 0, 0)

	cellWidth :=
		(printPDFPageWidth - printPDFMarginX*2) /
			printPDFCols

	qrSize := math.Min(
		cellWidth-printPDFQRPaddingX,
		printPDFCellHeight-printPDFQRReservedText,
	)

	if qrSize <= 0 {
		return nil, fmt.Errorf(
			"print pdf: invalid QR size %.2f",
			qrSize,
		)
	}

	availableHeight :=
		printPDFPageHeight -
			printPDFMarginY*2

	rowsPerPage :=
		int(math.Floor(
			availableHeight /
				printPDFCellHeight,
		))

	if rowsPerPage <= 0 {
		return nil, fmt.Errorf(
			"print pdf: invalid rows per page",
		)
	}

	cellsPerPage :=
		rowsPerPage * printPDFCols

	translateLabel :=
		pdf.UnicodeTranslatorFromDescriptor("")

	for index, item := range items {
		pageCellIndex :=
			index % cellsPerPage

		if pageCellIndex == 0 {
			pdf.AddPage()

			if err := pdf.Error(); err != nil {
				return nil, fmt.Errorf(
					"print pdf: add page: %w",
					err,
				)
			}
		}

		row :=
			pageCellIndex /
				printPDFCols

		col :=
			pageCellIndex %
				printPDFCols

		cellX :=
			printPDFMarginX +
				float64(col)*cellWidth

		cellY :=
			printPDFMarginY +
				float64(row)*printPDFCellHeight

		if err := drawPrintPDFItem(
			pdf,
			item,
			index,
			cellX,
			cellY,
			cellWidth,
			qrSize,
			translateLabel,
		); err != nil {
			return nil, err
		}
	}

	if err := pdf.Error(); err != nil {
		return nil, fmt.Errorf(
			"print pdf: build document: %w",
			err,
		)
	}

	var output bytes.Buffer

	if err := pdf.Output(&output); err != nil {
		return nil, fmt.Errorf(
			"print pdf: output document: %w",
			err,
		)
	}

	if output.Len() == 0 {
		return nil, fmt.Errorf(
			"print pdf: generated document is empty",
		)
	}

	return output.Bytes(), nil
}

func flattenPrintPDFItems(
	logs []consolequery.PrintLogForPrintDTO,
) []consolequery.PrintedItemForPrintDTO {
	total := 0

	for _, log := range logs {
		total += len(log.Items)
	}

	items :=
		make(
			[]consolequery.PrintedItemForPrintDTO,
			0,
			total,
		)

	for _, log := range logs {
		for _, item := range log.Items {
			if strings.TrimSpace(
				item.QRPayload,
			) == "" {
				continue
			}

			items = append(
				items,
				item,
			)
		}
	}

	return items
}

func drawPrintPDFItem(
	pdf *fpdf.Fpdf,
	item consolequery.PrintedItemForPrintDTO,
	index int,
	cellX float64,
	cellY float64,
	cellWidth float64,
	qrSize float64,
	translateLabel func(string) string,
) error {
	payload :=
		strings.TrimSpace(
			item.QRPayload,
		)

	if payload == "" {
		return fmt.Errorf(
			"print pdf: QR payload is empty at item %d",
			index+1,
		)
	}

	pngBytes, err :=
		qrcode.Encode(
			payload,
			qrcode.Medium,
			printPDFQRImageSize,
		)
	if err != nil {
		return fmt.Errorf(
			"print pdf: generate QR at item %d: %w",
			index+1,
			err,
		)
	}

	imageName :=
		fmt.Sprintf(
			"qr-%06d",
			index+1,
		)

	pdf.RegisterImageOptionsReader(
		imageName,
		printPDFImageOptions,
		bytes.NewReader(pngBytes),
	)

	if err := pdf.Error(); err != nil {
		return fmt.Errorf(
			"print pdf: register QR image at item %d: %w",
			index+1,
			err,
		)
	}

	// 旧 frontend の座標:
	// qrY = yOffset + 20pt（bottom-left origin）
	// を FPDF の top-left origin に変換すると、
	// cell 上端から 10pt の位置になる。
	qrX :=
		cellX +
			(cellWidth-qrSize)/2

	qrY :=
		cellY +
			printPDFCellHeight -
			printPDFQRBottomGap -
			qrSize

	pdf.ImageOptions(
		imageName,
		qrX,
		qrY,
		qrSize,
		qrSize,
		false,
		printPDFImageOptions,
		0,
		"",
	)

	if err := pdf.Error(); err != nil {
		return fmt.Errorf(
			"print pdf: draw QR image at item %d: %w",
			index+1,
			err,
		)
	}

	label :=
		strings.TrimSpace(
			item.ModelNumber,
		)

	if label == "" {
		return nil
	}

	if translateLabel != nil {
		label =
			translateLabel(
				label,
			)
	}

	drawPrintPDFLabel(
		pdf,
		label,
		cellX,
		cellY,
		cellWidth,
	)

	if err := pdf.Error(); err != nil {
		return fmt.Errorf(
			"print pdf: draw model number at item %d: %w",
			index+1,
			err,
		)
	}

	return nil
}

func drawPrintPDFLabel(
	pdf *fpdf.Fpdf,
	label string,
	cellX float64,
	cellY float64,
	cellWidth float64,
) {
	maxWidth :=
		cellWidth -
			printPDFLabelPaddingX

	fontSize :=
		printPDFLabelMaxFontSize

	for {
		pdf.SetFont(
			"Helvetica",
			"",
			fontSize,
		)

		if pdf.GetStringWidth(label) <= maxWidth {
			break
		}

		if fontSize <= printPDFLabelMinFontSize {
			break
		}

		fontSize -= 1
	}

	labelY :=
		cellY +
			printPDFCellHeight -
			printPDFLabelHeight

	pdf.SetXY(
		cellX,
		labelY,
	)

	pdf.CellFormat(
		cellWidth,
		printPDFLabelHeight,
		label,
		"",
		0,
		"C",
		false,
		0,
		"",
	)
}
