// frontend/console/shell/src/features/admin/presentation/components/AdminCard.tsx

import * as React from "react";

import { Button } from "../../../../shared/ui/button";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../../../../shared/ui/card";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "../../../../shared/ui/popover";
import Stack from "../../../../shared/ui/stack";

import { useAdminCard as useAdminCardHook } from "../hook/useAdminCard";

import "../styles/admin.css";

export type AdminAssigneeCandidate = {
  id: string;
  name: string;
};

export type AdminCardProps = {
  title?: string;
  /**
   * 指定した場合、担当者欄の代わりに
   * 同じ位置へ宛先数を表示する。
   */
  targetAvatarCount?: number;
  showAssignee?: boolean;
  assigneeLabel?: string;
  assigneeName?: string;
  assigneeId?: string;
  assigneeCandidates?: AdminAssigneeCandidate[];
  loadingMembers?: boolean;
  onSelectAssignee?: (id: string) => void;
  createdByName?: string | null;
  createdAt?: string | null;
  updatedByName?: string | null;
  updatedAt?: string | null;
  onEditAssignee?: () => void;
  onClickAssignee?: () => void;
  mode?: "edit" | "view";
};

const closePopover = () =>
  document.dispatchEvent(
    new KeyboardEvent("keydown", {
      key: "Escape",
    }),
  );

export const AdminCard: React.FC<AdminCardProps> = ({
  title = "管理情報",
  targetAvatarCount,
  showAssignee = true,
  assigneeLabel = "担当者",
  assigneeName,
  assigneeId,
  assigneeCandidates,
  loadingMembers,
  onSelectAssignee,
  createdByName,
  createdAt,
  updatedByName,
  updatedAt,
  onEditAssignee,
  onClickAssignee,
  mode = "view",
}) => {
  const isEdit = mode === "edit";
  const showsTargetAvatarCount = typeof targetAvatarCount === "number";

  const {
    assigneeCandidates: hookAssigneeCandidates,
    loadingMembers: hookLoadingMembers,
  } = useAdminCardHook();

  const effectiveCandidates =
    assigneeCandidates ??
    hookAssigneeCandidates ??
    [];

  const effectiveLoading =
    typeof loadingMembers === "boolean"
      ? loadingMembers
      : hookLoadingMembers;

  const effectiveAssigneeName = assigneeName ?? "未設定";

  const selectedValue = React.useMemo(() => {
    const normalizedId = assigneeId?.trim() ?? "";

    if (normalizedId) {
      return normalizedId;
    }

    const matched = effectiveCandidates.find(
      (candidate) => candidate.name === effectiveAssigneeName,
    );

    return matched?.id ?? "";
  }, [assigneeId, effectiveCandidates, effectiveAssigneeName]);

  const selectedCandidateName = React.useMemo(() => {
    const selectedCandidate = effectiveCandidates.find(
      (candidate) => candidate.id === selectedValue,
    );

    return (
      selectedCandidate?.name ||
      effectiveAssigneeName ||
      `${assigneeLabel}を選択してください`
    );
  }, [
    assigneeLabel,
    effectiveAssigneeName,
    effectiveCandidates,
    selectedValue,
  ]);

  const handleSelectAssignee = React.useCallback(
    (nextId: string) => {
      if (!isEdit || !nextId) {
        return;
      }

      onClickAssignee?.();
      onEditAssignee?.();
      onSelectAssignee?.(nextId);
      closePopover();
    },
    [isEdit, onClickAssignee, onEditAssignee, onSelectAssignee],
  );

  return (
    <Card>
      <CardHeader>
        <CardTitle className="admin-card__title">
          {title}
        </CardTitle>
      </CardHeader>

      <CardContent>
        <Stack gap="md">
          {showsTargetAvatarCount ? (
            <div>
              <div className="admin-card__field-label">
                宛先数
              </div>
              <div className="admin-card__field-value">
                {targetAvatarCount}件
              </div>
            </div>
          ) : showAssignee ? (
            <div>
              <div className="admin-card__field-label">
                {assigneeLabel}
              </div>

              {!isEdit ? (
                <div className="admin-card__field-value">
                  {effectiveAssigneeName}
                </div>
              ) : (
                <>
                  <Popover>
                    <PopoverTrigger>
                      <Button
                        type="button"
                        variant="outline"
                        className="admin-card__assignee-trigger"
                        disabled={effectiveLoading}
                        aria-label={`${assigneeLabel}を選択`}
                      >
                        {effectiveLoading
                          ? `${assigneeLabel}を読み込み中です…`
                          : selectedCandidateName}
                      </Button>
                    </PopoverTrigger>

                    <PopoverContent
                      align="start"
                      className="popover__content--compact popover__content--medium"
                    >
                      {effectiveLoading ? (
                        <div className="popover__empty">
                          {assigneeLabel}を読み込み中です…
                        </div>
                      ) : effectiveCandidates.length === 0 ? (
                        <div className="popover__empty">
                          {assigneeLabel}候補がありません。
                        </div>
                      ) : (
                        <div className="popover__list">
                          {effectiveCandidates.map((candidate) => {
                            const isSelected =
                              candidate.id === selectedValue;

                            return (
                              <button
                                key={candidate.id}
                                type="button"
                                className={`popover__item${isSelected ? " is-active" : ""}`}
                                onClick={() =>
                                  handleSelectAssignee(candidate.id)
                                }
                              >
                                {candidate.name}
                              </button>
                            );
                          })}
                        </div>
                      )}
                    </PopoverContent>
                  </Popover>

                  {!effectiveLoading &&
                    effectiveCandidates.length === 0 && (
                      <p className="admin-card__empty-assignee">
                        {assigneeLabel}候補がありません。
                      </p>
                    )}
                </>
              )}
            </div>
          ) : null}

          {(createdByName || createdAt || updatedByName || updatedAt) && (
            <Stack gap="xs" className="admin-card__metadata">
              {(createdByName || createdAt) && (
                <div className="admin-card__metadata-row">
                  {createdByName && <span>作成者: {createdByName}</span>}
                  {createdAt && <span>作成日: {createdAt}</span>}
                </div>
              )}

              {(updatedByName || updatedAt) && (
                <div className="admin-card__metadata-row">
                  {updatedByName && <span>更新者: {updatedByName}</span>}
                  {updatedAt && <span>更新日: {updatedAt}</span>}
                </div>
              )}
            </Stack>
          )}
        </Stack>
      </CardContent>
    </Card>
  );
};

export default AdminCard;