// frontend/console/inventory/src/presentation/hook/listCreate/useListCreateParamsAndTitle.ts
import * as React from "react";
import { useParams } from "react-router-dom";

import {
  resolveListCreateParams,
  computeListCreateTitle,
  type ListCreateRouteParams,
  type ResolvedListCreateParams,
} from "../../../application/listCreate/listCreateService";

export function useListCreateParamsAndTitle(): {
  resolvedParams: ResolvedListCreateParams;
  inventoryId: string | undefined;
  title: string;
} {
  const params = useParams<ListCreateRouteParams>();

  const resolvedParams: ResolvedListCreateParams = React.useMemo(
    () => resolveListCreateParams(params),
    [params],
  );

  const { inventoryId } = resolvedParams;

  const title = React.useMemo(() => computeListCreateTitle(inventoryId), [inventoryId]);

  return { resolvedParams, inventoryId, title };
}
