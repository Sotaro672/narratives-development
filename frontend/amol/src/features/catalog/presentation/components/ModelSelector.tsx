// frontend/amol/src/features/catalog/components/ModelSelector.tsx
import { rgbToCssColor } from "../../../../components/utils/color";
import type {
  CatalogListPrice,
  CatalogModelVariation,
  ModelColorOption,
} from "../../types";
import { formatPrice } from "../../utils/format";

type ModelSelectorProps = {
  colorOptions: ModelColorOption[];
  sizeOptions: string[];
  selectedColorKey: string;
  selectedSize: string;
  selectedModel: CatalogModelVariation | null;
  selectedModelPrice: CatalogListPrice | undefined;
  selectedModelStock: number | undefined;
  cartMessage: string;
  cartErrorMessage: string;
  onSelectColor: (colorKey: string) => void;
  onSelectSize: (size: string) => void;
};

export default function ModelSelector({
  colorOptions,
  sizeOptions,
  selectedColorKey,
  selectedSize,
  selectedModel,
  selectedModelPrice,
  selectedModelStock,
  cartMessage,
  cartErrorMessage,
  onSelectColor,
  onSelectSize,
}: ModelSelectorProps) {
  return (
    <section className="catalog-page-card">
      <h2 className="catalog-page-card-title">モデル</h2>

      <div className="catalog-page-option-section">
        <p className="catalog-page-option-label">カラー</p>

        <div className="catalog-page-option-list">
          {colorOptions.map((option) => {
            const colorHex = rgbToCssColor(option.colorRGB);
            const isSelected = selectedColorKey === option.key;

            return (
              <button
                key={option.key}
                type="button"
                className={[
                  "catalog-page-option-button",
                  "catalog-page-color-option-button",
                  isSelected ? "catalog-page-option-button--selected" : "",
                ]
                  .filter(Boolean)
                  .join(" ")}
                onClick={() => onSelectColor(option.key)}
              >
                <span
                  className="catalog-page-color-swatch"
                  style={{ backgroundColor: colorHex }}
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
                ]
                  .filter(Boolean)
                  .join(" ")}
                onClick={() => onSelectSize(size)}
              >
                {size}
              </button>
            );
          })}
        </div>
      </div>

      {selectedModel ? (
        <div className="catalog-page-selected-model">
          <dl className="catalog-page-definition-list">
            <div>
              <dt>カラー</dt>
              <dd>{selectedModel.colorName}</dd>
            </div>
            <div>
              <dt>サイズ</dt>
              <dd>{selectedModel.size}</dd>
            </div>
            <div>
              <dt>価格</dt>
              <dd>
                {selectedModelPrice
                  ? formatPrice(selectedModelPrice.price)
                  : "価格未設定"}
              </dd>
            </div>
            <div>
              <dt>在庫</dt>
              <dd>{selectedModelStock}</dd>
            </div>
          </dl>
        </div>
      ) : (
        <p className="catalog-page-model-help">
          カラーとサイズを選択してください。
        </p>
      )}

      {cartMessage ? (
        <p className="catalog-page-cart-message">{cartMessage}</p>
      ) : null}

      {cartErrorMessage ? (
        <p className="catalog-page-cart-error" role="alert">
          {cartErrorMessage}
        </p>
      ) : null}
    </section>
  );
}