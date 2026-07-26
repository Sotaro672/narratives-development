// frontend/console/shell/src/features/brand/presentation/components/ManagerCard.tsx

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

export type ManagerCandidate = {
  id: string;
  name: string;
};

export type ManagerCardProps = {
  title?: string;

  managerName?: string;
  managerId?: string | null;

  managerCandidates?: ManagerCandidate[];
  loadingMembers?: boolean;
  memberError?: string | null;

  onSelectManager?: (id: string) => void;

  registeredAt?: string | null;
  updatedAt?: string | null;

  onEditManager?: () => void;
  onClickManager?: () => void;

  mode?: "edit" | "view";
};

export const ManagerCard: React.FC<ManagerCardProps> = ({
  title = "管理情報",

  managerName,
  managerId,

  managerCandidates,
  loadingMembers,
  memberError,

  onSelectManager,

  registeredAt,
  updatedAt,

  onEditManager,
  onClickManager,

  mode = "view",
}) => {
  const isEdit = mode === "edit";

  const effectiveManagerName =
    managerName || managerId || "未設定";

  const effectiveCandidates = managerCandidates ?? [];

  const effectiveLoading = Boolean(loadingMembers);

  const handleTriggerClick = () => {
    if (!isEdit) {
      return;
    }

    onClickManager?.();
    onEditManager?.();
  };

  const handleSelect = (id: string) => {
    if (!isEdit) {
      return;
    }

    onSelectManager?.(id);
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
            責任者
          </div>

          {!isEdit && (
            <div className="py-1 text-sm text-slate-800">
              {effectiveManagerName}
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
                  onClick={handleTriggerClick}
                >
                  <span>{effectiveManagerName}</span>

                  <span className="text-[11px] text-slate-400">
                    選択
                  </span>
                </Button>
              </PopoverTrigger>

              <PopoverContent className="admin-card__popover space-y-1 p-2">
                {effectiveLoading && (
                  <p className="text-xs text-slate-400">
                    責任者を読み込み中です…
                  </p>
                )}

                {!effectiveLoading && memberError && (
                  <p className="whitespace-pre-wrap text-xs text-red-500">
                    {memberError}
                  </p>
                )}

                {!effectiveLoading &&
                  !memberError &&
                  effectiveCandidates.length > 0 && (
                    <div className="space-y-1">
                      {effectiveCandidates.map((candidate) => {
                        const isSelected =
                          candidate.id === managerId;

                        return (
                          <button
                            key={candidate.id}
                            type="button"
                            className={[
                              "block w-full rounded px-2 py-1 text-left text-sm",
                              "hover:bg-slate-100",
                              isSelected
                                ? "bg-slate-100 font-semibold"
                                : "",
                            ].join(" ")}
                            onClick={() =>
                              handleSelect(candidate.id)
                            }
                          >
                            {candidate.name}
                          </button>
                        );
                      })}
                    </div>
                  )}

                {!effectiveLoading &&
                  !memberError &&
                  effectiveCandidates.length === 0 && (
                    <p className="text-xs text-slate-400">
                      責任者候補がありません。
                    </p>
                  )}
              </PopoverContent>
            </Popover>
          )}
        </div>

        {(registeredAt || updatedAt) && (
          <div className="admin-card__section space-y-1 text-xs text-slate-500">
            {registeredAt && (
              <div>登録日: {registeredAt}</div>
            )}

            {updatedAt && (
              <div>更新日: {updatedAt}</div>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  );
};

export default ManagerCard;