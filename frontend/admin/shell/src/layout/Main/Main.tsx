//frontend\admin\shell\src\layout\Main\Main.tsx
import {
  Navigate,
  Route,
  Routes,
} from "react-router-dom";

import BillingPage from "../../pages/BillingPage";
import ContractsPage from "../../pages/ContractsPage";
import GasPage from "../../pages/GasPage";
import InquiriesPage from "../../pages/InquiriesPage";
import ReportsPage from "../../pages/ReportsPage";

import "./Main.css";

export default function Main() {
  return (
    <main className="main-area">
      <div className="main-content">
        <Routes>
          <Route
            path="/"
            element={
              <Navigate
                to="/inquiries"
                replace
              />
            }
          />

          <Route
            path="/inquiries"
            element={<InquiriesPage />}
          />

          <Route
            path="/gas"
            element={<GasPage />}
          />

          <Route
            path="/contracts"
            element={<ContractsPage />}
          />

          <Route
            path="/reports"
            element={<ReportsPage />}
          />

          <Route
            path="/billing"
            element={<BillingPage />}
          />

          <Route
            path="*"
            element={
              <Navigate
                to="/inquiries"
                replace
              />
            }
          />
        </Routes>
      </div>
    </main>
  );
}