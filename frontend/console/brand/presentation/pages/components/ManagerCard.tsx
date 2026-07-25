// frontend/console/brand/src/presentation/pages/components/ManagerCard.tsx

import * as React from "react";

import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "../../../../../shell/src/shared/ui/card";

import { Button } from "../../../../../shell/src/shared/ui/button";

import {
  Popover,
  PopoverTrigger,
  PopoverContent,
} from "../../../../../shell/src/shared/ui/popover";

import "../../styles/brand.css";

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

  openManagerPopover?: boolean;
  setOpenManagerPopover?: (v: boolean) => void;
  onSelectManager?: (id: string) => void;

  registeredAt?: string | null;
  updatedAt?: string | null;

  onEditManager?: () => void;
  onClickManager?: () => void;

  /** 表示モード（編集 or 閲覧） */
  mode?: "edit" | "view";
};

export const ManagerCard: React.FC<ManagerCardProps> = ({
  title = "管理情報",

  managerName,
  managerId,

  managerCandidates,
  loadingMembers,

  openManagerPopover,
  setOpenManagerPopover,
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

  const effectiveOpen =
    typeof openManagerPopover === "boolean" ? openManagerPopover : false;

  const effectiveSetOpen = setOpenManagerPopover;

  const handleTriggerClick = () => {
    if (!isEdit) return;

    onClickManager?.();
    onEditManager?.();

    if (typeof effectiveOpen === "boolean" && effectiveSetOpen) {
      effectiveSetOpen(!effectiveOpen);
    }
  };

  const handleSelect = (id: string) => {
    if (!isEdit) return;
    onSelectManager?.(id);
  };

  return (
    <Card className="admin-card">
      <CardHeader className="admin-card__header">
        <CardTitle className="admin-card__title">{title}</CardTitle>
      </CardHeader>

      <CardContent className="admin-card__body space-y-4">
        {/* 責任者 */}
        <div className="admin-card__section">
          <div className="admin-card__label text-xs text-slate-500 mb-1">
            責任者
          </div>

          {/* view */}
          {!isEdit && (
            <div className="text-sm text-slate-800 py-1">
              {effectiveManagerName || "未設定"}
            </div>
          )}

          {/* edit */}
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
                  <span>{effectiveManagerName || "未設定"}</span>
                  <span className="text-[11px] text-slate-400" />
                </Button>
              </PopoverTrigger>

              <PopoverContent className="p-2 space-y-1 admin-card__popover">
                {effectiveLoading && (
                  <p className="text-xs text-slate-400">
                    責任者を読み込み中です…
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
                      責任者候補がありません。
                    </p>
                  )}
              </PopoverContent>
            </Popover>
          )}
        </div>

        {/* 登録 / 更新（AdminCard と同じ “情報ブロック” 形式） */}
        {(registeredAt || updatedAt) && (
          <div className="admin-card__section space-y-1 text-xs text-slate-500">
            {registeredAt && <div>登録日: {registeredAt}</div>}
            {updatedAt && <div>更新日: {updatedAt}</div>}
          </div>
        )}
      </CardContent>
    </Card>
  );
};

export default ManagerCard;