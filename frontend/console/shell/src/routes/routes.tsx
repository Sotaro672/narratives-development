// frontend/console/shell/src/routes/routes.tsx

import type { RouteObject } from "react-router-dom";

import {
  AuthPage,
  InvitationPage,
  InquiryManagement,
  InquiryDetail,
  ProductBlueprintManagement,
  ProductBlueprintDetail,
  ProductBlueprintCreate,
  ProductionManagement,
  ProductionDetail,
  ProductionCreate,
  InventoryManagementPage,
  InventoryDetailPage,
  InventoryListCreatePage,
  TokenBlueprintManagement,
  TokenBlueprintDetail,
  TokenBlueprintCreate,
  MintManagement,
  MintDetail,
  ProductBlueprintReviewManagement,
  ProductBlueprintReviewDetail,
  TokenBlueprintReviewManagement,
  TokenBlueprintReviewDetail,
  ListManagement,
  ListDetail,
  OrderManagement,
  OrderDetail,
  MemberManagement,
  MemberDetail,
  MemberCreate,
  BrandManagement,
  BrandCreate,
  BrandDetail,
  CompanyDetail,
  PermissionList,
  PermissionDetail,
  AccountManagement,
  TransactionsList,
  TransactionDetail,
  AnnouncementManagementPage,
  AnnouncementCreatePage,
  AnnouncementTokenListPage,
  AnnouncementDetailPage,
} from "../pages";

export const routes: RouteObject[] = [
  {
    path: "/auth",
    element: <AuthPage />,
  },
  {
    path: "/invitation",
    element: <InvitationPage />,
  },
  {
    path: "/company",
    element: <CompanyDetail />,
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
    path: "/mint",
    children: [
      {
        path: "",
        element: <MintManagement />,
      },
      {
        path: ":requestId",
        element: <MintDetail />,
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