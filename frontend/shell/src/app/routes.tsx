// frontend/shell/src/app/routes.tsx
import type { RouteObject } from "react-router-dom";

// モジュールのルート定義（型衝突を避けるため unknown→RouteObject[] にキャスト）
import announcementRoutesRaw from "../../../announcement/src/routes";
const announcementRoutes = announcementRoutesRaw as unknown as RouteObject[];

import messageRouteRaw from "../../../message/src/routes";
const messageRoutes = messageRouteRaw as unknown as RouteObject[];

import inquiryRoutesRaw from "../../../inquiry/src/presentation/routes/routes";
const inquiryRoutes = inquiryRoutesRaw as unknown as RouteObject[];

import productBlueprintRoutesRaw from "../../../productBlueprint/src/presentation/routes/routes";
const productBlueprintRoutes = productBlueprintRoutesRaw as unknown as RouteObject[];

import productionRoutesRaw from "../../../production/src/presentation/routes/routes";
const productionRoutes = productionRoutesRaw as unknown as RouteObject[];

import inventoryRoutesRaw from "../../../inventory/src/presentation/routes/routes";
const inventoryRoutes = inventoryRoutesRaw as unknown as RouteObject[];

import tokenBlueprintRoutesRaw from "../../../tokenBlueprint/src/presentation/routes/routes";
const tokenBlueprintRoutes = tokenBlueprintRoutesRaw as unknown as RouteObject[];

import mintRequestRoutesRaw from "../../../mintRequest/src/presentation/routes/routes";
const mintRequestRoutes = mintRequestRoutesRaw as unknown as RouteObject[];

import operationRoutesRaw from "../../../operation/src/presentation/routes/routes";
const operationRoutes = operationRoutesRaw as unknown as RouteObject[];

import listRoutesRaw from "../../../list/src/presentation/routes/routes";
const listRoutes = listRoutesRaw as unknown as RouteObject[];

import orderRoutesRaw from "../../../order/src/routes";
const orderRoutes = orderRoutesRaw as unknown as RouteObject[];

import adRtoutesRaw from "../../../ad/src/presentation/routes/routes";
const adRoutes = adRtoutesRaw as unknown as RouteObject[];

import memberRoutesRaw from "../../../member/src/presentation/routes/routes";
const memberRoutes = memberRoutesRaw as unknown as RouteObject[];

import brandRoutesRaw from "../../../brand/src/presentation/routes/routes";
const brandRoutes = brandRoutesRaw as unknown as RouteObject[];

import permissionRoutesRaw from "../../../permission/src/routes";
const permissionRoutes = permissionRoutesRaw as unknown as RouteObject[];

import accountRoutesRaw from "../../../account/src/routes";
const accountRoutes = accountRoutesRaw as unknown as RouteObject[];

import transactionRoutesRaw from "../../../transaction/src/routes";
const transactionRoutes = transactionRoutesRaw as unknown as RouteObject[];
/**
 * Shell全体で使用するルーティング定義
 * - Layout (Main.tsx) からインポートされる
 * - 各モジュールの routes.tsx を children として統合
 */


export const routes: RouteObject[] = [
  // Announcement モジュール
  {
    path: "/announcement",
    children: announcementRoutes
  },
  // Message モジュール
  {
    path: "/message",
    children: messageRoutes,
  },
  // Inquiry モジュール  
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
  // Ads モジュール
  {
    path: "/ad",
    children: adRoutes,
  },
  // Members モジュール
  {
    path: "/member",
    children: memberRoutes,
  },
  // Brands モジュール
  {
    path: "/brand",
    children: brandRoutes,
  },
  // Permissions モジュール
  {
    path: "/permission",
    children: permissionRoutes,
  },
  // Accounts モジュール
  {
    path: "/account",
    children: accountRoutes,
  },
// Transactions モジュール
  {
    path: "/transaction",
    children: transactionRoutes,
  },
];

export default routes;
