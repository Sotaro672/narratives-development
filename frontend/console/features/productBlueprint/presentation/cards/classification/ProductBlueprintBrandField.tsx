// frontend/console/productBlueprint/src/presentation/components/ProductBlueprintBrandField.tsx

import * as React from "react";
import { Button } from "../../../../../shell/src/shared/ui/button";
import { Input } from "../../../../../shell/src/shared/ui/input";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "../../../../../shell/src/shared/ui/popover";

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
    <>
      <div className="label">ブランド</div>
      {isEdit && brandOptions && onChangeBrandId ? (
        <div className="mb-2 space-y-1">
          <Popover>
            <PopoverTrigger>
              <Button
                variant="outline"
                className="w-full justify-between pbc-select-trigger"
                aria-label="ブランドを選択"
              >
                {selectedBrandName || "ブランドを選択してください。"}
              </Button>
            </PopoverTrigger>
            <PopoverContent align="start" className="p-1">
              {brandOptions.map((brand) => (
                <div
                  key={brand.id}
                  className={`px-3 py-2 rounded-md cursor-pointer hover:bg-blue-50 ${
                    brandId === brand.id
                      ? "bg-blue-100 text-blue-700 font-medium"
                      : ""
                  }`}
                  onClick={() => onChangeBrandId(brand.id)}
                >
                  {brand.name}
                </div>
              ))}
            </PopoverContent>
          </Popover>

          {brandLoading && (
            <p className="text-xs text-slate-400">ブランドを取得中…</p>
          )}
          {brandError && (
            <p className="text-xs text-red-500">
              ブランド一覧の取得に失敗しました。
            </p>
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
    </>
  );
};

export default ProductBlueprintBrandField;