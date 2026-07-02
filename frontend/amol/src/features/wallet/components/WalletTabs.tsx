// frontend/amol/src/features/wallet/components/WalletTabs.tsx
import type { WalletTabKey } from "../types";

type WalletTabsProps = {
  activeTab: WalletTabKey;
  onChange: (tab: WalletTabKey) => void;
};

export default function WalletTabs({ activeTab, onChange }: WalletTabsProps) {
  return (
    <div
      className="wallet-page-tabs"
      role="tablist"
      aria-label="ウォレット表示切替"
    >
      <button
        type="button"
        role="tab"
        aria-selected={activeTab === "history"}
        className={[
          "wallet-page-tabs__button",
          activeTab === "history" ? "wallet-page-tabs__button--active" : "",
        ]
          .filter(Boolean)
          .join(" ")}
        onClick={() => onChange("history")}
      >
        取引履歴
      </button>

      <button
        type="button"
        role="tab"
        aria-selected={activeTab === "tokens"}
        className={[
          "wallet-page-tabs__button",
          activeTab === "tokens" ? "wallet-page-tabs__button--active" : "",
        ]
          .filter(Boolean)
          .join(" ")}
        onClick={() => onChange("tokens")}
      >
        トークン
      </button>

      <button
        type="button"
        role="tab"
        aria-selected={activeTab === "resales"}
        className={[
          "wallet-page-tabs__button",
          activeTab === "resales" ? "wallet-page-tabs__button--active" : "",
        ]
          .filter(Boolean)
          .join(" ")}
        onClick={() => onChange("resales")}
      >
        出品
      </button>
    </div>
  );
}