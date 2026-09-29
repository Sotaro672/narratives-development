// frontend/mall/src/pages/WalletStackPage.tsx

import { useOutlet } from "react-router-dom";

import { useMobilePortrait } from "../components/hooks/useMobilePortrait";
import WalletPage from "./WalletPage";

export default function WalletStackPage() {
  const outlet = useOutlet();
  const isMobilePortrait = useMobilePortrait();

  if (!outlet) {
    return <WalletPage />;
  }

  if (!isMobilePortrait) {
    return outlet;
  }

  return (
    <>
      <WalletPage />
      {outlet}
    </>
  );
}