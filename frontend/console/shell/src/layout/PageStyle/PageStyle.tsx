// frontend/console/shell/src/layout/PageStyle/PageStyle.tsx
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
  RefreshCw,
  Send,
  MessageSquareReply,
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
      style={{
        marginRight: 4,
        display: "inline-block",
        verticalAlign: "middle",
      }}
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

type HeaderStatusButtonVariant = "default" | "danger" | "neutral";

interface PageStyleProps {
  children: ReactNode | [ReactNode, ReactNode];
  layout?: "grid-2" | "single";
  className?: string;

  onBack?: () => void | Promise<void>;
  onSave?: () => void | Promise<void>;
  isSaving?: boolean;
  onSend?: () => void | Promise<void>;
  isSending?: boolean;
  onReply?: () => void | Promise<void>;
  isReplying?: boolean;
  onCreate?: () => void | Promise<void>;
  onRefresh?: () => void | Promise<void>;
  isRefreshing?: boolean;

  onEdit?: () => void | Promise<void>;
  onDelete?: () => void | Promise<void>;

  onCancel?: () => void | Promise<void>;

  onClose?: () => void | Promise<void>;
  isClosing?: boolean;

  onPurge?: () => void | Promise<void>;

  onList?: () => void | Promise<void>;

  title?: ReactNode;
  badge?: ReactNode;
  actions?: ReactNode;

  statusButtonLabel?: string;
  statusButtonBusyLabel?: string;
  statusButtonVariant?: HeaderStatusButtonVariant;
  onStatusButtonClick?: () => void | Promise<void>;
  isStatusButtonLoading?: boolean;
  statusButtonDisabled?: boolean;

  stickyAside?: boolean;
}

