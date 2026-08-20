// frontend/console/shell/src/features/transportation/presentation/component/singlePrefectureFeeCard.tsx

import * as React from "react";
import { MapPin } from "lucide-react";

import { Card, CardContent, CardHeader, CardTitle } from "../../../../shared/ui/card";
import { Input } from "../../../../shared/ui/input";

import type { TransportationRegionVM } from "../../application/transportationService";
import type { PrefectureCode } from "../../../../shared/types/transporation";

export type SinglePrefectureFeeCardProps = {
  region: TransportationRegionVM;
  disabled?: boolean;
  className?: string;
  onChangePrefectureAmount: (prefectureCode: PrefectureCode, amount: string | number) => void;
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
      <CardHeader>
        <div className="flex items-center gap-2">
          <MapPin size={18} />
          <CardTitle className="text-base">{region.regionName}</CardTitle>
        </div>
      </CardHeader>

      <CardContent>
        {prefecture ? (
          <div className="flex items-center justify-between gap-4">
            <div className="flex items-center gap-3">
              <span className="font-medium text-slate-900">{prefecture.prefectureName}</span>
              <span className="text-xs text-slate-400">{prefecture.prefectureCode}</span>
            </div>

            <div className="flex items-center gap-2">
              <span className="text-sm text-slate-500">¥</span>
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
                  onChangePrefectureAmount(prefecture.prefectureCode, event.target.value);
                }}
              />
            </div>
          </div>
        ) : (
          <div className="py-6 text-center text-sm text-slate-500">
            都道府県データがありません。
          </div>
        )}
      </CardContent>
    </Card>
  );
};

export default SinglePrefectureFeeCard;