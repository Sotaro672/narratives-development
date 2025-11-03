// frontend/shell/src/layout/Main/Main.tsx
import type { ReactNode } from "react";
import "./Main.css"; // スタイルをCSSへ移譲

/**
 * Main
 * 画面から Header と Sidebar を除いた表示領域。
 * CSS 変数でレイアウトを制御します（未定義時はデフォルト値を使用）。
 *  - --header-h  : ヘッダー高さ（既定 64px）
 *  - --sidebar-w : サイドバー幅（既定 280px）
 */
type MainProps = {
  children: ReactNode;
  className?: string;
};

export default function Main({ children, className }: MainProps) {
  return (
    <main role="main" className={`main-area ${className ?? ""}`}>
      <div className="p-6">{children}</div>
    </main>
  );
}