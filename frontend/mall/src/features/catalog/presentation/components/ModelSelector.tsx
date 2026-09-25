// frontend/mall/src/features/catalog/presentation/components/ModelSelector.tsx

import Alert from "../../../../components/ui/Alert";
import Chip from "../../../../components/ui/Chip";
import InfoList, { InfoRow } from "../../../../components/ui/InfoList";
import SectionHeader from "../../../../components/ui/SectionHeader";
import { rgbToCssColor } from "../../../../components/utils/color";
import { formatPrice } from "../../../../components/utils/price";
import type {
  CatalogListPrice,
  CatalogModelVariation,
  ModelColorOption,
} from "../../../shared/types/catalog";
import { createProductModelDisplay } from "../../../shared/presentation/utils/productModelDisplay";
import { formatAlcoholVolumeLabel } from "../../application/catalogModelMapper";
import type { CatalogAlcoholOption } from "../../application/catalogSelectionFactory";

type ModelSelectorProps = {
  alcoholOptions: CatalogAlcoholOption[];
  colorOptions: ModelColorOption[];
  sizeOptions: string[];
  selectedColorKey: string;
  selectedSize: string;
  selectedModelId: string;
  selectedModel: CatalogModelVariation | null;
  selectedModelPrice: CatalogListPrice | undefined;
  selectedModelStock: number | undefined;
  cartErrorMessage: string;
  isAlcoholCatalog: boolean;
  onSelectColor: (colorKey: string) => void;
  onSelectSize: (size: string) => void;
  onSelectModel: (modelId: string) => void;
};

export default function ModelSelector({
  alcoholOptions,
  colorOptions,
  sizeOptions,
  selectedColorKey,
  selectedSize,
  selectedModelId,
  selectedModel,
  selectedModelPrice,
  selectedModelStock,
  cartErrorMessage,
  isAlcoholCatalog,
  onSelectColor,
  onSelectSize,
  onSelectModel,
}: ModelSelectorProps) {
  const hasAlcoholOptions = alcoholOptions.length > 0;
  const hasColorOptions = colorOptions.length > 0;
  const hasSizeOptions = sizeOptions.length > 0;
  const selectedModelMeasurementsLabel = selectedModel
    ? createProductModelDisplay({
        measurements: selectedModel.measurements,
      }).measurementsLabel
    : "-";

  return (
    <section className="catalog-page-model">
      <SectionHeader
        title="モデル"
        titleAs="h2"
        className="catalog-page-card-header"
      />

      {isAlcoholCatalog ? (
        <div className="catalog-page-option-section">
          <p className="catalog-page-option-label">容量</p>

          <div className="catalog-page-option-list">
            {alcoholOptions.map((option) => {
              const isSelected = selectedModelId === option.modelId;

              return (
                <Chip
                  key={option.modelId}
                  selected={isSelected}
                  onClick={() => onSelectModel(option.modelId)}
                >
                  {option.label}
                </Chip>
              );
            })}
          </div>
        </div>
      ) : (
        <>
          <div className="catalog-page-option-section">
            <p className="catalog-page-option-label">カラー</p>

            <div className="catalog-page-option-list">
              {colorOptions.map((option) => {
                const isSelected = selectedColorKey === option.key;

                return (
                  <Chip
                    key={option.key}
                    selected={isSelected}
                    className="catalog-page-color-option-button"
                    onClick={() => onSelectColor(option.key)}
                  >
                    <span
                      className="catalog-page-color-swatch"
                      style={{
                        backgroundColor: rgbToCssColor(option.colorRGB),
                      }}
                      aria-hidden="true"
                    />
                    <span>{option.colorName}</span>
                  </Chip>
                );
              })}
            </div>
          </div>

          <div className="catalog-page-option-section">
            <p className="catalog-page-option-label">サイズ</p>

            <div className="catalog-page-option-list">
              {sizeOptions.map((size) => {
                const isSelected = selectedSize === size;

                return (
                  <Chip
                    key={size}
                    selected={isSelected}
                    onClick={() => onSelectSize(size)}
                  >
                    {size}
                  </Chip>
                );
              })}
            </div>
          </div>
        </>
      )}

      {selectedModel ? (
        <div className="catalog-page-selected-model">
          <InfoList className="catalog-page-definition-list">
            {isAlcoholCatalog ? (
              <>
                <InfoRow label="モデル番号">
                  {selectedModel.modelNumber.trim() || "-"}
                </InfoRow>
                <InfoRow label="容量">
                  {formatAlcoholVolumeLabel(selectedModel) || "-"}
                </InfoRow>
              </>
            ) : (
              <InfoRow label="採寸">
                {selectedModelMeasurementsLabel}
              </InfoRow>
            )}

            <InfoRow label="価格">
              {formatPrice(selectedModelPrice?.price)}
            </InfoRow>
            <InfoRow label="在庫">
              {typeof selectedModelStock === "number"
                ? selectedModelStock
                : "-"}
            </InfoRow>
          </InfoList>
        </div>
      ) : (
        <p className="catalog-page-model-help">
          {isAlcoholCatalog
            ? hasAlcoholOptions
              ? "容量を選択してください。"
              : "選択できる容量がありません。"
            : hasColorOptions || hasSizeOptions
              ? "カラーとサイズを選択してください。"
              : "選択できるモデルがありません。"}
        </p>
      )}

      {cartErrorMessage ? (
        <Alert variant="error" className="catalog-page-cart-error">
          {cartErrorMessage}
        </Alert>
      ) : null}
    </section>
  );
}