// frontend/shell/src/layout/Main/Main.tsx
import { Routes, Route } from "react-router-dom";
import routes from "../../app/routes";
import "./Main.css";

/**
 * Main Layout
 * - スタイル/レイアウトの責務のみ保持
 * - 実際のルーティングは app/routes.tsx に委譲
 */
export default function Main() {
  return (
    <div className="main-content">
      <Routes>
        {routes.map(({ path, element }) => (
          <Route key={path} path={path} element={element} />
        ))}
      </Routes>
    </div>
  );
}
