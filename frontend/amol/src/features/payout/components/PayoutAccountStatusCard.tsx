// frontend/amol/src/features/payout/components/PayoutAccountStatusCard.tsx

type PayoutAccountStatusCardProps = {
  statusLabel: string;
  bankName: string;
  bankLast4: string;
};

export default function PayoutAccountStatusCard({
  statusLabel,
  bankName,
  bankLast4,
}: PayoutAccountStatusCardProps) {
  const hasBankAccount = Boolean(bankName || bankLast4);

  return (
    <div className="payout-account-page__status-card">
      <div className="payout-account-page__status-row">
        <span className="payout-account-page__label">
          登録状況
        </span>

        <strong className="payout-account-page__status">
          {statusLabel}
        </strong>
      </div>

      {hasBankAccount ? (
        <div className="payout-account-page__bank-account">
          {bankName ? (
            <div className="payout-account-page__bank-row">
              <span className="payout-account-page__label">
                金融機関
              </span>

              <span className="payout-account-page__bank-value">
                {bankName}
              </span>
            </div>
          ) : null}

          {bankLast4 ? (
            <div className="payout-account-page__bank-row">
              <span className="payout-account-page__label">
                口座番号
              </span>

              <span className="payout-account-page__bank-value">
                •••• {bankLast4}
              </span>
            </div>
          ) : null}
        </div>
      ) : null}
    </div>
  );
}