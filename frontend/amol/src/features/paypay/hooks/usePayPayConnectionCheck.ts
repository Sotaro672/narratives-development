//frontend\src\features\paypay\hooks\usePayPayConnectionCheck.ts
import { useCallback, useEffect, useState } from "react";
import { checkPayPayConnection } from "../api/checkPayPayConnection";

export function usePayPayConnectionCheck() {
  const [message, setMessage] = useState("Checking PayPay connection...");
  const [environment, setEnvironment] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  const execute = useCallback(async () => {
    try {
      setLoading(true);
      setMessage("Checking PayPay connection...");
      setEnvironment(null);

      const data = await checkPayPayConnection();

      setMessage(data.message);
      setEnvironment(data.environment ?? null);
    } catch (error) {
      console.error(error);

      if (error instanceof Error) {
        setMessage(error.message);
      } else {
        setMessage("Failed to connect to backend or PayPay");
      }

      setEnvironment(null);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void execute();
  }, [execute]);

  return {
    message,
    environment,
    loading,
    reload: execute,
  };
}