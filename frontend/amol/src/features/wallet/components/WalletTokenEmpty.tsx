// frontend/amol/src/features/wallet/components/WalletTokenEmpty.tsx
export default function WalletTokenEmpty() {
  return (
    <div className="wallet-page-token-empty">
      <div className="wallet-page-token-empty__icon">◎</div>
      <p className="wallet-page-token-empty__title">
        表示できるトークンはまだありません。
      </p>
      <p className="wallet-page-token-empty__text">
        取得したトークンはここに表示されます。
      </p>
    </div>
  );
}