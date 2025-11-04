// frontend/shell/src/app/App.tsx
import { BrowserRouter } from "react-router-dom";
import MainPage from "../pages/MainPage";

export default function App() {

  return (
    <BrowserRouter>
      <div className="min-h-screen flex flex-col bg-slate-900 text-white">
          <main className="flex-1 p-6 overflow-y-auto">
            <MainPage /> {/* ← MainがRoutesを管理 */}
          </main>
        </div>
    </BrowserRouter>
  );
}
