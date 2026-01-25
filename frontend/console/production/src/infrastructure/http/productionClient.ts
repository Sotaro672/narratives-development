//frontend\console\production\src\infrastructure\http\productionClient.ts
import { BACKEND_API_BASE } from "./apiBase";

/**
 * Production 更新（HTTP）
 * - throwOnError: true がデフォルト
 * - notify系など「握りたい」場合は swallowError を使う
 */
export async function updateProduction(params: {
  productionId: string;
  token: string;
  payload: any;

  swallowError?: boolean;

  logContext?: {
    tag?: string;
    productionId?: string;
  };
}): Promise<void> {
  const { productionId, token, payload, swallowError, logContext } = params;

  const safeId = encodeURIComponent(productionId);

  try {
    const res = await fetch(`${BACKEND_API_BASE}/productions/${safeId}`, {
      method: "PUT",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify(payload),
    });

    if (!res.ok) {
      const body = await res.text().catch(() => "");
      const err = new Error(
        `Production update failed: ${res.status} ${res.statusText}${body ? ` - ${body}` : ""}`,
      );

      if (swallowError) {
        console.error(logContext?.tag ?? "[updateProduction] failed", {
          productionId,
          status: res.status,
          statusText: res.statusText,
          body,
        });
        return;
      }

      throw err;
    }
  } catch (e) {
    if (swallowError) {
      console.error(logContext?.tag ?? "[updateProduction] unexpected error", {
        productionId,
        error: e,
      });
      return;
    }
    throw e;
  }
}
