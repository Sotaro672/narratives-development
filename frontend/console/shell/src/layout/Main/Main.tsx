// frontend/shell/src/layout/Main/Main.tsx
import { useRoutes } from "react-router-dom";
import routes from "../../routes/routes"; // ルーティング定義をインポート
import "./Main.css";

/**
 * Main
 * - Sidebar・Header を除いた「右側のメインエリア」
 * - routes.tsx に定義されたルーティングをここで描画
 * - Main領域のみスクロール可能（全体は固定）
 */
export default function Main() {
  // react-router-dom の useRoutes でルート配列を解釈
  const element = useRoutes(routes);

  return (
    // Main.css の .main-area でスクロール制御
    <div className="main-area">
      {element}
    </div>
  );
}
