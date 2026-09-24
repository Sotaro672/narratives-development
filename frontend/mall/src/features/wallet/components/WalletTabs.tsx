// frontend/amol/src/features/wallet/components/WalletTabs.tsx

import Tab from "../../../components/ui/Tab";
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
      <Tab
        role="tab"
        variant="underline"
        selected={activeTab === "history"}
        fullWidth
        onClick={() => onChange("history")}
      >
        取引履歴
      </Tab>

      <Tab
        role="tab"
        variant="underline"
        selected={activeTab === "tokens"}
        fullWidth
        onClick={() => onChange("tokens")}
      >
        トークン
      </Tab>

      <Tab
        role="tab"
        variant="underline"
        selected={activeTab === "resales"}
        fullWidth
        onClick={() => onChange("resales")}
      >
        出品
      </Tab>
    </div>
  );
}