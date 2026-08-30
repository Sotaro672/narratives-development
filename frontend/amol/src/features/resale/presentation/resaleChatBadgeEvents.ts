// frontend/amol/src/features/resale/presentation/resaleChatBadgeEvents.ts

const RESALE_CHAT_BADGE_DELTA_EVENT = "resale-chat:badge-delta";
const RESALE_CHAT_BADGE_REFRESH_EVENT = "resale-chat:badge-refresh";

type ResaleChatBadgeDeltaDetail = {
  delta: number;
};

export function updateResaleChatBadgeCount(delta: number): void {
  window.dispatchEvent(
    new CustomEvent<ResaleChatBadgeDeltaDetail>(
      RESALE_CHAT_BADGE_DELTA_EVENT,
      {
        detail: { delta },
      },
    ),
  );
}

export function refreshResaleChatBadgeCount(): void {
  window.dispatchEvent(
    new Event(RESALE_CHAT_BADGE_REFRESH_EVENT),
  );
}

export function subscribeResaleChatBadgeDelta(
  listener: (delta: number) => void,
): () => void {
  const handleEvent = (event: Event) => {
    const customEvent =
      event as CustomEvent<ResaleChatBadgeDeltaDetail>;

    listener(customEvent.detail.delta);
  };

  window.addEventListener(
    RESALE_CHAT_BADGE_DELTA_EVENT,
    handleEvent,
  );

  return () => {
    window.removeEventListener(
      RESALE_CHAT_BADGE_DELTA_EVENT,
      handleEvent,
    );
  };
}

export function subscribeResaleChatBadgeRefresh(
  listener: () => void,
): () => void {
  window.addEventListener(
    RESALE_CHAT_BADGE_REFRESH_EVENT,
    listener,
  );

  return () => {
    window.removeEventListener(
      RESALE_CHAT_BADGE_REFRESH_EVENT,
      listener,
    );
  };
}