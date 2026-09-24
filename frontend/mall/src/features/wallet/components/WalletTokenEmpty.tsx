// frontend/amol/src/features/wallet/components/WalletTokenEmpty.tsx

import StatePanel from "../../../components/ui/StatePanel";

export default function WalletTokenEmpty() {
  return (
    <StatePanel
      variant="empty"
      icon="◎"
      title="表示できるトークンはまだありません。"
      description="取得したトークンはここに表示されます。"
      className="wallet-page-token-empty"
    />
  );
}