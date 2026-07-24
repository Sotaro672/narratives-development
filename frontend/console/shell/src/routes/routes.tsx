// frontend/console/shell/src/routes/routes.tsx
import type { RouteObject } from "react-router-dom";

import InvitationPage from "../auth/presentation/pages/InvitationPage";

import InquiryManagement from "../../../inquiry/presentation/pages/inquiryManagement";
import InquiryDetail from "../../../inquiry/presentation/pages/inquiryDetail";

import ProductBlueprintManagement from "../../../productBlueprint/src/presentation/pages/productBlueprintManagement";
import ProductBlueprintDetail from "../../../productBlueprint/src/presentation/pages/productBlueprintDetail";
import ProductBlueprintCreate from "../../../productBlueprint/src/presentation/pages/productBlueprintCreate";

import ProductionManagement from "../../../production/src/presentation/pages/productionManagement";
import ProductionDetail from "../../../production/src/presentation/pages/productionDetail";
import ProductionCreate from "../../../production/src/presentation/pages/productionCreate";

import InventoryManagementPage from "../../../inventory/src/presentation/pages/inventoryManagement";
import InventoryDetailPage from "../../../inventory/src/presentation/pages/inventoryDetail";
import InventoryListCreatePage from "../../../inventory/src/presentation/pages/listCreate";

import TokenBlueprintManagement from "../../../tokenBlueprint/src/presentation/pages/tokenBlueprintManagement";
import TokenBlueprintDetail from "../../../tokenBlueprint/src/presentation/pages/tokenBlueprintDetail";
import TokenBlueprintCreate from "../../../tokenBlueprint/src/presentation/pages/tokenBlueprintCreate";

import MintRequestManagement from "../../../mintRequest/src/presentation/pages/mintRequestManagement";
import MintRequestDetail from "../../../mintRequest/src/presentation/pages/mintRequestDetail";

import ProductBlueprintReviewManagement from "../../../productBlueprintReview/src/presentation/pages/productBlueprintReviewManagement";
import ProductBlueprintReviewDetail from "../../../productBlueprintReview/src/presentation/pages/productBlueprintReviewDetail";

import TokenBlueprintReviewManagement from "../../../tokenBlueprintReview/src/presentation/pages/tokenBlueprintReviewManagement";
import TokenBlueprintReviewDetail from "../../../tokenBlueprintReview/src/presentation/pages/tokenBlueprintReviewDetail";

import ListManagement from "../../../list/presentation/pages/listManagement";
import ListDetail from "../../../list/presentation/pages/listDetail";

import OrderManagement from "../../../order/src/presentation/pages/orderManagement";
import OrderDetail from "../../../order/src/presentation/pages/orderDetail";

import MemberManagement from "../../../member/src/presentation/pages/memberManagement";
import MemberDetail from "../../../member/src/presentation/pages/memberDetail";
import MemberCreate from "../../../member/src/presentation/pages/memberCreate";

import BrandManagement from "../../../brand/src/presentation/pages/brandManagement";
import BrandCreate from "../../../brand/src/presentation/pages/brandCreate";
import BrandDetail from "../../../brand/src/presentation/pages/brandDetail";

import PermissionList from "../../../permission/src/presentation/pages/permissionList";
import PermissionDetail from "../../../permission/src/presentation/pages/permissionDetail";

import AccountManagement from "../../../account/presentation/pages/accountManagement";

import TransactionsList from "../../../transaction/src/presentation/pages/transactionList";
import TransactionDetail from "../../../transaction/src/presentation/pages/transactionDetail";

import AnnouncementManagementPage from "../../../sales/presentation/pages/announcementManagement";
import AnnouncementCreatePage from "../../../sales/presentation/pages/announcementCreatePage";
import AnnouncementTokenListPage from "../../../sales/presentation/pages/announcementTokenListPage";
import AnnouncementDetailPage from "../../../sales/presentation/pages/announcementDetailPage";

