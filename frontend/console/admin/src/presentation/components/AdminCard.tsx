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

  // 表示中の担当者名（スタイル側では文字列をそのまま表示するだけ）
  assigneeName?: string;

  // 担当者候補一覧（スタイル側では map してボタン表示するだけ）
  assigneeCandidates?: AdminAssigneeCandidate[];
  loadingMembers?: boolean;

  // ポップオーバー開閉制御（必要なら上位から渡す。未指定なら AdminCard 側では特に制御しない）
  openAssigneePopover?: boolean;
  setOpenAssigneePopover?: (v: boolean) => void;
  onSelectAssignee?: (id: string) => void;

  // 作成 / 更新情報（あれば表示）
  createdByName?: string | null;
  createdAt?: string | null;
  updatedByName?: string | null;
  updatedAt?: string | null;

  // クリックや編集操作を上位に通知するためのコールバック
  onEditAssignee?: () => void;
  onClickAssignee?: () => void;
};

export const AdminCard: React.FC<AdminCardProps> = ({
  title = "管理情報",
  assigneeName = "未設定",
  assigneeCandidates,
  loadingMembers,

  // controlled 用（必要なら上位が使う。AdminCard 側では open 値を直接は使わない）
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
  // スタイル側では「トリガーが押された」という事実だけ上位へ通知
  const handleTriggerClick = () => {
    onClickAssignee?.();
    onEditAssignee?.();

    // 上位が openAssigneePopover を使っている場合だけ補助的にトグル
    if (typeof openAssigneePopover === "boolean" && setOpenAssigneePopover) {
      setOpenAssigneePopover(!openAssigneePopover);
    }
  };

  const handleSelect = (id: string) => {
    onSelectAssignee?.(id);
    // Popover の開閉自体はコンポーネント内では制御しない（Radix 側のデフォルト動作に任せる）
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
              {/* asChild は使わず、Button をそのまま children に渡すだけ */}
              <Button
                type="button"
                variant="outline"
                size="sm"
                className="w-full justify-between admin-card__assignee-btn"
                onClick={handleTriggerClick}
              >
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

        {/* 作成 / 更新情報（あれば表示） */}
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

// default export も用意しておくと、
// import AdminCard from ".../AdminCard"
// でも
// import { AdminCard } from ".../AdminCard"
// でも利用可能
export default AdminCard;
