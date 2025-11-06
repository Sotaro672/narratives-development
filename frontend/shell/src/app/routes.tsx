// frontend/shell/src/app/routes.tsx
import type { RouteObject } from "react-router-dom";

// 単ページのインポート
import MemberManagementPage from "../../../member/src/pages/memberManagement";
import BrandManagementPage from "../../../brand/src/pages/brandManagement";
import PermissionListPage from "../../../permission/src/pages/permissionList";
import AdManagementPage from "../../../ad/src/pages/adManagement";
import AccountManagementPage from "../../../account/src/pages/accountManagement";
import TransactionListPage from "../../../transaction/src/pages/transactionList";

// モジュールのルート定義（型衝突を避けるため unknown→RouteObject[] にキャスト）
import inquiryRoutesRaw from "../../../inquiry/src/routes";
const inquiryRoutes = inquiryRoutesRaw as unknown as RouteObject[];

import productBlueprintRoutesRaw from "../../../productBlueprint/src/routes";
const productBlueprintRoutes = productBlueprintRoutesRaw as unknown as RouteObject[];

import productionRoutesRaw from "../../../production/src/routes";
const productionRoutes = productionRoutesRaw as unknown as RouteObject[];

import inventoryRoutesRaw from "../../../inventory/src/routes";
const inventoryRoutes = inventoryRoutesRaw as unknown as RouteObject[];

import tokenBlueprintRoutesRaw from "../../../tokenBlueprint/src/routes";
const tokenBlueprintRoutes = tokenBlueprintRoutesRaw as unknown as RouteObject[];

import mintRequestRoutesRaw from "../../../mintRequest/src/routes";
const mintRequestRoutes = mintRequestRoutesRaw as unknown as RouteObject[];

import operationRoutesRaw from "../../../operation/src/routes";
const operationRoutes = operationRoutesRaw as unknown as RouteObject[];

import listRoutesRaw from "../../../list/src/routes";
const listRoutes = listRoutesRaw as unknown as RouteObject[];

import orderRoutesRaw from "../../../order/src/routes";
const orderRoutes = orderRoutesRaw as unknown as RouteObject[];
/**
 * Shell全体で使用するルーティング定義
 * - Layout (Main.tsx) からインポートされる
 * - 各モジュールの routes.tsx を children として統合
 */
// Inquiry モジュール
export const routes: RouteObject[] = [
  { path: "/inquiry", 
    children: inquiryRoutes 
  },

  // ProductBlueprint モジュール
  {
    path: "/productBlueprint",
    children: productBlueprintRoutes,
  },

  // Production モジュール
  {
    path: "/production",
    children: productionRoutes,
  },

  // Inventory モジュール
  {
    path: "/inventory",
    children: inventoryRoutes,
  },
  // TokenBlueprint モジュール
  {
    path: "/tokenBlueprint",
    children: tokenBlueprintRoutes,
  },
  // MintRequest モジュール
  {
    path: "/mintRequest",
    children: mintRequestRoutes,
  },
  // TokenOperation モジュール
  {
    path: "/operation",
    children: operationRoutes,
  },
  // Listings モジュール
  {
    path: "/list",
    children: listRoutes,
  },
  // Orders モジュール
  {
    path: "/order",
    children: orderRoutes,
  },

  { path: "/member", element: <MemberManagementPage /> },
  { path: "/brand", element: <BrandManagementPage /> },
  { path: "/permission", element: <PermissionListPage /> },
  { path: "/ad", element: <AdManagementPage /> },
  { path: "/account", element: <AccountManagementPage /> },
  { path: "/transaction", element: <TransactionListPage /> },
];

export default routes;
