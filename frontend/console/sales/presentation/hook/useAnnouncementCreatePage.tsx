// frontend/console/sales/src/presentation/hook/useAnnouncementCreatePage.tsx
import { useCallback, useEffect, useMemo, useState } from "react";
import { useLocation, useNavigate, useParams } from "react-router-dom";

import {
  createEmptyAnnouncementCreateVM,
  fetchAnnouncementCreateVM,
  normalizeAnnouncementCreateLocationState,
  saveAnnouncement,
  sendAnnouncement,
  type AnnouncementCreateInputPayload,
  type AnnouncementCreateVM,
  type AnnouncementOwnerVM,
} from "../../application/announcement_create_service";

export type { AnnouncementCreateInputPayload, AnnouncementOwnerVM };

export type SubmitAnnouncementParams = {
  payload: AnnouncementCreateInputPayload;
  targetAvatarIds: string[];
};

type UseAnnouncementCreatePageHandlers = {
  onBack: () => void;
  onSaveAnnouncement: (params: SubmitAnnouncementParams) => Promise<void>;
  onSendAnnouncement: (params: SubmitAnnouncementParams) => Promise<void>;
};

export type UseAnnouncementCreatePageResult = {
  vm: AnnouncementCreateVM;
  handlers: UseAnnouncementCreatePageHandlers;
};

export function useAnnouncementCreatePage(): UseAnnouncementCreatePageResult {
  const navigate = useNavigate();
  const location = useLocation();
  const { tokenBlueprintId } = useParams<{ tokenBlueprintId: string }>();

  const [vm, setVm] = useState<AnnouncementCreateVM>(() =>
    createEmptyAnnouncementCreateVM(),
  );

  const locationState = useMemo(() => {
    return normalizeAnnouncementCreateLocationState(location.state);
  }, [location.state]);

  useEffect(() => {
    let cancelled = false;

    (async () => {
      try {
        const nextVm = await fetchAnnouncementCreateVM(
          tokenBlueprintId,
          locationState,
        );

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
      await saveAnnouncement({
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
      await sendAnnouncement({
        sales: vm.sales,
        payload,
        createdBy: resolveCreatedBy(),
        targetAvatarIds,
      });
    },
    [resolveCreatedBy, vm.sales],
  );

  const handlers: UseAnnouncementCreatePageHandlers = {
    onBack: handleBack,
    onSaveAnnouncement: handleSaveAnnouncement,
    onSendAnnouncement: handleSendAnnouncement,
  };

  return { vm, handlers };
}

export default useAnnouncementCreatePage;