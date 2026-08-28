// frontend/amol/src/features/payout/components/PayoutAccountNotice.tsx

import type { ReactNode } from "react";

type PayoutAccountNoticeProps = {
  children: ReactNode;
};

export default function PayoutAccountNotice({
  children,
}: PayoutAccountNoticeProps) {
  return (
    <div className="payout-account-page__notice">
      <p className="payout-account-page__notice-text">
        {children}
      </p>
    </div>
  );
}