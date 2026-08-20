// frontend/console/shell/src/features/transportation/presentation/component/feeEditCard.tsx

import * as React from "react";
import { ChevronDown, MapPin } from "lucide-react";

import { Card, CardContent, CardHeader, CardTitle } from "../../../../shared/ui/card";
import { Input } from "../../../../shared/ui/input";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "../../../../shared/ui/table";

import type { TransportationRegionVM } from "../../application/transportationService";
import type { PrefectureCode, TransportationRegion } from "../../../../shared/types/transporation";

export type FeeEditCardProps = {
  region: TransportationRegionVM;
  disabled?: boolean;
  className?: string;
  onChangePrefectureAmount: (prefectureCode: PrefectureCode, amount: string | number | null) => void;
  onChangeRegionAmount: (region: TransportationRegion, amount: string | number | null) => void;
};

function getRegionAmountValue(region: TransportationRegionVM): string {
  if (region.prefectures.length === 0) {
    return "";
  }

  const firstAmount = region.prefectures[0]?.amount;

  if (firstAmount === undefined || firstAmount === null) {
    return "";
  }

  const allSame = region.prefectures.every((prefecture) => prefecture.amount === firstAmount);
  return allSame ? String(firstAmount) : "";
}

const FeeEditCard: React.FC<FeeEditCardProps> = ({
  region,
  disabled = false,
  className,
  onChangePrefectureAmount,
  onChangeRegionAmount,
}) => {
  const [isOpen, setIsOpen] = React.useState(false);
  const regionAmountValue = React.useMemo(() => getRegionAmountValue(region), [region]);
  const contentId = `transportation-region-content-${region.region}`;

  const handleToggle = React.useCallback(() => {
    setIsOpen((current) => !current);
  }, []);

  const handleRegionAmountChange = React.useCallback(
    (event: React.ChangeEvent<HTMLInputElement>) => {
      onChangeRegionAmount(region.region, event.target.value);
    },
    [onChangeRegionAmount, region.region],
  );

  return (
    <Card className={className}>
      <CardHeader>
        <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex items-center gap-2">
            <MapPin size={18} />
            <CardTitle className="text-base">{region.regionName}</CardTitle>
          </div>

          <div className="flex items-center gap-2">
            <label
              htmlFor={`transportation-region-${region.region}`}
              className="whitespace-nowrap text-sm font-medium text-slate-700"
            >
              地方一括
            </label>

            <div className="flex items-center gap-2">
              <span className="text-sm text-slate-500">¥</span>
              <Input
                id={`transportation-region-${region.region}`}
                type="number"
                inputMode="numeric"
                min={0}
                step={1}
                disabled={disabled}
                value={regionAmountValue}
                placeholder="未設定"
                className="h-9 w-32 text-right"
                onChange={handleRegionAmountChange}
              />
            </div>
          </div>
        </div>

        <button
          type="button"
          onClick={handleToggle}
          aria-expanded={isOpen}
          aria-controls={contentId}
          aria-label={`${region.regionName}の都道府県一覧を${isOpen ? "閉じる" : "開く"}`}
          className="flex w-fit items-center gap-2 text-sm font-medium text-slate-600 transition hover:text-slate-900"
        >
          <span className="text-xs font-normal text-slate-400">{region.prefectures.length}都道府県</span>
          <ChevronDown
            size={16}
            aria-hidden="true"
            className={`text-slate-500 transition-transform duration-200 ${isOpen ? "rotate-180" : ""}`}
          />
        </button>
      </CardHeader>

      {isOpen && (
        <CardContent id={contentId}>
          <div className="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>都道府県</TableHead>
                  <TableHead className="w-48 text-right">送料</TableHead>
                </TableRow>
              </TableHeader>

              <TableBody>
                {region.prefectures.map((prefecture) => (
                  <TableRow key={prefecture.prefectureCode}>
                    <TableCell>
                      <div className="flex items-center gap-3">
                        <span className="font-medium text-slate-900">{prefecture.prefectureName}</span>
                      </div>
                    </TableCell>

                    <TableCell>
                      <div className="flex items-center justify-end gap-2">
                        <span className="text-sm text-slate-500">¥</span>
                        <Input
                          type="number"
                          inputMode="numeric"
                          min={0}
                          step={1}
                          disabled={disabled}
                          value={prefecture.amount ?? ""}
                          placeholder="未設定"
                          className="h-9 w-32 text-right"
                          aria-label={`${prefecture.prefectureName}の送料`}
                          onChange={(event) => {
                            onChangePrefectureAmount(prefecture.prefectureCode, event.target.value);
                          }}
                        />
                      </div>
                    </TableCell>
                  </TableRow>
                ))}

                {region.prefectures.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={2} className="py-8 text-center text-sm text-slate-500">
                      都道府県データがありません。
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </div>
        </CardContent>
      )}
    </Card>
  );
};

export default FeeEditCard;