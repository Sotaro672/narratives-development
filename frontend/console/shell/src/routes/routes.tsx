// frontend/shell/src/router/routes.tsx
import type { RouteObject } from "react-router-dom";

// モジュールのルート定義（型衝突を避けるため unknown→RouteObject[] にキャスト）
import announcementRoutesRaw from "../../../announcement/presentation/routes/routes";
const announcementRoutes = announcementRoutesRaw as unknown as RouteObject[];

import messageRouteRaw from "../../../message/src/presentation/routes/routes";
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

import listRoutesRaw from "../../../list/src/presentation/routes/routes";
const listRoutes = listRoutesRaw as unknown as RouteObject[];

import orderRoutesRaw from "../../../order/src/presentation/routes/routes";
const orderRoutes = orderRoutesRaw as unknown as RouteObject[];

import memberRoutesRaw from "../../../member/src/presentation/routes/routes";
const memberRoutes = memberRoutesRaw as unknown as RouteObject[];

import brandRoutesRaw from "../../../brand/src/presentation/routes/routes";
const brandRoutes = brandRoutesRaw as unknown as RouteObject[];

import permissionRoutesRaw from "../../../permission/src/presentation/routes/routes";
const permissionRoutes = permissionRoutesRaw as unknown as RouteObject[];

import accountRoutesRaw from "../../../account/src/presentation/routes/routes";
const accountRoutes = accountRoutesRaw as unknown as RouteObject[];

import transactionRoutesRaw from "../../../transaction/src/presentation/routes/routes";
const transactionRoutes = transactionRoutesRaw as unknown as RouteObject[];

import productBlueprintReviewRoutesRaw from "../../../productBlueprintReview/src/presentation/routes/routes";
const productBlueprintReviewRoutes =
  productBlueprintReviewRoutesRaw as unknown as RouteObject[];

import tokenBlueprintReviewRoutesRaw from "../../../tokenBlueprintReview/src/presentation/routes/routes";
const tokenBlueprintReviewRoutes =
  tokenBlueprintReviewRoutesRaw as unknown as RouteObject[];

import salesRoutesRaw from "../../../sales/presentation/routes/routes";
const salesRoutes = salesRoutesRaw as unknown as RouteObject[];

export const routes: RouteObject[] = [
  {
    path: "/announcement",
    children: announcementRoutes,
  },
  {
    path: "/message",
    children: messageRoutes,
  },
  {
    path: "/inquiry",
    children: inquiryRoutes,
  },
  {
    path: "/productBlueprint",
    children: productBlueprintRoutes,
  },
  {
    path: "/production",
    children: productionRoutes,
  },
  {
    path: "/inventory",
    children: inventoryRoutes,
  },
  {
    path: "/tokenBlueprint",
    children: tokenBlueprintRoutes,
  },
  {
    path: "/mintRequest",
    children: mintRequestRoutes,
  },
  {
    path: "/productBlueprintReview",
    children: productBlueprintReviewRoutes,
  },
  {
    path: "/tokenBlueprintReview",
    children: tokenBlueprintReviewRoutes,
  },
  {
    path: "/list",
    children: listRoutes,
  },
  {
    path: "/order",
    children: orderRoutes,
  },
  {
    path: "/member",
    children: memberRoutes,
  },
  {
    path: "/brand",
    children: brandRoutes,
  },
  {
    path: "/permission",
    children: permissionRoutes,
  },
  {
    path: "/account",
    children: accountRoutes,
  },
  {
    path: "/transaction",
    children: transactionRoutes,
  },
  {
    path: "/sales",
    children: salesRoutes,
  },
];

export default routes;