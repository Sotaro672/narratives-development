// frontend/amol/src/features/catalog/presentation/components/ModelSelector.tsx

import { rgbToCssColor } from "../../../../components/utils/color";
import type {
  CatalogListPrice,
  CatalogModelVariation,
  ModelColorOption,
} from "../../../shared/types/catalog";
import { formatPrice } from "../../../shared/utils/price";
import {
  formatAlcoholVolumeLabel,
} from "../../application/catalogModelMapper";

type ModelSelectorProps = {
  colorOptions: ModelColorOption[];
  sizeOptions: string[];
  selectedColorKey: string;
  selectedSize: string;
  selectedModel: CatalogModelVariation | null;
  selectedModelPrice: CatalogListPrice | undefined;
  selectedModelStock: number | undefined;
  cartErrorMessage: string;
  isAlcoholCatalog?: boolean;
  onSelectColor: (colorKey: string) => void;
  onSelectSize: (size: string) => void;
};

function formatModelNumber(
  model: CatalogModelVariation,
): string {
  return model.modelNumber?.trim() || "-";
}

export default function ModelSelector({
  colorOptions,
  sizeOptions,
  selectedColorKey,
  selectedSize,
  selectedModel,
  selectedModelPrice,
  selectedModelStock,
  cartErrorMessage,
  isAlcoholCatalog = false,
  onSelectColor,
  onSelectSize,
}: ModelSelectorProps) {
  const hasColorOptions =
    colorOptions.length > 0;

  const hasSizeOptions =
    sizeOptions.length > 0;

  return (
    <section className="catalog-page-card">
      <h2 className="catalog-page-card-title">
        モデル
      </h2>

      <div className="catalog-page-option-section">
        <p className="catalog-page-option-label">
          {isAlcoholCatalog
            ? "容量"
            : "カラー"}
        </p>

        <div className="catalog-page-option-list">
          {colorOptions.map((option) => {
            const colorHex =
              rgbToCssColor(
                option.colorRGB,
              );

            const isSelected =
              selectedColorKey ===
              option.key;

            return (
              <button
                key={option.key}
                type="button"
                className={[
                  "catalog-page-option-button",
                  !isAlcoholCatalog
                    ? "catalog-page-color-option-button"
                    : "",
                  isSelected
                    ? "catalog-page-option-button--selected"
                    : "",
                ]
                  .filter(Boolean)
                  .join(" ")}
                onClick={() =>
                  onSelectColor(
                    option.key,
                  )
                }
              >
                {!isAlcoholCatalog ? (
                  <span
                    className="catalog-page-color-swatch"
                    style={{
                      backgroundColor:
                        colorHex,
                    }}
                    aria-hidden="true"
                  />
                ) : null}

                <span>
                  {option.colorName}
                </span>
              </button>
            );
          })}
        </div>
      </div>

      {!isAlcoholCatalog ? (
        <div className="catalog-page-option-section">
          <p className="catalog-page-option-label">
            サイズ
          </p>

          <div className="catalog-page-option-list">
            {sizeOptions.map((size) => {
              const isSelected =
                selectedSize === size;

              return (
                <button
                  key={size}
                  type="button"
                  className={[
                    "catalog-page-option-button",
                    isSelected
                      ? "catalog-page-option-button--selected"
                      : "",
                  ]
                    .filter(Boolean)
                    .join(" ")}
                  onClick={() =>
                    onSelectSize(size)
                  }
                >
                  {size}
                </button>
              );
            })}
          </div>
        </div>
      ) : null}

      {selectedModel ? (
        <div className="catalog-page-selected-model">
          <dl className="catalog-page-definition-list">
            {isAlcoholCatalog ? (
              <>
                <div>
                  <dt>
                    モデル番号
                  </dt>

                  <dd>
                    {formatModelNumber(
                      selectedModel,
                    )}
                  </dd>
                </div>

                <div>
                  <dt>容量</dt>

                  <dd>
                    {formatAlcoholVolumeLabel(
                      selectedModel,
                    ) || "-"}
                  </dd>
                </div>
              </>
            ) : (
              <>
                <div>
                  <dt>カラー</dt>

                  <dd>
                    {selectedModel
                      .colorName ||
                      "-"}
                  </dd>
                </div>

                <div>
                  <dt>サイズ</dt>

                  <dd>
                    {selectedModel.size ||
                      "-"}
                  </dd>
                </div>
              </>
            )}

            <div>
              <dt>価格</dt>

              <dd>
                {formatPrice(
                  selectedModelPrice?.price,
                )}
              </dd>
            </div>

            <div>
              <dt>在庫</dt>

              <dd>
                {typeof selectedModelStock ===
                "number"
                  ? selectedModelStock
                  : "-"}
              </dd>
            </div>
          </dl>
        </div>
      ) : (
        <p className="catalog-page-model-help">
          {isAlcoholCatalog
            ? hasColorOptions
              ? "容量を選択してください。"
              : "選択できる容量がありません。"
            : hasColorOptions ||
                hasSizeOptions
              ? "カラーとサイズを選択してください。"
              : "選択できるモデルがありません。"}
        </p>
      )}

      {cartErrorMessage ? (
        <p
          className="catalog-page-cart-error"
          role="alert"
        >
          {cartErrorMessage}
        </p>
      ) : null}
    </section>
  );
}