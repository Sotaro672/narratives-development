// frontend/admin/src/presentation/components/AdminCard.tsx
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../../../../shell/src/shared/ui/card";
import "../styles/admin.css";

import {
  Popover,
  PopoverTrigger,
  PopoverContent,
} from "../../../../shell/src/shared/ui/popover";

export type AdminAssigneeCandidate = {
  id: string;
  name: string;
};

export type AdminCardProps = {
  title?: string;
  assigneeName: string;

  assigneeCandidates?: AdminAssigneeCandidate[];
  loadingMembers?: boolean;

  // Popover の開閉状態（外から監視したい場合用。今回は実質未使用）
  openAssigneePopover: boolean;
  setOpenAssigneePopover: (v: boolean) => void;

  onSelectAssignee: (id: string) => void;

  createdByName?: string | null;
  createdAt?: string | null;
  updatedByName?: string | null;
  updatedAt?: string | null;
};

export default function AdminCard(props: AdminCardProps) {
  const {
    title = "管理情報",
    assigneeName,
    assigneeCandidates,
    loadingMembers,
    // 今回の実装では Popover 側の内部 state に任せるので、実際には使わない
    openAssigneePopover,
    setOpenAssigneePopover,
    onSelectAssignee,
    createdByName,
    createdAt,
    updatedByName,
    updatedAt,
  } = props;

  // assigneeCandidates が undefined の場合でも map できるように安全化
  const safeCandidates: AdminAssigneeCandidate[] = assigneeCandidates ?? [];

  const safeCreatedBy = createdByName ?? "―";
  const safeCreatedAt = createdAt ?? "―";
  const safeUpdatedBy = updatedByName ?? "―";
  const safeUpdatedAt = updatedAt ?? "―";

  const hasMeta =
    Boolean(createdByName || createdAt || updatedByName || updatedAt);

  return (
    <Card className="admin-card">
      <CardHeader className="admin-card__header">
        <CardTitle className="admin-card__title">{title}</CardTitle>
      </CardHeader>

      <CardContent className="admin-card__body">
        {/* 担当者（唯一の編集可フィールド） */}
        <div className="admin-card__row">
          <div className="admin-card__label">担当者</div>

          {/* ▼ assigneeName 自体がプルダウンボタン */}
          <Popover>
            <PopoverTrigger>
              <button
                type="button"
                className="admin-card__assigneeButton"
              >
                <span className="admin-card__assigneeButtonText">
                  {assigneeName}
                </span>
                <span className="admin-card__assigneeButtonCaret">▾</span>
              </button>
            </PopoverTrigger>

            <PopoverContent align="start">
              {loadingMembers && <div>読み込み中...</div>}

              {!loadingMembers && safeCandidates.length === 0 && (
                <div className="admin-card__assigneeEmpty">
                  メンバーが登録されていません
                </div>
              )}

              {!loadingMembers && safeCandidates.length > 0 && (
                <ul className="admin-card__assigneeList">
                  {safeCandidates.map((c) => (
                    <li
                      key={c.id}
                      className="admin-card__assigneeItem"
                      onClick={() => onSelectAssignee(c.id)}
                    >
                      {c.name}
                    </li>
                  ))}
                </ul>
              )}
            </PopoverContent>
          </Popover>
        </div>

        {/* メタ情報が無ければ高さを縮める（担当者だけのカードになる） */}
        {hasMeta && (
          <>
            <div className="admin-card__divider" />

            <div className="admin-card__meta">
              <div className="admin-card__metaCol">
                <div className="admin-card__subLabel">作成者</div>
                <span className="admin-card__readonly">{safeCreatedBy}</span>
              </div>
              <div className="admin-card__metaCol">
                <div className="admin-card__subLabel">作成日時</div>
                <span className="admin-card__readonly">{safeCreatedAt}</span>
              </div>
            </div>

            <div className="admin-card__meta">
              <div className="admin-card__metaCol">
                <div className="admin-card__subLabel">更新者</div>
                <span className="admin-card__readonly">{safeUpdatedBy}</span>
              </div>
              <div className="admin-card__metaCol">
                <div className="admin-card__subLabel">更新日時</div>
                <span className="admin-card__readonly">{safeUpdatedAt}</span>
              </div>
            </div>
          </>
        )}
      </CardContent>
    </Card>
  );
}
