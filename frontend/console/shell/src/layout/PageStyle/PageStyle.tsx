// frontend/shell/src/layout/PageStyle/PageStyle.tsx
import type { ReactNode } from "react";
import {
  ArrowLeft,
  Save,
  Plus,
  Pencil,
  Trash2,
  X, // ★ 追加
} from "lucide-react";
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
  onCreate?: () => void;

  // 追加済み
  onEdit?: () => void;
  onDelete?: () => void;

  // ★ 新規追加（キャンセル）
  onCancel?: () => void;

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
  onCreate,
  onEdit,
  onDelete,
  onCancel, // ★ 追加
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
          {/* 左側：戻る + タイトル */}
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

          {/* 右側：アクションボタン */}
          <div className="page-header__actions">
            {/* 編集 */}
            {onEdit && (
              <button
                type="button"
                className="page-header__btn"
                onClick={onEdit}
              >
                <Pencil size={16} style={{ marginRight: 4 }} />
                編集
              </button>
            )}

            {/* 削除 */}
            {onDelete && (
              <button
                type="button"
                className="page-header__btn page-header__btn--danger"
                onClick={onDelete}
              >
                <Trash2 size={16} style={{ marginRight: 4 }} />
                削除
              </button>
            )}

            {/* ★ キャンセル */}
            {onCancel && (
              <button
                type="button"
                className="page-header__btn page-header__btn--ghost"
                onClick={onCancel}
              >
                <X size={16} style={{ marginRight: 4 }} />
                キャンセル
              </button>
            )}

            {/* 作成 */}
            {onCreate && (
              <button
                type="button"
                className="page-header__btn"
                onClick={onCreate}
              >
                <Plus size={16} style={{ marginRight: 4 }} />
                作成
              </button>
            )}

            {/* 保存 */}
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
