// frontend/amol/src/features/inquiry/presentation/inquiryBadgeEvents.ts

const INQUIRY_BADGE_DELTA_EVENT = "inquiry:badge-delta";
const INQUIRY_BADGE_REFRESH_EVENT = "inquiry:badge-refresh";

type InquiryBadgeDeltaDetail = {
  delta: number;
};

export function updateInquiryBadgeCount(delta: number): void {
  window.dispatchEvent(
    new CustomEvent<InquiryBadgeDeltaDetail>(
      INQUIRY_BADGE_DELTA_EVENT,
      {
        detail: { delta },
      },
    ),
  );
}

export function refreshInquiryBadgeCount(): void {
  window.dispatchEvent(
    new Event(INQUIRY_BADGE_REFRESH_EVENT),
  );
}

export function subscribeInquiryBadgeDelta(
  listener: (delta: number) => void,
): () => void {
  const handleEvent = (event: Event) => {
    const customEvent =
      event as CustomEvent<InquiryBadgeDeltaDetail>;

    listener(customEvent.detail.delta);
  };

  window.addEventListener(
    INQUIRY_BADGE_DELTA_EVENT,
    handleEvent,
  );

  return () => {
    window.removeEventListener(
      INQUIRY_BADGE_DELTA_EVENT,
      handleEvent,
    );
  };
}

export function subscribeInquiryBadgeRefresh(
  listener: () => void,
): () => void {
  window.addEventListener(
    INQUIRY_BADGE_REFRESH_EVENT,
    listener,
  );

  return () => {
    window.removeEventListener(
      INQUIRY_BADGE_REFRESH_EVENT,
      listener,
    );
  };
}