//frontend\console\admin\src\presentation\components\AdminCard.tsx
import * as React from "react";

import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "../../../../shell/src/shared/ui/card";

import "../styles/admin.css";

// ★ Admin 用 hook（候補一覧取得）
import { useAdminCard as useAdminCardHook } from "../hook/useAdminCard";

export type AdminAssigneeCandidate = {
  id: string;
  name: string;
};

export type AdminCardProps = {
  title?: string;

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

  /** 表示モード（編集 or 閲覧） */
  mode?: "edit" | "view";
};

export const AdminCard: React.FC<AdminCardProps> = ({
  title = "管理情報",
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

  const {
    assigneeCandidates: hookAssigneeCandidates,
    loadingMembers: hookLoadingMembers,
  } = useAdminCardHook();

  const effectiveCandidates =
    assigneeCandidates ?? hookAssigneeCandidates ?? [];

  const effectiveLoading =
    typeof loadingMembers === "boolean"
      ? loadingMembers
      : hookLoadingMembers;

  const effectiveAssigneeName = assigneeName ?? "未設定";

  const selectedValue = React.useMemo(() => {
    const normalizedId = String(assigneeId ?? "").trim();
    if (normalizedId) return normalizedId;

    const matched = effectiveCandidates.find((c) => c.name === effectiveAssigneeName);
    return matched?.id ?? "";
  }, [assigneeId, effectiveCandidates, effectiveAssigneeName]);

  const handleChange = React.useCallback(
    (e: React.ChangeEvent<HTMLSelectElement>) => {
      if (!isEdit) return;

      const nextId = String(e.target.value ?? "").trim();
      if (!nextId) return;

      onClickAssignee?.();
      onEditAssignee?.();
      onSelectAssignee?.(nextId);
    },
    [isEdit, onClickAssignee, onEditAssignee, onSelectAssignee],
  );

  return (
    <Card className="admin-card">
      <CardHeader className="admin-card__header">
        <CardTitle className="admin-card__title">{title}</CardTitle>
      </CardHeader>

      <CardContent className="admin-card__body space-y-4">
        <div className="admin-card__section">
          <div className="admin-card__label text-xs text-slate-500 mb-1">
            担当者
          </div>

          {!isEdit && (
            <div className="text-sm text-slate-800 py-1">
              {effectiveAssigneeName || "未設定"}
            </div>
          )}

          {isEdit && (
            <>
              <select
                className="w-full rounded-md border border-slate-200 bg-white px-3 py-2 text-sm outline-none focus:border-slate-400"
                value={selectedValue}
                onChange={handleChange}
                disabled={effectiveLoading}
              >
                <option value="" disabled>
                  {effectiveLoading ? "担当者を読み込み中です…" : "担当者を選択してください"}
                </option>

                {effectiveCandidates.map((c) => (
                  <option key={c.id} value={c.id}>
                    {c.name}
                  </option>
                ))}
              </select>

              {!effectiveLoading && effectiveCandidates.length === 0 && (
                <p className="mt-2 text-xs text-slate-400">
                  担当者候補がありません。
                </p>
              )}
            </>
          )}
        </div>

        {(createdByName || createdAt || updatedByName || updatedAt) && (
          <div className="admin-card__section space-y-1 text-xs text-slate-500">
            {createdByName && <div>作成者: {createdByName}</div>}
            {createdAt && <div>作成日: {createdAt}</div>}
            {updatedByName && <div>最終更新者: {updatedByName}</div>}
            {updatedAt && <div>最終更新日: {updatedAt}</div>}
          </div>
        )}
      </CardContent>
    </Card>
  );
};

export default AdminCard;