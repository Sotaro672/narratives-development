//frontend\shell\src\app\App.tsx
import React, { Suspense, lazy } from "react";
import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import Layout from "../layout/PageFrame";
import DashboardPage from "../pages/DashboardPage";
import NotFoundPage from "../pages/NotFoundPage";

// ──────────────────────────────────────────────
// 各リモートモジュール（Module Federation）を遅延ロード
// ──────────────────────────────────────────────
const OrgModule = lazy(() => import("org/routes"));
const InquiriesModule = lazy(() => import("inquiries/routes"));
const ListingsModule = lazy(() => import("listings/routes"));
const OperationsModule = lazy(() => import("operations/routes"));
const PreviewModule = lazy(() => import("preview/routes"));
const ProductionModule = lazy(() => import("production/routes"));
const DesignModule = lazy(() => import("design/routes"));
const MintModule = lazy(() => import("mint/routes"));
const OrdersModule = lazy(() => import("orders/routes"));
const AdsModule = lazy(() => import("ads/routes"));
const AccountsModule = lazy(() => import("accounts/routes"));
const TransactionsModule = lazy(() => import("transactions/routes"));

// ──────────────────────────────────────────────
// React Query 設定
// ──────────────────────────────────────────────
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      refetchOnWindowFocus: false,
      staleTime: 60000,
    },
  },
});

// ──────────────────────────────────────────────
// アプリ構成
// ──────────────────────────────────────────────
export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <Layout>
          <Suspense fallback={<div className="p-8 text-gray-500">Loading module...</div>}>
            <Routes>
              {/* ダッシュボード */}
              <Route path="/" element={<DashboardPage />} />

              {/* 組織管理 */}
              <Route path="/org/*" element={<OrgModule />} />

              {/* 問い合わせ */}
              <Route path="/inquiries/*" element={<InquiriesModule />} />

              {/* 出品・運用・プレビュー */}
              <Route path="/listings/*" element={<ListingsModule />} />
              <Route path="/operations/*" element={<OperationsModule />} />
              <Route path="/preview/*" element={<PreviewModule />} />

              {/* 生産・設計 */}
              <Route path="/production/*" element={<ProductionModule />} />
              <Route path="/design/*" element={<DesignModule />} />

              {/* ミント・注文 */}
              <Route path="/mint/*" element={<MintModule />} />
              <Route path="/orders/*" element={<OrdersModule />} />

              {/* 広告・財務 */}
              <Route path="/ads/*" element={<AdsModule />} />
              <Route path="/accounts/*" element={<AccountsModule />} />
              <Route path="/transactions/*" element={<TransactionsModule />} />

              {/* リダイレクト・404 */}
              <Route path="/crm" element={<Navigate to="/" replace />} />
              <Route path="*" element={<NotFoundPage />} />
            </Routes>
          </Suspense>
        </Layout>
      </BrowserRouter>
    </QueryClientProvider>
  );
}
