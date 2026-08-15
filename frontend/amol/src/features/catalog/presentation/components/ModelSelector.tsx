// frontend/amol/src/features/catalog/presentation/components/ModelSelector.tsx

import { rgbToCssColor } from "../../../../components/utils/color";
import { formatPrice } from "../../../../components/utils/price";
import type { CatalogListPrice, CatalogModelVariation, ModelColorOption } from "../../../shared/types/catalog";
import type { CatalogAlcoholOption } from "../../application/catalogSelectionFactory";
import { formatAlcoholVolumeLabel } from "../../application/catalogModelMapper";

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

  return (
    <section className="catalog-page-card">
      <h2 className="catalog-page-card-title">モデル</h2>

      {isAlcoholCatalog ? (
        <div className="catalog-page-option-section">
          <p className="catalog-page-option-label">容量</p>

          <div className="catalog-page-option-list">
            {alcoholOptions.map((option) => {
              const isSelected = selectedModelId === option.modelId;

              return (
                <button
                  key={option.modelId}
                  type="button"
                  className={[
                    "catalog-page-option-button",
                    isSelected ? "catalog-page-option-button--selected" : "",
                  ].filter(Boolean).join(" ")}
                  onClick={() => onSelectModel(option.modelId)}
                >
                  {option.label}
                </button>
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
                  <button
                    key={option.key}
                    type="button"
                    className={[
                      "catalog-page-option-button",
                      "catalog-page-color-option-button",
                      isSelected ? "catalog-page-option-button--selected" : "",
                    ].filter(Boolean).join(" ")}
                    onClick={() => onSelectColor(option.key)}
                  >
                    <span
                      className="catalog-page-color-swatch"
                      style={{ backgroundColor: rgbToCssColor(option.colorRGB) }}
                      aria-hidden="true"
                    />
                    <span>{option.colorName}</span>
                  </button>
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
                  <button
                    key={size}
                    type="button"
                    className={[
                      "catalog-page-option-button",
                      isSelected ? "catalog-page-option-button--selected" : "",
                    ].filter(Boolean).join(" ")}
                    onClick={() => onSelectSize(size)}
                  >
                    {size}
                  </button>
                );
              })}
            </div>
          </div>
        </>
      )}

      {selectedModel ? (
        <div className="catalog-page-selected-model">
          <dl className="catalog-page-definition-list">
            {isAlcoholCatalog ? (
              <>
                <div>
                  <dt>モデル番号</dt>
                  <dd>{selectedModel.modelNumber.trim() || "-"}</dd>
                </div>

                <div>
                  <dt>容量</dt>
                  <dd>{formatAlcoholVolumeLabel(selectedModel) || "-"}</dd>
                </div>
              </>
            ) : (
              <>
                <div>
                  <dt>カラー</dt>
                  <dd>{selectedModel.colorName || "-"}</dd>
                </div>

                <div>
                  <dt>サイズ</dt>
                  <dd>{selectedModel.size || "-"}</dd>
                </div>
              </>
            )}

            <div>
              <dt>価格</dt>
              <dd>{formatPrice(selectedModelPrice?.price)}</dd>
            </div>

            <div>
              <dt>在庫</dt>
              <dd>{typeof selectedModelStock === "number" ? selectedModelStock : "-"}</dd>
            </div>
          </dl>
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
        <p className="catalog-page-cart-error" role="alert">
          {cartErrorMessage}
        </p>
      ) : null}
    </section>
  );
}