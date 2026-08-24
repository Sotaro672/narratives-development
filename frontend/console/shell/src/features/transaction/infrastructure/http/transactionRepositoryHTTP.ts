// frontend/console/shell/src/features/transaction/infrastructure/http/transactionRepositoryHTTP.ts

import type { PageParams, PageResult } from "../../../../shared/types/common/common";
import { buildConsoleUrl } from "../../../../shared/http/apiBase";
import { fetchJSON } from "../../../../shared/http/fetchJSON";
import type { TransactionManagementRowDTO } from "../../../../shared/types/transaction";

export type TransactionListParams = PageParams & {
  createdFrom?: string;
  createdTo?: string;
};

const BASE_URL = buildConsoleUrl("/transactions");

function buildQuery(params: TransactionListParams): string {
  const searchParams = new URLSearchParams();

  if (params.page !== undefined) {
    searchParams.set("page", String(params.page));
  }

  if (params.perPage !== undefined) {
    searchParams.set("perPage", String(params.perPage));
  }

  if (params.createdFrom) {
    searchParams.set("createdFrom", params.createdFrom);
  }

  if (params.createdTo) {
    searchParams.set("createdTo", params.createdTo);
  }

  const query = searchParams.toString();

  return query ? `?${query}` : "";
}

export class TransactionRepositoryHTTP {
  private readonly baseUrl: string;

  constructor(baseUrl: string = BASE_URL) {
    this.baseUrl = baseUrl.replace(/\/+$/g, "");

    if (!this.baseUrl) {
      throw new Error("[TransactionRepositoryHTTP] baseUrl is empty.");
    }
  }

  /**
   * 認証中ユーザーの Company に属する入出金履歴を取得します。
   *
   * Backend:
   * GET /transactions
   *
   * CompanyID は認証ContextからBackend側で決定するため、
   * Frontendから companyId は送信しません。
   *
   * Transaction は Settlement を基に生成されます。
   *
   * - receive: Stripe Connect Transfer による入金
   * - send: Stripe Connect Transfer Reversal による出金
   */
  async list(
    params: TransactionListParams = {},
  ): Promise<PageResult<TransactionManagementRowDTO>> {
    const query = buildQuery(params);

    return fetchJSON<PageResult<TransactionManagementRowDTO>>(
      `${this.baseUrl}${query}`,
      {
        method: "GET",
        auth: "required",
      },
    );
  }
}

export const transactionRepositoryHTTP = new TransactionRepositoryHTTP();