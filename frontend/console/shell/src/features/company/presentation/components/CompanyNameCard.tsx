// frontend/console/shell/src/features/company/presentation/components/CompanyNameCard.tsx

import * as React from "react";

import {
  Card,
  CardContent,
  CardHeader,
  CardInput,
  CardLabel,
  CardTitle,
} from "../../../../shared/ui/card";

export type CompanyNameCardProps = {
  companyName: string;
  onChangeCompanyName: (value: string) => void;
  disabled?: boolean;
};

export const CompanyNameCard: React.FC<CompanyNameCardProps> = ({
  companyName,
  onChangeCompanyName,
  disabled = false,
}) => {
  const handleChange = React.useCallback(
    (event: React.ChangeEvent<HTMLInputElement>) => {
      onChangeCompanyName(event.target.value);
    },
    [onChangeCompanyName],
  );

  return (
    <Card>
      <CardHeader>
        <CardTitle>会社名</CardTitle>
      </CardHeader>

      <CardContent>
        <CardInput
          id="company-name"
          name="companyName"
          type="text"
          value={companyName}
          onChange={handleChange}
          placeholder="会社名を入力してください"
          autoComplete="organization"
          maxLength={100}
          disabled={disabled}
        />

        <p className="mt-2 text-xs text-[hsl(var(--muted-foreground))]">
          Consoleに表示する会社名を設定します。
        </p>
      </CardContent>
    </Card>
  );
};

export default CompanyNameCard;