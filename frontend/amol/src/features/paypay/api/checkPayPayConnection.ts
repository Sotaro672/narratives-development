//frontend\src\features\paypay\api\checkPayPayConnection.ts
import type { PayPayCheckResponse } from "../types/paypay";

export async function checkPayPayConnection(): Promise<PayPayCheckResponse> {
  const backendBaseUrl = import.meta.env.VITE_API_BASE_URL;

  if (!backendBaseUrl) {
    throw new Error("VITE_API_BASE_URL is not set");
  }

  const response = await fetch(`${backendBaseUrl}/api/paypay/check`, {
    method: "GET",
  });

  if (!response.ok) {
    throw new Error(`Backend request failed: ${response.status}`);
  }

  const data: PayPayCheckResponse = await response.json();
  return data;
}