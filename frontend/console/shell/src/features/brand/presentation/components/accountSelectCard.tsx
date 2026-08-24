// frontend/console/shell/src/features/brand/presentation/components/accountSelectCard.tsx

import * as React from "react";

import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "../../../../shared/ui/card";

import { Button } from "../../../../shared/ui/button";

import {
  Popover,
  PopoverTrigger,
  PopoverContent,
} from "../../../../shared/ui/popover";

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

  onSelectAccount?: (id: string) => void;
  onOpenAccountConnect?: () => void;

  mode?: "edit" | "view";
};

function accountStatusLabel(status?: AccountStatus): string {
  switch (status) {
    case "active":
      return "利用中";
    case "inactive":
      return "未利用";
    case "suspended":
      return "停止中";
    case "deleted":
      return "削除済み";
    default:
      return "";
  }
}

export const AccountSelectCard: React.FC<AccountSelectCardProps> = ({
  title = "売上受取口座",
  accountId,
  accountLabel,
  accountCandidates,
  loadingAccounts,
  accountError,
  onSelectAccount,
  onOpenAccountConnect,
  mode = "view",
}) => {
  const isEdit = mode === "edit";
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

  const selectableCandidates = React.useMemo(
    () => candidates.filter((candidate) => candidate.status !== "deleted"),
    [candidates],
  );

  const handleSelect = (id: string) => {
    if (!isEdit) {
      return;
    }

    onSelectAccount?.(id);
  };

  return (
    <Card className="admin-card">
      <CardHeader className="admin-card__header">
        <CardTitle className="admin-card__title">
          {title}
        </CardTitle>
      </CardHeader>

      <CardContent className="admin-card__body space-y-4">
        <div className="admin-card__section">
          <div className="admin-card__label mb-1 text-xs text-slate-500">
            接続口座
          </div>

          {!isEdit && (
            <div className="py-1 text-sm text-slate-800">
              {displayLabel}
            </div>
          )}

          {isEdit && (
            <Popover>
              <PopoverTrigger>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  className="admin-card__assignee-btn w-full justify-between"
                  disabled={loading}
                >
                  <span className="min-w-0 truncate text-left">
                    {displayLabel}
                  </span>

                  <span className="ml-2 shrink-0 text-[11px] text-slate-400">
                    選択
                  </span>
                </Button>
              </PopoverTrigger>

              <PopoverContent className="admin-card__popover space-y-1 p-2">
                {loading && (
                  <p className="text-xs text-slate-400">
                    口座を読み込み中です…
                  </p>
                )}

                {!loading && accountError && (
                  <p className="whitespace-pre-wrap text-xs text-red-500">
                    {accountError}
                  </p>
                )}

                {!loading &&
                  !accountError &&
                  selectableCandidates.length > 0 && (
                    <div className="space-y-1">
                      {selectableCandidates.map((candidate) => {
                        const isSelected = candidate.id === accountId;
                        const statusLabel = accountStatusLabel(candidate.status);

                        return (
                          <button
                            key={candidate.id}
                            type="button"
                            className={[
                              "block w-full rounded px-2 py-1.5 text-left text-sm",
                              "hover:bg-slate-100",
                              isSelected
                                ? "bg-slate-100 font-semibold"
                                : "",
                            ].join(" ")}
                            onClick={() => handleSelect(candidate.id)}
                          >
                            <div className="flex items-center justify-between gap-3">
                              <span className="min-w-0 truncate">
                                {candidate.label || candidate.id}
                              </span>

                              {statusLabel && (
                                <span className="shrink-0 text-[11px] font-normal text-slate-400">
                                  {statusLabel}
                                </span>
                              )}
                            </div>
                          </button>
                        );
                      })}
                    </div>
                  )}

                {!loading &&
                  !accountError &&
                  selectableCandidates.length === 0 && (
                    <div className="space-y-2">
                      <p className="text-xs text-slate-400">
                        選択可能な口座がありません。
                      </p>

                      {onOpenAccountConnect && (
                        <Button
                          type="button"
                          variant="outline"
                          size="sm"
                          className="w-full"
                          onClick={onOpenAccountConnect}
                        >
                          口座を接続する
                        </Button>
                      )}
                    </div>
                  )}
              </PopoverContent>
            </Popover>
          )}
        </div>

        {accountId && (
          <div className="admin-card__section">
            <div className="admin-card__label mb-1 text-xs text-slate-500">
              Account ID
            </div>

            <div className="break-all py-1 text-xs text-slate-500">
              {accountId}
            </div>
          </div>
        )}

        {isEdit && onOpenAccountConnect && (
          <div className="admin-card__section">
            <Button
              type="button"
              variant="outline"
              size="sm"
              className="w-full"
              onClick={onOpenAccountConnect}
            >
              新しい口座を接続する
            </Button>
          </div>
        )}
      </CardContent>
    </Card>
  );
};

export default AccountSelectCard;