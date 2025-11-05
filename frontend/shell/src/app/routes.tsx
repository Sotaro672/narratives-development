// frontend/shell/src/app/routes.tsx
import type { RouteObject } from "react-router-dom";

// 各サブアプリのページをインポート
import InquiryManagementPage from "../../../inquiry/src/pages/InquiryManagementPage";
import ProductBlueprintManagementPage from "../../../productBlueprint/src/pages/productBlueprintManagement";
import ProductBlueprintDetailPage from "../../../productBlueprint/src/pages/productBlueprintDetail";
import ProductionManagementPage from "../../../production/src/pages/productionManagement";
import InventoryManagementPage from "../../../inventory/src/pages/inventoryManagement";
import TokenBlueprintManagementPage from "../../../tokenBlueprint/src/pages/tokenBlueprintManagement";
import MintRequestManagementPage from "../../../mintRequest/src/pages/mintRequestManagement";
import TokenOperationPage from "../../../operation/src/pages/tokenOperation";
import ListManagementPage from "../../../list/src/pages/listManagement";
import OrderManagementPage from "../../../order/src/pages/orderManagement";
import MemberManagementPage from "../../../member/src/pages/memberManagement";
import BrandManagementPage from "../../../brand/src/pages/brandManagement";
import PermissionListPage from "../../../permission/src/pages/permissionList";
import AdManagementPage from "../../../ad/src/pages/adManagement";
import AccountManagementPage from "../../../account/src/pages/accountManagement";
import TransactionListPage from "../../../transaction/src/pages/transactionList";

/**
 * Shell全体で使用するルーティング定義
 * - Layout (Main.tsx) からインポートされる
 */
export const routes: RouteObject[] = [
  { path: "/inquiry", element: <InquiryManagementPage /> },
  { path: "/productBlueprint", element: <ProductBlueprintManagementPage /> },
  { path: "/productBlueprint/detail", element: <ProductBlueprintDetailPage /> },
  { path: "/production", element: <ProductionManagementPage /> },
  { path: "/inventory", element: <InventoryManagementPage /> },
  { path: "/tokenBlueprint", element: <TokenBlueprintManagementPage /> },
  { path: "/mintRequest", element: <MintRequestManagementPage /> },
  { path: "/operation", element: <TokenOperationPage /> },
  { path: "/list", element: <ListManagementPage /> },
  { path: "/order", element: <OrderManagementPage /> },
  { path: "/member", element: <MemberManagementPage /> },
  { path: "/brand", element: <BrandManagementPage /> },
  { path: "/permission", element: <PermissionListPage /> },
  { path: "/ad", element: <AdManagementPage /> },
  { path: "/account", element: <AccountManagementPage /> },
  { path: "/transaction", element: <TransactionListPage /> },
];

export default routes;
