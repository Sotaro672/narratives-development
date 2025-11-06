// frontend/shell/src/layout/PageStyle/PageStyle.tsx
import type { ReactNode } from "react";
import PageHeader from "../PageHeader/PageHeader";
import "./PageStyle.css";

/** クラス結合ヘルパー */
function cn(...classes: Array<string | undefined | false | null>) {
  return classes.filter(Boolean).join(" ");
}

interface PageStyleProps {
  /** ページ全体の背景/文字色/高さなどを適用 */
  children: ReactNode | [ReactNode, ReactNode]; // grid-2想定時は左右2要素
  /** レイアウトバリエーション（未指定なら単一カラム） */
  layout?: "none" | "grid-2";
  /** 追加クラス */
  className?: string;
  /** 戻るボタン押下時ハンドラ */
  onBack?: () => void;
  /** 保存ボタン押下時ハンドラ（任意） */
  onSave?: () => void;
  /** ページタイトル（任意） */
  title?: string;
  /** タイトル横に表示するバッジ（任意） */
  badge?: ReactNode;
  /** 右側の追加アクション（任意） */
  actions?: ReactNode;
  /** 右ペインを sticky にするか（デフォルト true） */
  stickyAside?: boolean;
}

/**
 * PageStyle
 * - ページ全体スタイル（背景・高さ・文字色）を適用
 * - PageHeader を上辺（grid の外）に配置
 * - layout="grid-2": 左右2カラム + 右ペイン sticky
 */
export default function PageStyle({
  children,
  layout = "none",
  className,
  onBack,
  onSave,
  title,
  badge,
  actions,
  stickyAside = true,
}: PageStyleProps) {
  const rootClass = cn("pbp", className);

  if (layout === "grid-2") {
    const [left, right] = Array.isArray(children) ? children : [children, null];

    return (
      <div className={rootClass}>
        {/* PageHeader を grid の外に配置 */}
        <PageHeader
          title={title ?? ""}
          onBack={onBack ?? (() => {})}
          onSave={onSave}
          badge={badge}
          actions={actions}
        />

        {/* grid 本体 */}
        <div className="page-container">
          <div className="content-grid">
            <div>{left}</div>
            <div className={stickyAside ? "sticky-aside" : undefined}>{right}</div>
          </div>
        </div>
      </div>
    );
  }

  // 単一カラムレイアウト
  return (
    <div className={rootClass}>
      <PageHeader
        title={title ?? ""}
        onBack={onBack ?? (() => {})}
        onSave={onSave}
        badge={badge}
        actions={actions}
      />

      <div className="page-container">{children}</div>
    </div>
  );
}

