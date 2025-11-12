// frontend/shell/src/layout/PageStyle/PageStyle.tsx
import type { ReactNode } from "react";
import { ArrowLeft, Save, Plus } from "lucide-react"; // ← Plus を追加
import "./PageStyle.css";

function cn(...classes: Array<string | undefined | false | null>) {
  return classes.filter(Boolean).join(" ");
}

interface PageStyleProps {
  children: ReactNode | [ReactNode, ReactNode];
  layout?: "grid-2" | "single";
  className?: string;
  onBack?: () => void;
  onSave?: () => void;
  onCreate?: () => void; // ← 追加
  title?: string;
  badge?: ReactNode;
  actions?: ReactNode;
  stickyAside?: boolean;
}

export default function PageStyle({
  children,
  layout = "single",
  className,
  onBack,
  onSave,
  onCreate, // ← 追加
  title,
  badge,
  actions,
  stickyAside = true,
}: PageStyleProps) {
  const rootClass = cn("pbp", className);
  const hasBack = Boolean(onBack);
  const handleBack = onBack ?? (() => {});

  const header = (
    <header className="page-header">
      <div className="px-4 py-3">
        <div className="flex items-center justify-between">
          <div className="page-header__left">
            {hasBack && (
              <button
                type="button"
                className="page-header__back"
                onClick={handleBack}
                aria-label="戻る"
              >
                <ArrowLeft size={16} />
              </button>
            )}
            <div className="flex items-center gap-2">
              <h1 className="page-header__title">{title ?? ""}</h1>
              {badge}
            </div>
          </div>

          <div className="page-header__actions">
            {onCreate && ( // ← 追加
              <button
                type="button"
                className="page-header__btn"
                onClick={onCreate}
              >
                <Plus size={16} style={{ marginRight: 4 }} />
                作成
              </button>
            )}
            {onSave && (
              <button
                type="button"
                className="page-header__btn"
                onClick={onSave}
              >
                <Save size={16} style={{ marginRight: 4 }} />
                保存
              </button>
            )}
            {actions}
          </div>
        </div>
      </div>
    </header>
  );

  if (layout === "grid-2") {
    const [left, right] = Array.isArray(children) ? children : [children, null];

    return (
      <div className={rootClass}>
        {header}
        <div className="page-container">
          <div className="content-grid">
            <div>{left}</div>
            <div className={stickyAside ? "sticky-aside" : undefined}>
              {right}
            </div>
          </div>
        </div>
      </div>
    );
  }

  // single
  return (
    <div className={rootClass}>
      {header}
      <div className="page-container">{children}</div>
    </div>
  );
}
