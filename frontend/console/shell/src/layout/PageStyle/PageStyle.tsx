//frontend\console\shell\src\layout\PageStyle\PageStyle.tsx
import type { ReactNode } from "react";
import {
  ArrowLeft,
  Save,
  Plus,
  Pencil,
  Trash2,
  X,
  RotateCw, // ★ 復旧（Restore）
  Tag, // ★ 出品
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

  onEdit?: () => void;
  onDelete?: () => void; // 論理削除

  // ★ キャンセル
  onCancel?: () => void;

  // ★ 新規追加
  onPurge?: () => void; // 物理削除
  onRestore?: () => void; // 復旧

  // ★ 出品
  onList?: () => void;

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
  onCancel,
  onPurge,
  onRestore,
  onList, // ★ 出品
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

            {onRestore && (
              <button
                type="button"
                className="page-header__btn"
                onClick={onRestore}
              >
                <RotateCw size={16} style={{ marginRight: 4 }} />
                復旧
              </button>
            )}

            {onPurge && (
              <button
                type="button"
                className="page-header__btn page-header__btn--danger"
                onClick={onPurge}
              >
                <Trash2 size={16} style={{ marginRight: 4 }} />
                物理削除
              </button>
            )}

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

            {/* ★ 出品 */}
            {onList && (
              <button
                type="button"
                className="page-header__btn"
                onClick={onList}
              >
                <Tag size={16} style={{ marginRight: 4 }} />
                出品
              </button>
            )}

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

  return (
    <div className={rootClass}>
      {header}
      <div className="page-container">{children}</div>
    </div>
  );
}
