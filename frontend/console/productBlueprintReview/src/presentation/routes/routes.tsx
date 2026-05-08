// frontend/console/productBlueprintReview/src/presentation/routes/routes.tsx
import type { RouteObject } from "react-router-dom";
import ProductBlueprintReviewManagement from "../../presentation/pages/productBlueprintReviewManagement";
import ProductBlueprintReviewDetail from "../../presentation/pages/productBlueprintReviewDetail";

const routes: RouteObject[] = [
  { path: "", element: <ProductBlueprintReviewManagement /> },
  { path: ":productBlueprintReviewId", element: <ProductBlueprintReviewDetail /> },
];

export default routes;