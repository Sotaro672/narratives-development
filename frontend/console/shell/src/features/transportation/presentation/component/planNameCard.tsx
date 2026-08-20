// frontend/console/shell/src/features/transportation/presentation/component/planNameCard.tsx

import * as React from "react";
import { Tag } from "lucide-react";

import { Card, CardContent, CardHeader, CardTitle } from "../../../../shared/ui/card";
import { Input } from "../../../../shared/ui/input";

export type PlanNameCardProps = {
  name: string;
  disabled?: boolean;
  className?: string;
  onChangeName: (name: string) => void;
};

const PlanNameCard: React.FC<PlanNameCardProps> = ({
  name,
  disabled = false,
  className,
  onChangeName,
}) => {
  const handleChange = React.useCallback(
    (event: React.ChangeEvent<HTMLInputElement>) => {
      onChangeName(event.target.value);
    },
    [onChangeName],
  );

  return (
    <Card className={className}>
      <CardHeader>
        <div className="flex items-center gap-2">
          <Tag size={18} aria-hidden="true" />
          <CardTitle className="text-base">料金設定名</CardTitle>
        </div>
      </CardHeader>

      <CardContent>
        <div className="space-y-2">
          <label htmlFor="transportation-plan-name" className="text-sm font-medium text-slate-700">
            プラン名
          </label>

          <Input
            id="transportation-plan-name"
            type="text"
            disabled={disabled}
            value={name}
            maxLength={100}
            placeholder="例: 通常配送"
            className="w-full"
            onChange={handleChange}
          />

          <p className="text-xs text-slate-500">
            配送料金設定を識別するための名前を入力してください。
          </p>
        </div>
      </CardContent>
    </Card>
  );
};

export default PlanNameCard;