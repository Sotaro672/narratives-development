// frontend/console/inventory/src/presentation/hook/listCreate/useListCreateNavigation.ts
import * as React from "react";
import { useNavigate } from "react-router-dom";

import { buildBackPath, type ResolvedListCreateParams } from "../../../application/listCreate/listCreateService";

export function useListCreateNavigation(resolvedParams: ResolvedListCreateParams): {
  navigate: ReturnType<typeof useNavigate>;
  onBack: () => void;
} {
  const navigate = useNavigate();

  const onBack = React.useCallback(() => {
    navigate(buildBackPath(resolvedParams));
  }, [navigate, resolvedParams]);

  return { navigate, onBack };
}
