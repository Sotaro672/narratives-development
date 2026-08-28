// frontend/amol/src/features/payout/components/PayoutAccountConnectPanel.tsx

import { useCallback, useMemo } from "react";
import { loadConnectAndInitialize } from "@stripe/connect-js";
import {
  ConnectAccountManagement,
  ConnectAccountOnboarding,
  ConnectComponentsProvider,
} from "@stripe/react-connect-js";

import type { PayoutAccountConnectMode } from "../hooks/usePayoutAccountPage";

type PayoutAccountConnectPanelProps = {
  publishableKey: string;
  fetchClientSecret: () => Promise<string>;
  mode: PayoutAccountConnectMode;
  onExit: () => void | Promise<void>;
  onError: (error: unknown) => void;
};

export default function PayoutAccountConnectPanel({
  publishableKey,
  fetchClientSecret,
  mode,
  onExit,
  onError,
}: PayoutAccountConnectPanelProps) {
  const connectInstance = useMemo(() => {
    if (!publishableKey) {
      return null;
    }

    return loadConnectAndInitialize({
      publishableKey,
      fetchClientSecret,
    });
  }, [fetchClientSecret, publishableKey]);

  const handleExit = useCallback(() => {
    void onExit();
  }, [onExit]);

  if (!connectInstance) {
    return (
      <div className="payout-account-page__connect-panel">
        <p className="payout-account-page__connect-message">
          Stripeを初期化しています...
        </p>
      </div>
    );
  }

  return (
    <div className="payout-account-page__connect-panel">
      <div className="payout-account-page__connect-header">
        <button
          type="button"
          className="payout-account-page__connect-close-button"
          onClick={handleExit}
        >
          閉じる
        </button>
      </div>

      <div className="payout-account-page__connect-content">
        <ConnectComponentsProvider connectInstance={connectInstance}>
          {mode === "management" ? (
            <ConnectAccountManagement
              onLoadError={({ error }) => {
                onError(error);
              }}
            />
          ) : (
            <ConnectAccountOnboarding
              onExit={handleExit}
              onLoadError={({ error }) => {
                onError(error);
              }}
            />
          )}
        </ConnectComponentsProvider>
      </div>
    </div>
  );
}