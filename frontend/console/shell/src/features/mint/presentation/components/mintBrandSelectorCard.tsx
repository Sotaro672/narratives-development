// frontend/console/shell/src/features/mint/presentation/components/mintBrandSelectorCard.tsx

import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../../../../shared/ui/card";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "../../../../shared/ui/popover";

import type { BrandSummary } from "../../infrastructure/dto/MintRequestRepository";

export type MintBrandSelectorCardProps = {
  brandOptions: BrandSummary[];
  selectedBrandId: string;
  selectedBrandName: string;
  disabled?: boolean;
  onSelectBrand: (brandId: string) => void;
};

export default function MintBrandSelectorCard({
  brandOptions,
  selectedBrandId,
  selectedBrandName,
  disabled = false,
  onSelectBrand,
}: MintBrandSelectorCardProps) {
  return (
    <Card className="pb-select">
      <CardHeader>
        <CardTitle>ブランド選択</CardTitle>
      </CardHeader>

      <CardContent>
        <Popover>
          <PopoverTrigger>
            <div className="pb-select__trigger">
              {selectedBrandName || "ブランドを選択"}
            </div>
          </PopoverTrigger>

          <PopoverContent>
            <div className="pb-select__list">
              {brandOptions.map((brand) => (
                <button
                  key={brand.id}
                  type="button"
                  className={
                    "pb-select__row" +
                    (selectedBrandId === brand.id
                      ? " is-active"
                      : "")
                  }
                  onClick={() => onSelectBrand(brand.id)}
                  disabled={disabled}
                >
                  {brand.name}
                </button>
              ))}

              {brandOptions.length === 0 && (
                <div className="pb-select__empty">
                  ブランド候補が未設定です
                </div>
              )}
            </div>
          </PopoverContent>
        </Popover>
      </CardContent>
    </Card>
  );
}