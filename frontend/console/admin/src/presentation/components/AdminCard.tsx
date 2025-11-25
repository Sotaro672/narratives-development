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

export type AdminAssigneeCandidate = {
  id: string;
  name: string;
};

export type AdminCardProps = {
  // タイトル
  title?: string;

  // 表示中の担当者名（画面表示用のみ）
  assigneeName?: string;

  // 担当者候補一覧
  assigneeCandidates?: AdminAssigneeCandidate[];
  loadingMembers?: boolean;

  // ポップオーバー開閉制御
  openAssigneePopover?: boolean;
  setOpenAssigneePopover?: (v: boolean) => void;
  onSelectAssignee?: (id: string) => void;

  // 作成 / 更新情報
  createdByName?: string | null;
  createdAt?: string | null;
  updatedByName?: string | null;
  updatedAt?: string | null;

  // 各種コールバック
  onEditAssignee?: () => void;
  onClickAssignee?: () => void;
};

export const AdminCard: React.FC<AdminCardProps> = ({
  title = "管理情報",
  assigneeName = "未設定",
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
}) => {
  const handleTriggerClick = () => {
    onClickAssignee?.();
    onEditAssignee?.();

    if (typeof openAssigneePopover === "boolean" && setOpenAssigneePopover) {
      setOpenAssigneePopover(!openAssigneePopover);
    }
  };

  const handleSelect = (id: string) => {
    onSelectAssignee?.(id);
    // Popover の open/close は親側に任せる
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

          <Popover>
            <PopoverTrigger>
              <Button
                type="button"
                variant="outline"
                size="sm"
                className="w-full justify-between admin-card__assignee-btn"
                onClick={handleTriggerClick}
              >
                {/* ▼ ここで assigneeId ではなく assigneeName を表示 */}
                <span>{assigneeName || "未設定"}</span>
                <span className="text-[11px] text-slate-400" />
              </Button>
            </PopoverTrigger>

            <PopoverContent className="p-2 space-y-1 admin-card__popover">
              {loadingMembers && (
                <p className="text-xs text-slate-400">
                  担当者を読み込み中です…
                </p>
              )}

              {!loadingMembers &&
                assigneeCandidates &&
                assigneeCandidates.length > 0 && (
                  <div className="space-y-1">
                    {assigneeCandidates.map((c) => (
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

              {!loadingMembers &&
                (!assigneeCandidates || assigneeCandidates.length === 0) && (
                  <p className="text-xs text-slate-400">
                    担当者候補がありません。
                  </p>
                )}
            </PopoverContent>
          </Popover>
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
