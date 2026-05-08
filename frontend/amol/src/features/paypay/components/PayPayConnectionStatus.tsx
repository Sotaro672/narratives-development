//frontend\src\features\paypay\api\checkPayPayConnection.ts
import { usePayPayConnectionCheck } from "../hooks/usePayPayConnectionCheck";

export function PayPayConnectionStatus() {
  const { message, environment, loading } = usePayPayConnectionCheck();

  return (
    <section>
      <h2>PayPay Connection Check</h2>
      <p>{loading ? "Checking..." : message}</p>
      <p>Environment: {environment ?? "not loaded"}</p>
    </section>
  );
}