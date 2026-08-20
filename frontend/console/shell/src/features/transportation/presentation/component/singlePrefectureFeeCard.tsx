// frontend/console/shell/src/features/transportation/presentation/component/singlePrefectureFeeCard.tsx

import * as React from "react";
import { MapPin } from "lucide-react";

import { Card, CardContent, CardTitle } from "../../../../shared/ui/card";
import { Input } from "../../../../shared/ui/input";

import type { TransportationRegionVM } from "../../application/transportationService";
import type { PrefectureCode } from "../../../../shared/types/transporation";

export type SinglePrefectureFeeCardProps = {
  region: TransportationRegionVM;
  disabled?: boolean;
  className?: string;
  onChangePrefectureAmount: (prefectureCode: PrefectureCode, amount: string | number | null) => void;
};

const SinglePrefectureFeeCard: React.FC<SinglePrefectureFeeCardProps> = ({
  region,
  disabled = false,
  className,
  onChangePrefectureAmount,
}) => {
  const prefecture = region.prefectures[0];

  return (
    <Card className={className}>
      <CardContent className="pt-6">
        {prefecture ? (
          <div className="flex flex-wrap items-center gap-3 sm:flex-nowrap">
            <MapPin size={18} className="shrink-0" />
            <CardTitle className="whitespace-nowrap text-base">{region.regionName}</CardTitle>
            <span className="whitespace-nowrap font-medium text-slate-900">{prefecture.prefectureName}</span>
            <div className="flex items-center gap-2">
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
          </div>
        ) : (
          <div className="py-2 text-left text-sm text-slate-500">
            都道府県データがありません。
          </div>
        )}
      </CardContent>
    </Card>
  );
};

export default SinglePrefectureFeeCard;