// frontend/shell/src/layout/PageStyle/PageStyle.tsx
import type { ReactNode } from "react";
import { ArrowLeft, Save } from "lucide-react";
import "./PageStyle.css";

/** クラス結合ヘルパー */
function cn(...classes: Array<string | undefined | false | null>) {
  return classes.filter(Boolean).join(" ");
}

interface PageStyleProps {
  /** ページ全体の背景/文字色/高さなどを適用 */
  children: ReactNode | [ReactNode, ReactNode]; // grid-2想定時は左右2要素
  /** レイアウトバリエーション */
  layout?: "grid-2" | "single";
  /** 追加クラス */
  className?: string;
  /** 戻るボタン押下時ハンドラ（任意） */
  onBack?: () => void;
  /** 保存/作成ボタン押下時ハンドラ（任意） */
  onSave?: () => void;
  /** ページタイトル（任意） */
  title?: string;
  /** タイトル横に表示するバッジ（任意） */
  badge?: ReactNode;
  /** 右側の追加アクション（任意） */
  actions?: ReactNode;
  /** 右ペインを sticky にするか（デフォルト true, grid-2 のみ） */
  stickyAside?: boolean;
}

/**
 * PageStyle
 * - ページ全体スタイル（背景・高さ・文字色）を適用
 * - 内部に共通 PageHeader を持つ
 * - layout="grid-2": 左右2カラム + 右ペイン sticky
 * - layout="single" or 未指定: 単一カラム
 */
export default function PageStyle({
  children,
  layout = "single",
  className,
  onBack,
  onSave,
  title,
  badge,
  actions,
  stickyAside = true,
}: PageStyleProps) {
  const rootClass = cn("pbp", className);
  const hasBack = Boolean(onBack);
  const handleBack = onBack ?? (() => {});

  // 共通ヘッダー
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

          {/* 右側：保存/作成ボタン + 追加アクション */}
          <div className="page-header__actions">
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

  // 2カラムレイアウト
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

  // 単一カラムレイアウト（layout === "single"）
  return (
    <div className={rootClass}>
      {header}
      <div className="page-container">{children}</div>
    </div>
  );
}
