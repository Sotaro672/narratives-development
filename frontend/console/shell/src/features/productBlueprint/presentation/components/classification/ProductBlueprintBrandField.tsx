// frontend/console/shell/src/features/productBlueprint/presentation/components/classification/ProductBlueprintBrandField.tsx

import * as React from "react";

import { Button } from "../../../../../shared/ui/button";
import { CardField } from "../../../../../shared/ui/card";
import { ErrorMessage } from "../../../../../shared/ui/error";
import { Input } from "../../../../../shared/ui/input";
import { Label } from "../../../../../shared/ui/label";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "../../../../../shared/ui/popover";

import "../../../../../styles/productBlueprint.css";

export type BrandOption = {
  id: string;
  name: string;
};

type ProductBlueprintBrandFieldProps = {
  brandId: string;
  brandName?: string;
  brandOptions?: BrandOption[];
  brandLoading?: boolean;
  brandError?: Error | null;
  mode?: "edit" | "view";
  onChangeBrandId?: (id: string) => void;
};

const ProductBlueprintBrandField: React.FC<ProductBlueprintBrandFieldProps> = ({
  brandId,
  brandName,
  brandOptions,
  brandLoading,
  brandError,
  mode = "edit",
  onChangeBrandId,
}) => {
  const isEdit = mode === "edit";

  const selectedBrandName =
    brandOptions?.find((brand) => brand.id === brandId)?.name ?? "";

  const displayBrandName =
    String(brandName ?? "").trim() ||
    String(selectedBrandName ?? "").trim() ||
    (brandId ? `(${brandId})` : "");

  return (
    <CardField>
      <Label>ブランド</Label>

      {isEdit && brandOptions && onChangeBrandId ? (
        <div className="pbc-brand-field">
          <Popover>
            <PopoverTrigger>
              <Button
                type="button"
                variant="outline"
                className="pbc-brand-field__trigger"
                aria-label="ブランドを選択"
              >
                {selectedBrandName || "ブランドを選択してください。"}
              </Button>
            </PopoverTrigger>

            <PopoverContent
              align="start"
              className="popover__content--compact"
            >
              {brandOptions.length === 0 ? (
                <div className="popover__empty">
                  ブランド候補がありません。
                </div>
              ) : (
                <div className="popover__list">
                  {brandOptions.map((brand) => {
                    const isSelected = brandId === brand.id;

                    return (
                      <button
                        key={brand.id}
                        type="button"
                        className={`popover__item${isSelected ? " is-active" : ""}`}
                        onClick={() => onChangeBrandId(brand.id)}
                      >
                        {brand.name}
                      </button>
                    );
                  })}
                </div>
              )}
            </PopoverContent>
          </Popover>

          {brandLoading && (
            <p className="pbc-brand-field__status">
              ブランドを取得中…
            </p>
          )}

          {brandError && (
            <ErrorMessage
              as="p"
              size="xs"
              className="pbc-brand-field__error"
            >
              ブランド一覧の取得に失敗しました。
            </ErrorMessage>
          )}
        </div>
      ) : (
        <Input
          value={displayBrandName}
          variant="readonly"
          readOnly
          aria-label="ブランド"
        />
      )}
    </CardField>
  );
};

export default ProductBlueprintBrandField;