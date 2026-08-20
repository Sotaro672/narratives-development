// frontend/console/shell/src/features/transportation/presentation/component/feeEditCard.tsx

import * as React from "react";
import { MapPin } from "lucide-react";

import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../../../../shared/ui/card";
import { Input } from "../../../../shared/ui/input";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "../../../../shared/ui/table";

import type {
  TransportationRegionVM,
} from "../../application/transportationService";
import type {
  PrefectureCode,
  TransportationRegion,
} from "../../../../shared/types/transporation";

export type FeeEditCardProps = {
  region: TransportationRegionVM;
  disabled?: boolean;
  className?: string;
  onChangePrefectureAmount: (
    prefectureCode: PrefectureCode,
    amount: string | number,
  ) => void;
  onChangeRegionAmount: (
    region: TransportationRegion,
    amount: string | number,
  ) => void;
};

function getRegionAmountValue(
  region: TransportationRegionVM,
): string {
  if (region.prefectures.length === 0) {
    return "";
  }

  const firstAmount = region.prefectures[0]?.amount;

  if (firstAmount === undefined) {
    return "";
  }

  const allSame = region.prefectures.every(
    (prefecture) => prefecture.amount === firstAmount,
  );

  return allSame ? String(firstAmount) : "";
}

const FeeEditCard: React.FC<FeeEditCardProps> = ({
  region,
  disabled = false,
  className,
  onChangePrefectureAmount,
  onChangeRegionAmount,
}) => {
  const regionAmountValue = React.useMemo(
    () => getRegionAmountValue(region),
    [region],
  );

  const handleRegionAmountChange = React.useCallback(
    (event: React.ChangeEvent<HTMLInputElement>) => {
      onChangeRegionAmount(
        region.region,
        event.target.value,
      );
    },
    [
      onChangeRegionAmount,
      region.region,
    ],
  );

  return (
    <Card className={className}>
      <CardHeader>
        <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex items-center gap-2">
            <MapPin size={18} />
            <CardTitle className="text-base">
              {region.regionName}
            </CardTitle>
          </div>

          <div className="flex items-center gap-2">
            <label
              htmlFor={`transportation-region-${region.region}`}
              className="whitespace-nowrap text-sm font-medium text-slate-700"
            >
              地方一括
            </label>

            <div className="flex items-center gap-2">
              <span className="text-sm text-slate-500">
                ¥
              </span>

              <Input
                id={`transportation-region-${region.region}`}
                type="number"
                inputMode="numeric"
                min={0}
                step={1}
                disabled={disabled}
                value={regionAmountValue}
                placeholder="個別設定"
                className="h-9 w-32 text-right"
                onChange={handleRegionAmountChange}
              />
            </div>
          </div>
        </div>

        <p className="text-xs text-slate-500">
          地方一括に金額を入力すると、この地方に含まれるすべての都道府県へ同じ送料を設定します。
        </p>
      </CardHeader>

      <CardContent>
        <div className="overflow-x-auto">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>
                  都道府県
                </TableHead>

                <TableHead className="w-48 text-right">
                  送料
                </TableHead>
              </TableRow>
            </TableHeader>

            <TableBody>
              {region.prefectures.map((prefecture) => (
                <TableRow key={prefecture.prefectureCode}>
                  <TableCell>
                    <div className="flex items-center gap-3">
                      <span className="font-medium text-slate-900">
                        {prefecture.prefectureName}
                      </span>

                      <span className="text-xs text-slate-400">
                        {prefecture.prefectureCode}
                      </span>
                    </div>
                  </TableCell>

                  <TableCell>
                    <div className="flex items-center justify-end gap-2">
                      <span className="text-sm text-slate-500">
                        ¥
                      </span>

                      <Input
                        type="number"
                        inputMode="numeric"
                        min={0}
                        step={1}
                        disabled={disabled}
                        value={prefecture.amount}
                        className="h-9 w-32 text-right"
                        aria-label={`${prefecture.prefectureName}の送料`}
                        onChange={(event) => {
                          onChangePrefectureAmount(
                            prefecture.prefectureCode,
                            event.target.value,
                          );
                        }}
                      />
                    </div>
                  </TableCell>
                </TableRow>
              ))}

              {region.prefectures.length === 0 && (
                <TableRow>
                  <TableCell
                    colSpan={2}
                    className="py-8 text-center text-sm text-slate-500"
                  >
                    都道府県データがありません。
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </div>
      </CardContent>
    </Card>
  );
};

export default FeeEditCard;