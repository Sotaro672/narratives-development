// frontend/console/shell/src/layout/PageStyle/PageStyle.tsx

import * as React from "react";
import type { ReactNode } from "react";
import {
  ArrowLeft,
  FileSpreadsheet,
  Link2,
  MessageSquareReply,
  Pencil,
  Plus,
  QrCode,
  Save,
  Send,
  Tag,
  Trash2,
  X,
} from "lucide-react";

import { Button, type BtnVariant } from "../../shared/ui/button";
import RefreshButton from "../../shared/ui/refresh";

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

function resolveStatusButtonVariant(
  variant: HeaderStatusButtonVariant,
): BtnVariant {
  switch (variant) {
    case "danger":
      return "destructive";
    case "neutral":
      return "ghost";
    case "default":
    default:
      return "outline";
  }
}

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
  onConnect?: () => void | Promise<void>;
  isConnecting?: boolean;
  connectLabel?: string;
  connectBusyLabel?: string;
  connectDisabled?: boolean;
  onRefresh?: () => void | Promise<void>;
  isRefreshing?: boolean;
  onEdit?: () => void | Promise<void>;
  onDelete?: () => void | Promise<void>;
  onCancel?: () => void | Promise<void>;
  onClose?: () => void | Promise<void>;
  isClosing?: boolean;
  onPurge?: () => void | Promise<void>;
  onList?: () => void | Promise<void>;
  onQrOutput?: () => void | Promise<void>;
  isQrOutputting?: boolean;
  qrOutputDisabled?: boolean;
  onCsvOutput?: () => void | Promise<void>;
  isCsvOutputting?: boolean;
  csvOutputDisabled?: boolean;
  title?: ReactNode;
  badge?: ReactNode;
  leadingActions?: ReactNode;
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
  onConnect,
  isConnecting: controlledIsConnecting,
  connectLabel = "接続",
  connectBusyLabel = "接続中",
  connectDisabled,
  onRefresh,
  isRefreshing: controlledIsRefreshing,
  onEdit,
  onDelete,
  onCancel,
  onClose,
  isClosing: controlledIsClosing,
  onPurge,
  onList,
  onQrOutput,
  isQrOutputting: controlledIsQrOutputting,
  qrOutputDisabled,
  onCsvOutput,
  isCsvOutputting: controlledIsCsvOutputting,
  csvOutputDisabled,
  title,
  badge,
  leadingActions,
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
  const [internalIsConnecting, setInternalIsConnecting] = React.useState(false);
  const [isListing, setIsListing] = React.useState(false);
  const [internalIsRefreshing, setInternalIsRefreshing] = React.useState(false);
  const [internalIsClosing, setInternalIsClosing] = React.useState(false);
  const [internalIsQrOutputting, setInternalIsQrOutputting] = React.useState(false);
  const [internalIsCsvOutputting, setInternalIsCsvOutputting] = React.useState(false);
  const [internalIsStatusButtonLoading, setInternalIsStatusButtonLoading] =
    React.useState(false);

  const isSaving = controlledIsSaving ?? internalIsSaving;
  const isSending = controlledIsSending ?? internalIsSending;
  const isReplying = controlledIsReplying ?? internalIsReplying;
  const isConnecting = controlledIsConnecting ?? internalIsConnecting;
  const isRefreshing = controlledIsRefreshing ?? internalIsRefreshing;
  const isClosing = controlledIsClosing ?? internalIsClosing;
  const isQrOutputting = controlledIsQrOutputting ?? internalIsQrOutputting;
  const isCsvOutputting = controlledIsCsvOutputting ?? internalIsCsvOutputting;
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

  const handleConnect = React.useCallback(async () => {
    if (!onConnect || isConnecting || connectDisabled) return;

    try {
      setInternalIsConnecting(true);
      await onConnect();
    } finally {
      setInternalIsConnecting(false);
    }
  }, [onConnect, isConnecting, connectDisabled]);

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

  const handleQrOutput = React.useCallback(async () => {
    if (
      !onQrOutput ||
      isQrOutputting ||
      isCsvOutputting ||
      qrOutputDisabled
    ) {
      return;
    }

    try {
      setInternalIsQrOutputting(true);
      await onQrOutput();
    } finally {
      setInternalIsQrOutputting(false);
    }
  }, [
    onQrOutput,
    isQrOutputting,
    isCsvOutputting,
    qrOutputDisabled,
  ]);

  const handleCsvOutput = React.useCallback(async () => {
    if (
      !onCsvOutput ||
      isCsvOutputting ||
      isQrOutputting ||
      csvOutputDisabled
    ) {
      return;
    }

    try {
      setInternalIsCsvOutputting(true);
      await onCsvOutput();
    } finally {
      setInternalIsCsvOutputting(false);
    }
  }, [
    onCsvOutput,
    isCsvOutputting,
    isQrOutputting,
    csvOutputDisabled,
  ]);

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
  }, [
    onStatusButtonClick,
    isStatusButtonLoading,
    statusButtonDisabled,
  ]);

  const header = (
    <header className="page-header">
      <div className="page-header__inner">
        <div className="page-header__row">
          <div className="page-header__left">
            {hasBack && (
              <Button
                variant="ghost"
                size="icon"
                onClick={() => void handleBack()}
                aria-label="戻る"
              >
                <ArrowLeft />
              </Button>
            )}

            <div className="page-header__title-group">
              <h1 className="page-header__title">{title ?? ""}</h1>
              {badge}
            </div>
          </div>

          <div className="page-header__actions">
            {onStatusButtonClick && statusButtonLabel && (
              <Button
                variant={resolveStatusButtonVariant(statusButtonVariant)}
                size="sm"
                onClick={() => void handleStatusButtonClick()}
                disabled={
                  isStatusButtonLoading ||
                  Boolean(statusButtonDisabled)
                }
                aria-busy={isStatusButtonLoading}
              >
                {isStatusButtonLoading && <SpinnerArrow />}
                {isStatusButtonLoading
                  ? statusButtonBusyLabel ?? "更新中"
                  : statusButtonLabel}
              </Button>
            )}

            {onReply && (
              <Button
                variant="outline"
                size="sm"
                onClick={() => void handleReply()}
                disabled={isReplying}
                aria-busy={isReplying}
              >
                {isReplying ? <SpinnerArrow /> : <MessageSquareReply />}
                {isReplying ? "準備中" : "返信"}
              </Button>
            )}

            {onRefresh && (
              <RefreshButton
                onClick={() => void handleRefresh()}
                loading={isRefreshing}
                title="リフレッシュ"
                ariaLabel="リフレッシュ"
              />
            )}

            {leadingActions}

            {onQrOutput && (
              <Button
                variant="outline"
                size="sm"
                onClick={() => void handleQrOutput()}
                disabled={
                  isQrOutputting ||
                  isCsvOutputting ||
                  Boolean(qrOutputDisabled)
                }
                aria-busy={isQrOutputting}
              >
                {isQrOutputting ? <SpinnerArrow /> : <QrCode />}
                {isQrOutputting ? "QR出力中..." : "QR出力"}
              </Button>
            )}

            {onCsvOutput && (
              <Button
                variant="outline"
                size="sm"
                onClick={() => void handleCsvOutput()}
                disabled={
                  isCsvOutputting ||
                  isQrOutputting ||
                  Boolean(csvOutputDisabled)
                }
                aria-busy={isCsvOutputting}
              >
                {isCsvOutputting ? <SpinnerArrow /> : <FileSpreadsheet />}
                {isCsvOutputting ? "CSV出力中..." : "CSV出力"}
              </Button>
            )}

            {onEdit && (
              <Button
                variant="outline"
                size="sm"
                onClick={() => void onEdit()}
              >
                <Pencil />
                編集
              </Button>
            )}

            {onDelete && (
              <Button
                variant="destructive"
                size="sm"
                onClick={() => void onDelete()}
              >
                <Trash2 />
                削除
              </Button>
            )}

            {onPurge && (
              <Button
                variant="destructive"
                size="sm"
                onClick={() => void onPurge()}
              >
                <Trash2 />
                削除
              </Button>
            )}

            {onCancel && (
              <Button
                variant="ghost"
                size="sm"
                onClick={() => void onCancel()}
              >
                <X />
                キャンセル
              </Button>
            )}

            {onClose && (
              <Button
                variant="ghost"
                size="sm"
                onClick={() => void handleClose()}
                disabled={isClosing}
                aria-busy={isClosing}
              >
                {isClosing ? <SpinnerArrow /> : <X />}
                {isClosing ? "クローズ中" : "クローズ"}
              </Button>
            )}

            {onList && (
              <Button
                variant="outline"
                size="sm"
                onClick={() => void handleList()}
                disabled={isListing}
                aria-busy={isListing}
              >
                {isListing ? <SpinnerArrow /> : <Tag />}
                {isListing ? "出品中" : "出品"}
              </Button>
            )}

            {onCreate && (
              <Button
                variant="outline"
                size="sm"
                onClick={() => void handleCreate()}
                disabled={isCreating}
                aria-busy={isCreating}
              >
                {isCreating ? <SpinnerArrow /> : <Plus />}
                {isCreating ? "作成中" : "作成"}
              </Button>
            )}

            {onConnect && (
              <Button
                variant="outline"
                size="sm"
                onClick={() => void handleConnect()}
                disabled={
                  isConnecting ||
                  Boolean(connectDisabled)
                }
                aria-busy={isConnecting}
              >
                {isConnecting ? <SpinnerArrow /> : <Link2 />}
                {isConnecting ? connectBusyLabel : connectLabel}
              </Button>
            )}

            {onSave && (
              <Button
                variant="outline"
                size="sm"
                onClick={() => void handleSave()}
                disabled={isSaving}
                aria-busy={isSaving}
              >
                {isSaving ? <SpinnerArrow /> : <Save />}
                {isSaving ? "保存中" : "保存"}
              </Button>
            )}

            {onSend && (
              <Button
                variant="outline"
                size="sm"
                onClick={() => void handleSend()}
                disabled={isSending}
                aria-busy={isSending}
              >
                {isSending ? <SpinnerArrow /> : <Send />}
                {isSending ? "送信中" : "送信"}
              </Button>
            )}

            {actions}
          </div>
        </div>
      </div>
    </header>
  );

  if (layout === "grid-2") {
    const [left, right] = Array.isArray(children)
      ? children
      : [children, null];

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