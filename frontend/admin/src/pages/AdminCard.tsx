import * as React from "react";
import { Card, CardContent, CardHeader, CardTitle } from "../../../shared/ui/card";
import { Edit2 } from "lucide-react";
import "./AdminCard.css";

type Props = {
  title?: string;
  assigneeName: string;   // 例: "佐藤 美咲"
  createdByName: string;  // 例: "山田 太郎"
  createdAt: string;      // 例: "2024/1/20"
  onEditAssignee?: () => void;
  onClickAssignee?: () => void;
  onClickCreatedBy?: () => void;
};

export default function AdminCard({
  title = "管理情報",
  assigneeName,
  createdByName,
  createdAt,
  onEditAssignee,
  onClickAssignee,
  onClickCreatedBy,
}: Props) {
  return (
    <Card className="admin-card">
      {/* ヘッダー */}
      <CardHeader className="admin-card__header">
        <CardTitle className="admin-card__title">{title}</CardTitle>
      </CardHeader>

      {/* 本体 */}
      <CardContent className="admin-card__body">
        {/* 担当者 行 */}
        <div className="admin-card__row">
          <div className="admin-card__label">担当者</div>
          <button
            type="button"
            className="admin-card__btn"
            onClick={onEditAssignee}
            aria-label="担当者を編集"
            title="担当者を編集"
          >
            <Edit2 size={16} />
          </button>
        </div>

        {/* 担当者名（リンク風） */}
        <div>
          <span
            className="admin-card__link"
            onClick={onClickAssignee}
            role={onClickAssignee ? "button" : undefined}
          >
            {assigneeName}
          </span>
        </div>

        {/* セパレータ */}
        <div className="admin-card__divider" />

        {/* 作成者 / 作成日時 2カラム */}
        <div className="admin-card__meta">
          <div className="admin-card__metaCol">
            <div className="admin-card__subLabel">作成者</div>
            <span
              className="admin-card__link"
              onClick={onClickCreatedBy}
              role={onClickCreatedBy ? "button" : undefined}
            >
              {createdByName}
            </span>
          </div>

          <div className="admin-card__metaCol">
            <div className="admin-card__subLabel">作成日時</div>
            <div className="admin-card__date">{createdAt}</div>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
