// frontend/console/shell/src/features/brand/presentation/components/accountSelectCard.tsx

import * as React from "react";

import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "../../../../shared/ui/card";
import { ErrorMessage } from "../../../../shared/ui/error";

import "../../../../styles/brand.css";

export type AccountStatus =
  | "active"
  | "inactive"
  | "suspended"
  | "deleted";

export type AccountCandidate = {
  id: string;
  label: string;
  status?: AccountStatus;
};

export type AccountSelectCardProps = {
  title?: string;
  accountId?: string | null;
  accountLabel?: string | null;
  accountCandidates?: AccountCandidate[];
  loadingAccounts?: boolean;
  accountError?: string | null;
};

export const AccountSelectCard: React.FC<AccountSelectCardProps> = ({
  title = "売上受取口座",
  accountId,
  accountLabel,
  accountCandidates,
  loadingAccounts,
  accountError,
}) => {
  const candidates = accountCandidates ?? [];
  const loading = Boolean(loadingAccounts);

  const selectedAccount = React.useMemo(
    () => candidates.find((candidate) => candidate.id === accountId) ?? null,
    [candidates, accountId],
  );

  const displayLabel =
    accountLabel ||
    selectedAccount?.label ||
    accountId ||
    "未設定";

  return (
    <Card className="admin-card">
      <CardHeader className="admin-card__header">
        <CardTitle className="admin-card__title">
          {title}
        </CardTitle>
      </CardHeader>

      <CardContent className="admin-card__body account-select-card__body">
        <div className="admin-card__section">
          <div className="admin-card__label account-select-card__label">
            接続口座
          </div>

          {loading ? (
            <div className="account-select-card__loading">
              口座を読み込み中です…
            </div>
          ) : accountError ? (
            <ErrorMessage size="xs" className="account-select-card__error">
              {accountError}
            </ErrorMessage>
          ) : (
            <div className="account-select-card__value">
              {displayLabel}
            </div>
          )}
        </div>

        {accountId && (
          <div className="admin-card__section">
            <div className="admin-card__label account-select-card__label">
              Account ID
            </div>

            <div className="account-select-card__id">
              {accountId}
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  );
};

export default AccountSelectCard;