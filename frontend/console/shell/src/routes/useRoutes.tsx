// frontend/shell/src/routes/useRoutes.tsx
import type { ReactElement } from "react";
import { useRoutes as useReactRouterRoutes } from "react-router-dom";

import { routes } from "./routes";

/**
 * Shell 全体のルート定義（router/routes.tsx）を
 * react-router の useRoutes で解決するラッパ。
 *
 * - Main.tsx などからは react-router-dom を直接意識せず、
 *   このフックだけを呼び出せば OK。
 */
export default function useRoutes(): ReactElement | null {
  const element = useReactRouterRoutes(routes);
  return element;
}
