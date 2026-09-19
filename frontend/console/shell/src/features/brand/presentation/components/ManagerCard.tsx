// frontend/console/shell/src/features/brand/presentation/components/ManagerCard.tsx

import * as React from "react";

import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "../../../../shared/ui/card";
import { Button } from "../../../../shared/ui/button";
import { ErrorMessage } from "../../../../shared/ui/error";
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

      <CardContent className="admin-card__body manager-card__body">
        <div className="admin-card__section">
          <div className="admin-card__label manager-card__label">
            責任者
          </div>

          {!isEdit && (
            <div className="manager-card__value">
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
                  className="admin-card__assignee-btn manager-card__trigger"
                  onClick={handleTriggerClick}
                >
                  <span>{effectiveManagerName}</span>
                  <span className="manager-card__trigger-label">
                    選択
                  </span>
                </Button>
              </PopoverTrigger>

              <PopoverContent className="admin-card__popover manager-card__popover">
                {effectiveLoading && (
                  <p className="manager-card__message">
                    責任者を読み込み中です…
                  </p>
                )}

                {!effectiveLoading && memberError && (
                  <ErrorMessage
                    as="p"
                    size="xs"
                    className="manager-card__error"
                  >
                    {memberError}
                  </ErrorMessage>
                )}

                {!effectiveLoading &&
                  !memberError &&
                  effectiveCandidates.length > 0 && (
                    <div className="manager-card__candidate-list">
                      {effectiveCandidates.map((candidate) => {
                        const isSelected =
                          candidate.id === managerId;

                        return (
                          <button
                            key={candidate.id}
                            type="button"
                            className={[
                              "manager-card__candidate",
                              isSelected
                                ? "manager-card__candidate--selected"
                                : "",
                            ]
                              .filter(Boolean)
                              .join(" ")}
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
                    <p className="manager-card__message">
                      責任者候補がありません。
                    </p>
                  )}
              </PopoverContent>
            </Popover>
          )}
        </div>

        {(registeredAt || updatedAt) && (
          <div className="admin-card__section manager-card__dates">
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