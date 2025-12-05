// frontend/console/admin/src/presentation/components/AdminCard.tsx

import * as React from "react";

import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "../../../../shell/src/shared/ui/card";

import { Button } from "../../../../shell/src/shared/ui/button";

import {
  Popover,
  PopoverTrigger,
  PopoverContent,
} from "../../../../shell/src/shared/ui/popover";

import "../styles/admin.css";

// ★ Admin 用 hook（内部で AdminService + getDisplayName を使用）
import { useAdminCard as useAdminCardHook } from "../hook/useAdminCard";

export type AdminAssigneeCandidate = {
  id: string;
  name: string;
};

export type AdminCardProps = {
  title?: string;

  assigneeName?: string;

  assigneeCandidates?: AdminAssigneeCandidate[];
  loadingMembers?: boolean;

  openAssigneePopover?: boolean;
  setOpenAssigneePopover?: (v: boolean) => void;
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
  assigneeCandidates,
  loadingMembers,

  openAssigneePopover,
  setOpenAssigneePopover,
  onSelectAssignee,

  createdByName,
  createdAt,
  updatedByName,
  updatedAt,

  onEditAssignee,
  onClickAssignee,

  mode = "view", // デフォルト＝view
}) => {
  const isEdit = mode === "edit";

  // ★ hook 側（AdminService + getDisplayName）の値
  const {
    assigneeName: hookAssigneeName,
    assigneeCandidates: hookAssigneeCandidates,
    loadingMembers: hookLoadingMembers,
    openAssigneePopover: hookOpenAssigneePopover,
    setOpenAssigneePopover: hookSetOpenAssigneePopover,
    onSelectAssignee: hookOnSelectAssignee,
  } = useAdminCardHook();

  // ★ 担当者名の優先順位をモード別に変更
  //   - edit モード: hook の displayName を最優先
  //   - view モード: props を優先（既存画面との互換性）
  const effectiveAssigneeName = isEdit
    ? hookAssigneeName ?? assigneeName ?? "未設定"
    : assigneeName ?? hookAssigneeName ?? "未設定";

  const effectiveCandidates =
    assigneeCandidates ?? hookAssigneeCandidates ?? [];

  const effectiveLoading =
    typeof loadingMembers === "boolean"
      ? loadingMembers
      : hookLoadingMembers;

  const effectiveOpen =
    typeof openAssigneePopover === "boolean"
      ? openAssigneePopover
      : hookOpenAssigneePopover;

  const effectiveSetOpen =
    setOpenAssigneePopover ?? hookSetOpenAssigneePopover;

  const handleTriggerClick = () => {
    if (!isEdit) return;

    onClickAssignee?.();
    onEditAssignee?.();

    if (typeof effectiveOpen === "boolean" && effectiveSetOpen) {
      effectiveSetOpen(!effectiveOpen);
    }
  };

  const handleSelect = (id: string) => {
    if (!isEdit) return;

    // まず hook 側の選択処理
    hookOnSelectAssignee?.(id);
    // 親コンポーネントへも通知
    onSelectAssignee?.(id);
  };

  return (
    <Card className="admin-card">
      <CardHeader className="admin-card__header">
        <CardTitle className="admin-card__title">{title}</CardTitle>
      </CardHeader>

      <CardContent className="admin-card__body space-y-4">
        {/* 担当者 */}
        <div className="admin-card__section">
          <div className="admin-card__label text-xs text-slate-500 mb-1">
            担当者
          </div>

          {/* ▼▼▼ view モード：文字列だけ表示 ▼▼▼ */}
          {!isEdit && (
            <div className="text-sm text-slate-800 py-1">
              {effectiveAssigneeName || "未設定"}
            </div>
          )}

          {/* ▼▼▼ edit モード：ポップオーバー付き ▼▼▼ */}
          {isEdit && (
            <Popover>
              <PopoverTrigger>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  className="w-full justify-between admin-card__assignee-btn"
                  onClick={handleTriggerClick}
                >
                  <span>{effectiveAssigneeName || "未設定"}</span>
                  <span className="text-[11px] text-slate-400" />
                </Button>
              </PopoverTrigger>

              <PopoverContent className="p-2 space-y-1 admin-card__popover">
                {effectiveLoading && (
                  <p className="text-xs text-slate-400">
                    担当者を読み込み中です…
                  </p>
                )}

                {!effectiveLoading &&
                  effectiveCandidates &&
                  effectiveCandidates.length > 0 && (
                    <div className="space-y-1">
                      {effectiveCandidates.map((c) => (
                        <button
                          key={c.id}
                          type="button"
                          className="block w-full text-left px-2 py-1 rounded hover:bg-slate-100 text-sm"
                          onClick={() => handleSelect(c.id)}
                        >
                          {c.name}
                        </button>
                      ))}
                    </div>
                  )}

                {!effectiveLoading &&
                  (!effectiveCandidates ||
                    effectiveCandidates.length === 0) && (
                    <p className="text-xs text-slate-400">
                      担当者候補がありません。
                    </p>
                  )}
              </PopoverContent>
            </Popover>
          )}
        </div>

        {/* 作成 / 更新情報 */}
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