export const routes: RouteObject[] = [
  {
    path: "/invitation",
    element: <InvitationPage />,
  },
  {
    path: "/inquiry",
    children: [
      {
        path: "",
        element: <InquiryManagement />,
      },
      {
        path: ":inquiryId",
        element: <InquiryDetail />,
      },
    ],
  },
  {
    path: "/productBlueprint",
    children: [
      {
        path: "",
        element: <ProductBlueprintManagement />,
      },
      {
        path: "detail/:blueprintId",
        element: <ProductBlueprintDetail />,
      },
      {
        path: "create",
        element: <ProductBlueprintCreate />,
      },
    ],
  },
  {
    path: "/production",
    children: [
      {
        path: "",
        element: <ProductionManagement />,
      },
      {
        path: ":productionId",
        element: <ProductionDetail />,
      },
      {
        path: "create",
        element: <ProductionCreate />,
      },
    ],
  },
  {
    path: "/inventory",
    children: [
      {
        path: "",
        element: <InventoryManagementPage />,
      },
      {
        path: "detail/:inventoryId",
        element: <InventoryDetailPage />,
      },
      {
        path: "list/create/:inventoryId",
        element: <InventoryListCreatePage />,
      },
    ],
  },
  {
    path: "/tokenBlueprint",
    children: [
      {
        path: "",
        element: <TokenBlueprintManagement />,
      },
      {
        path: ":tokenBlueprintId",
        element: <TokenBlueprintDetail />,
      },
      {
        path: "create",
        element: <TokenBlueprintCreate />,
      },
    ],
  },
  {
    path: "/mintRequest",
    children: [
      {
        path: "",
        element: <MintRequestManagement />,
      },
      {
        path: ":requestId",
        element: <MintRequestDetail />,
      },
    ],
  },
  {
    path: "/productBlueprintReview",
    children: [
      {
        path: "",
        element: <ProductBlueprintReviewManagement />,
      },
      {
        path: ":productBlueprintReviewId",
        element: <ProductBlueprintReviewDetail />,
      },
    ],
  },
  {
    path: "/tokenBlueprintReview",
    children: [
      {
        path: "",
        element: <TokenBlueprintReviewManagement />,
      },
      {
        path: ":tokenBlueprintReviewId",
        element: <TokenBlueprintReviewDetail />,
      },
    ],
  },
  {
    path: "/list",
    children: [
      {
        path: "",
        element: <ListManagement />,
      },
      {
        path: ":listId",
        element: <ListDetail />,
      },
    ],
  },
  {
    path: "/order",
    children: [
      {
        path: "",
        element: <OrderManagement />,
      },
      {
        path: ":orderId",
        element: <OrderDetail />,
      },
    ],
  },
  {
    path: "/member",
    children: [
      {
        path: "",
        element: <MemberManagement />,
      },
      {
        /**
         * このURLパラメータはFirestore member docIdではなく、
         * Firebase Auth UID。
         *
         * backend:
         * - GET /members/{uid} はFirebase UID専用
         * - PATCH /members/{docId} はFirestore member docId専用
         */
        path: ":memberUid",
        element: <MemberDetail />,
      },
      {
        path: "create",
        element: <MemberCreate />,
      },
    ],
  },
  {
    path: "/brand",
    children: [
      {
        path: "",
        element: <BrandManagement />,
      },
      {
        path: "create",
        element: <BrandCreate />,
      },
      {
        path: ":brandId",
        element: <BrandDetail />,
      },
    ],
  },
  {
    path: "/permission",
    children: [
      {
        path: "",
        element: <PermissionList />,
      },
      {
        path: ":permissionId",
        element: <PermissionDetail />,
      },
    ],
  },
  {
    path: "/account",
    children: [
      {
        path: "",
        element: <AccountManagement />,
      },
    ],
  },
  {
    path: "/transaction",
    children: [
      {
        path: "",
        element: <TransactionsList />,
      },
      {
        path: ":transactionId",
        element: <TransactionDetail />,
      },
    ],
  },
  {
    path: "/sales",
    children: [
      {
        path: "",
        element: <AnnouncementManagementPage />,
      },
      {
        path: "create",
        element: <AnnouncementTokenListPage />,
      },
      {
        path: ":tokenBlueprintId/create",
        element: <AnnouncementCreatePage />,
      },
      {
        path: "announcements/:announcementId",
        element: <AnnouncementDetailPage />,
      },
    ],
  },
];

export default routes;