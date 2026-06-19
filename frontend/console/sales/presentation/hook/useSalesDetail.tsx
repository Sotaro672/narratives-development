// frontend/console/sales/src/presentation/hook/useSalesDetail.tsx
import { useCallback, useEffect, useMemo, useState } from "react";
import { useLocation, useNavigate, useParams } from "react-router-dom";

import {
  createEmptySalesDetailVM,
  fetchSalesDetailVM,
  normalizeSalesDetailLocationState,
  saveSalesAnnouncement,
  sendSalesAnnouncement,
  type SalesDetailInputPayload,
  type SalesDetailVM,
  type SalesOwnerVM,
} from "../../application/sales_detail_service";

export type { SalesDetailInputPayload, SalesOwnerVM };

export type SubmitAnnouncementParams = {
  payload: SalesDetailInputPayload;
  targetAvatarIds: string[];
};

type UseSalesDetailHandlers = {
  onBack: () => void;
  onSaveAnnouncement: (params: SubmitAnnouncementParams) => Promise<void>;
  onSendAnnouncement: (params: SubmitAnnouncementParams) => Promise<void>;
};

export type UseSalesDetailResult = {
  vm: SalesDetailVM;
  handlers: UseSalesDetailHandlers;
};

export function useSalesDetail(): UseSalesDetailResult {
  const navigate = useNavigate();
  const location = useLocation();
  const { tokenBlueprintId } = useParams<{ tokenBlueprintId: string }>();

  const [vm, setVm] = useState<SalesDetailVM>(() => createEmptySalesDetailVM());

  const locationState = useMemo(() => {
    return normalizeSalesDetailLocationState(location.state);
  }, [location.state]);

  useEffect(() => {
    let cancelled = false;

    (async () => {
      try {
        const nextVm = await fetchSalesDetailVM(tokenBlueprintId, locationState);

        if (cancelled) return;

        setVm(nextVm);
      } catch {
        if (!cancelled) {
          navigate("/sales/create", { replace: true });
        }
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [tokenBlueprintId, locationState, navigate]);

  const handleBack = useCallback(() => {
    navigate("/sales/create", { replace: true });
  }, [navigate]);

  const resolveCreatedBy = useCallback(() => {
    return (
      vm.updatedById ||
      vm.createdById ||
      vm.updatedByName ||
      vm.createdByName ||
      "system"
    );
  }, [vm.createdById, vm.createdByName, vm.updatedById, vm.updatedByName]);

  const handleSaveAnnouncement = useCallback(
    async ({ payload, targetAvatarIds }: SubmitAnnouncementParams) => {
      await saveSalesAnnouncement({
        sales: vm.sales,
        payload,
        createdBy: resolveCreatedBy(),
        targetAvatarIds,
      });
    },
    [resolveCreatedBy, vm.sales],
  );

  const handleSendAnnouncement = useCallback(
    async ({ payload, targetAvatarIds }: SubmitAnnouncementParams) => {
      await sendSalesAnnouncement({
        sales: vm.sales,
        payload,
        createdBy: resolveCreatedBy(),
        targetAvatarIds,
      });
    },
    [resolveCreatedBy, vm.sales],
  );

  const handlers: UseSalesDetailHandlers = {
    onBack: handleBack,
    onSaveAnnouncement: handleSaveAnnouncement,
    onSendAnnouncement: handleSendAnnouncement,
  };

  return { vm, handlers };
}

export default useSalesDetail;