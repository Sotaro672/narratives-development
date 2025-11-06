// frontend/shell/src/layout/PageHeader/PageHeader.tsx
import type { ReactNode } from "react";
import { Button } from "../../../../shared/ui/button";
import { ArrowLeft, Save } from "lucide-react";
import "./PageHeader.css";

interface PageHeaderProps {
  /** ページタイトル */
  title: string;
  /** 戻るボタン押下時ハンドラ */
  onBack: () => void;
  /** 保存ボタン押下時ハンドラ（任意） */
  onSave?: () => void;
  /** 右側の追加アクション（任意） */
  actions?: ReactNode;
  /** タイトル横に表示するバッジ（任意） */
  badge?: ReactNode;
}

const PageHeader = ({
  title,
  onBack,
  onSave,
  actions,
  badge,
}: PageHeaderProps) => {
  return (
    <div className="sticky top-0 z-10 border-b bg-card/95 backdrop-blur supports-[backdrop-filter]:bg-card/60 page-header">
      <div className="px-4 py-3">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-4">
            <Button variant="ghost" size="sm" onClick={onBack}>
              <ArrowLeft className="w-4 h-4" />
            </Button>
            <div className="flex items-center gap-2">
              <h1 className="page-header__title">{title}</h1>
              {badge}
            </div>
          </div>
          <div className="flex items-center gap-2 page-header__actions">
            {onSave && (
              <Button
                variant="default"
                size="sm"
                className="page-header__btn"
                onClick={onSave}
              >
                <Save size={16} className="mr-1" /> 保存
              </Button>
            )}
            {actions}
          </div>
        </div>
      </div>
    </div>
  );
};

export default PageHeader;
