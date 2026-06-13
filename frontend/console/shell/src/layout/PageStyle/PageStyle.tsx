import * as React from "react";
import type { ReactNode } from "react";
import {
  ArrowLeft,
  Save,
  Plus,
  Pencil,
  Trash2,
  X,
  Tag,
} from "lucide-react";
import "./PageStyle.css";

function cn(...classes: Array<string | undefined | false | null>) {
  return classes.filter(Boolean).join(" ");
}

function SpinnerArrow({ size = 16 }: { size?: number }) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      aria-hidden="true"
      style={{ marginRight: 4, display: "inline-block", verticalAlign: "middle" }}
    >
      <g>
        <animateTransform
          attributeName="transform"
          type="rotate"
          from="0 12 12"
          to="360 12 12"
          dur="0.9s"
          repeatCount="indefinite"
        />
        <path
          d="M21 12a9 9 0 1 1-2.64-6.36"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
        />
        <path
          d="M21 3v6h-6"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
        />
      </g>
    </svg>
  );
}

interface PageStyleProps {
  children: ReactNode | [ReactNode, ReactNode];
  layout?: "grid-2" | "single";
  className?: string;

  onBack?: () => void | Promise<void>;
  onSave?: () => void | Promise<void>;
  onCreate?: () => void | Promise<void>;

  onEdit?: () => void | Promise<void>;
  onDelete?: () => void | Promise<void>;

  onCancel?: () => void | Promise<void>;

  onPurge?: () => void | Promise<void>;

  onList?: () => void | Promise<void>;

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
  onList,
  title,
  badge,
  actions,
  stickyAside = true,
}: PageStyleProps) {
  const rootClass = cn("pbp", className);
  const hasBack = Boolean(onBack);
  const handleBack = onBack ?? (() => {});

  const [isCreating, setIsCreating] = React.useState(false);
  const [isSaving, setIsSaving] = React.useState(false);
  const [isListing, setIsListing] = React.useState(false);

  const handleCreate = React.useCallback(async () => {
    if (!onCreate || isCreating) return;

    try {
      setIsCreating(true);
      await onCreate();
    } finally {
      setIsCreating(false);
    }
  }, [onCreate, isCreating]);

  const handleSave = React.useCallback(async () => {
    if (!onSave || isSaving) return;

    try {
      setIsSaving(true);
      await onSave();
    } finally {
      setIsSaving(false);
    }
  }, [onSave, isSaving]);

  const handleList = React.useCallback(async () => {
    if (!onList || isListing) return;

    try {
      setIsListing(true);
      await onList();
    } finally {
      setIsListing(false);
    }
  }, [onList, isListing]);

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

            {onPurge && (
              <button
                type="button"
                className="page-header__btn page-header__btn--danger"
                onClick={onPurge}
              >
                <Trash2 size={16} style={{ marginRight: 4 }} />
                削除
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

            {onList && (
              <button
                type="button"
                className="page-header__btn"
                onClick={() => void handleList()}
                disabled={isListing}
                aria-busy={isListing}
              >
                {isListing ? (
                  <SpinnerArrow size={16} />
                ) : (
                  <Tag size={16} style={{ marginRight: 4 }} />
                )}
                {isListing ? "出品中" : "出品"}
              </button>
            )}

            {onCreate && (
              <button
                type="button"
                className="page-header__btn"
                onClick={() => void handleCreate()}
                disabled={isCreating}
                aria-busy={isCreating}
              >
                {isCreating ? (
                  <SpinnerArrow size={16} />
                ) : (
                  <Plus size={16} style={{ marginRight: 4 }} />
                )}
                {isCreating ? "作成中" : "作成"}
              </button>
            )}

            {onSave && (
              <button
                type="button"
                className="page-header__btn"
                onClick={() => void handleSave()}
                disabled={isSaving}
                aria-busy={isSaving}
              >
                {isSaving ? (
                  <SpinnerArrow size={16} />
                ) : (
                  <Save size={16} style={{ marginRight: 4 }} />
                )}
                {isSaving ? "保存中" : "保存"}
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