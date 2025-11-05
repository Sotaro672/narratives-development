import * as React from "react";
import { Save } from "lucide-react";
import "./PageHeader.css";

type PageHeaderProps = {
  /** ページタイトル */
  title: string;
  /** 保存時のクリックハンドラ */
  onSave?: () => void;
};

const PageHeader: React.FC<PageHeaderProps> = ({ title, onSave }) => {
  return (
    <div className="page-header">
      <h1 className="page-header__title">{title}</h1>
      <div className="page-header__actions">
        <button className="page-header__btn" onClick={onSave}>
          <Save size={16} /> 保存
        </button>
      </div>
    </div>
  );
};

export default PageHeader;