export default function PageStyle({
  children,
  layout = "single",
  className,
  onBack,
  onSave,
  isSaving: controlledIsSaving,
  onSend,
  isSending: controlledIsSending,
  onReply,
  isReplying: controlledIsReplying,
  onCreate,
  onRefresh,
  isRefreshing: controlledIsRefreshing,
  onEdit,
  onDelete,
  onCancel,
  onClose,
  isClosing: controlledIsClosing,
  onPurge,
  onList,
  title,
  badge,
  actions,
  statusButtonLabel,
  statusButtonBusyLabel,
  statusButtonVariant = "default",
  onStatusButtonClick,
  isStatusButtonLoading: controlledIsStatusButtonLoading,
  statusButtonDisabled,
  stickyAside = true,
}: PageStyleProps) {
  const rootClass = cn("pbp", className);
  const hasBack = Boolean(onBack);
  const handleBack = onBack ?? (() => {});

  const [isCreating, setIsCreating] = React.useState(false);
  const [internalIsSaving, setInternalIsSaving] = React.useState(false);
  const [internalIsSending, setInternalIsSending] = React.useState(false);
  const [internalIsReplying, setInternalIsReplying] = React.useState(false);
  const [isListing, setIsListing] = React.useState(false);
  const [internalIsRefreshing, setInternalIsRefreshing] = React.useState(false);
  const [internalIsClosing, setInternalIsClosing] = React.useState(false);
  const [internalIsStatusButtonLoading, setInternalIsStatusButtonLoading] =
    React.useState(false);

  const isSaving = controlledIsSaving ?? internalIsSaving;
  const isSending = controlledIsSending ?? internalIsSending;
  const isReplying = controlledIsReplying ?? internalIsReplying;
  const isRefreshing = controlledIsRefreshing ?? internalIsRefreshing;
  const isClosing = controlledIsClosing ?? internalIsClosing;
  const isStatusButtonLoading =
    controlledIsStatusButtonLoading ?? internalIsStatusButtonLoading;

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
      setInternalIsSaving(true);
      await onSave();
    } finally {
      setInternalIsSaving(false);
    }
  }, [onSave, isSaving]);

  const handleSend = React.useCallback(async () => {
    if (!onSend || isSending) return;

    try {
      setInternalIsSending(true);
      await onSend();
    } finally {
      setInternalIsSending(false);
    }
  }, [onSend, isSending]);

  const handleReply = React.useCallback(async () => {
    if (!onReply || isReplying) return;

    try {
      setInternalIsReplying(true);
      await onReply();
    } finally {
      setInternalIsReplying(false);
    }
  }, [onReply, isReplying]);

  const handleList = React.useCallback(async () => {
    if (!onList || isListing) return;

    try {
      setIsListing(true);
      await onList();
    } finally {
      setIsListing(false);
    }
  }, [onList, isListing]);

  const handleRefresh = React.useCallback(async () => {
    if (!onRefresh || isRefreshing) return;

    try {
      setInternalIsRefreshing(true);
      await onRefresh();
    } finally {
      setInternalIsRefreshing(false);
    }
  }, [onRefresh, isRefreshing]);

  const handleClose = React.useCallback(async () => {
    if (!onClose || isClosing) return;

    try {
      setInternalIsClosing(true);
      await onClose();
    } finally {
      setInternalIsClosing(false);
    }
  }, [onClose, isClosing]);

  const handleStatusButtonClick = React.useCallback(async () => {
    if (
      !onStatusButtonClick ||
      isStatusButtonLoading ||
      statusButtonDisabled
    ) {
      return;
    }

    try {
      setInternalIsStatusButtonLoading(true);
      await onStatusButtonClick();
    } finally {
      setInternalIsStatusButtonLoading(false);
    }
  }, [onStatusButtonClick, isStatusButtonLoading, statusButtonDisabled]);

  const statusButtonClassName = cn(
    "page-header__btn",
    statusButtonVariant === "danger" && "page-header__btn--danger",
    statusButtonVariant === "neutral" && "page-header__btn--ghost",
  );

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
            {onStatusButtonClick && statusButtonLabel && (
              <button
                type="button"
                className={statusButtonClassName}
                onClick={() => void handleStatusButtonClick()}
                disabled={isStatusButtonLoading || Boolean(statusButtonDisabled)}
                aria-busy={isStatusButtonLoading}
              >
                {isStatusButtonLoading && <SpinnerArrow size={16} />}
                {isStatusButtonLoading
                  ? statusButtonBusyLabel ?? "更新中"
                  : statusButtonLabel}
              </button>
            )}

            {onReply && (
              <button
                type="button"
                className="page-header__btn"
                onClick={() => void handleReply()}
                disabled={isReplying}
                aria-busy={isReplying}
              >
                {isReplying ? (
                  <SpinnerArrow size={16} />
                ) : (
                  <MessageSquareReply size={16} style={{ marginRight: 4 }} />
                )}
                {isReplying ? "準備中" : "返信"}
              </button>
            )}

            {onRefresh && (
              <button
                type="button"
                className="page-header__btn"
                onClick={() => void handleRefresh()}
                disabled={isRefreshing}
                aria-busy={isRefreshing}
              >
                {isRefreshing ? (
                  <SpinnerArrow size={16} />
                ) : (
                  <RefreshCw size={16} style={{ marginRight: 4 }} />
                )}
                {isRefreshing ? "更新中" : "更新"}
              </button>
            )}

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

            {onClose && (
              <button
                type="button"
                className="page-header__btn page-header__btn--ghost"
                onClick={() => void handleClose()}
                disabled={isClosing}
                aria-busy={isClosing}
              >
                {isClosing ? (
                  <SpinnerArrow size={16} />
                ) : (
                  <X size={16} style={{ marginRight: 4 }} />
                )}
                {isClosing ? "クローズ中" : "クローズ"}
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

            {onSend && (
              <button
                type="button"
                className="page-header__btn"
                onClick={() => void handleSend()}
                disabled={isSending}
                aria-busy={isSending}
              >
                {isSending ? (
                  <SpinnerArrow size={16} />
                ) : (
                  <Send size={16} style={{ marginRight: 4 }} />
                )}
                {isSending ? "送信中" : "送信"}
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