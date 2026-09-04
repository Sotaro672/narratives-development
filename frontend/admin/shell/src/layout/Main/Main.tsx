// frontend/admin/shell/src/layout/Main/Main.tsx
import MainRoutes from "./MainRoutes";

import "./Main.css";

export default function Main() {
  return (
    <main className="main-area">
      <div className="main-content">
        <MainRoutes />
      </div>
    </main>
  );
}