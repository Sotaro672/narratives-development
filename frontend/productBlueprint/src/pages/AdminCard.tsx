import * as React from "react";
import { User2, Pencil, Calendar } from "lucide-react";

type AdminCardProps = {
  assignee: string;
  creator: string;
  createdAt: string;
  /** 既存の動作に合わせて「新担当者」に更新するだけの簡易ハンドラ */
  onEditAssignee?: (nextAssignee: string) => void;
};

const AdminCard: React.FC<AdminCardProps> = ({
  assignee,
  creator,
  createdAt,
  onEditAssignee,
}) => {
  const handleEdit = () => {
    // 仕様どおり固定で「新担当者」に更新
    onEditAssignee?.("新担当者");
  };

  return (
    <aside className="box">
      <header className="box__header">
        <User2 size={16} /> <h2 className="box__title">管理情報</h2>
      </header>
      <div className="box__body">
        <div className="label">担当者</div>
        <div className="flex gap-8">
          <input className="readonly" value={assignee} readOnly />
          <button className="btn btn--icon" onClick={handleEdit}>
            <Pencil size={16} />
          </button>
        </div>

        <div className="label">作成者</div>
        <input className="readonly" value={creator} readOnly />

        <div className="label">作成日時</div>
        <div className="pill">
          <Calendar size={14} style={{ marginRight: 4 }} /> {createdAt}
        </div>
      </div>
    </aside>
  );
};

export default AdminCard;
