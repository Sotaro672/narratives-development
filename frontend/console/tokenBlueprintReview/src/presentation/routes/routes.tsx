// frontend/console/tokenBlueprintReview/src/presentation/routes/routes.tsx
import type { RouteObject } from "react-router-dom";
import TokenBlueprintReviewManagement from "../../presentation/pages/tokenBlueprintReviewManagement";
import TokenBlueprintReviewDetail from "../../presentation/pages/tokenBlueprintReviewDetail";

const routes: RouteObject[] = [
  { path: "", element: <TokenBlueprintReviewManagement /> },
  { path: ":tokenBlueprintReviewId", element: <TokenBlueprintReviewDetail /> },
];

export default routes;