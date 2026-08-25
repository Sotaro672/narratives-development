// frontend/console/shell/src/features/account/infrastructure/http/accountRepositoryHTTP.ts

import type {
  Account,
  AccountInput,
  AccountStatus,
  AccountType,
} from "../../../../shared/types/account";
import { buildConsoleUrl } from "../../../../shared/http/apiBase";
import { fetchJSON } from "../../../../shared/http/fetchJSON";

export type AccountListResponse = {
  items: Account[];
};

export type UpdateAccountInput = {
  stripeAccountId?: string;
  memberId?: string;
  bankName?: string;
  branchName?: string;
  accountNumber?: number;
  accountType?: AccountType | "";
  currency?: string;
  status?: AccountStatus;
};

export type ConnectAccountInput = {
  accountId?: string;
  displayName?: string;
  contactEmail: string;
  country?: string;
  returnUrl: string;
  refreshUrl: string;
};

export type ConnectAccountResponse = {
  account: Account;
  onboardingUrl: string;
  expiresAt: string;
};

const BASE_URL = buildConsoleUrl("/accounts");

export class AccountRepositoryHTTP {
  private readonly baseUrl: string;

  constructor(baseUrl: string = BASE_URL) {
    this.baseUrl = baseUrl.replace(/\/+$/g, "");

    if (!this.baseUrl) {
      throw new Error("[AccountRepositoryHTTP] baseUrl is empty.");
    }
  }

  /**
   * 認証中ユーザーの Company に所属する Account 一覧を取得します。
   *
   * Backend:
   * GET /accounts
   *
   * CompanyID は認証ContextからBackend側で決定するため、
   * Frontendから companyId は送信しません。
   */
  async list(): Promise<Account[]> {
    const response = await fetchJSON<AccountListResponse>(
      this.baseUrl,
      {
        method: "GET",
        auth: "required",
      },
    );

    return Array.isArray(response?.items)
      ? response.items
      : [];
  }

  /**
   * Account ID を指定して1件取得します。
   *
   * Backend:
   * GET /accounts/{id}
   */
  async getById(id: string): Promise<Account> {
    if (!id) {
      throw new Error("[AccountRepositoryHTTP] accountId is required.");
    }

    const url = `${this.baseUrl}/${encodeURIComponent(id)}`;

    return fetchJSON<Account>(
      url,
      {
        method: "GET",
        auth: "required",
      },
    );
  }

  /**
   * Account を作成します。
   *
   * Backend:
   * POST /accounts
   *
   * companyId は認証ContextからBackend側で設定されます。
   */
  async create(input: AccountInput): Promise<Account> {
    return fetchJSON<Account>(
      this.baseUrl,
      {
        method: "POST",
        auth: "required",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          stripeAccountId: input.stripeAccountId,
          memberId: input.memberId,
          bankName: input.bankName,
          branchName: input.branchName,
          accountNumber: input.accountNumber,
          accountType: input.accountType,
          currency: input.currency,
          status: input.status,
        }),
      },
    );
  }

  /**
   * Account を部分更新します。
   *
   * Backend:
   * PATCH /accounts/{id}
   */
  async update(
    id: string,
    patch: UpdateAccountInput,
  ): Promise<Account> {
    if (!id) {
      throw new Error("[AccountRepositoryHTTP] accountId is required.");
    }

    const url = `${this.baseUrl}/${encodeURIComponent(id)}`;

    return fetchJSON<Account>(
      url,
      {
        method: "PATCH",
        auth: "required",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(patch),
      },
    );
  }

  /**
   * Stripe Connected Account を新規作成するか、
   * 既存Accountの onboarding を再開します。
   *
   * accountId:
   * - 未指定: 新しいAccountを作成してStripeへ接続
   * - 指定あり: 既存AccountのStripe onboarding URLを再発行
   *
   * Backend:
   * POST /accounts/connect
   */
  async connect(
    input: ConnectAccountInput,
  ): Promise<ConnectAccountResponse> {
    if (!input.contactEmail) {
      throw new Error("[AccountRepositoryHTTP] contactEmail is required.");
    }

    if (!input.returnUrl) {
      throw new Error("[AccountRepositoryHTTP] returnUrl is required.");
    }

    if (!input.refreshUrl) {
      throw new Error("[AccountRepositoryHTTP] refreshUrl is required.");
    }

    const url = `${this.baseUrl}/connect`;

    return fetchJSON<ConnectAccountResponse>(
      url,
      {
        method: "POST",
        auth: "required",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          accountId: input.accountId ?? "",
          displayName: input.displayName ?? "",
          contactEmail: input.contactEmail,
          country: input.country ?? "JP",
          returnUrl: input.returnUrl,
          refreshUrl: input.refreshUrl,
        }),
      },
    );
  }

  /**
   * Stripe Connected Account の最新状態を取得し、
   * Account.status へ同期します。
   *
   * Backend:
   * POST /accounts/{id}/sync-stripe-status
   *
   * Stripe 側の transfer capability が active の場合は active、
   * pending の場合は inactive、
   * restricted / unsupported / closed の場合は suspended になります。
   */
  async syncStripeStatus(
    id: string,
  ): Promise<Account> {
    if (!id) {
      throw new Error("[AccountRepositoryHTTP] accountId is required.");
    }

    const url =
      `${this.baseUrl}/${encodeURIComponent(id)}/sync-stripe-status`;

    return fetchJSON<Account>(
      url,
      {
        method: "POST",
        auth: "required",
      },
    );
  }
}

export const accountRepositoryHTTP =
  new AccountRepositoryHTTP();

/**
 * Stripe Connected Account が設定されているAccountを取得します。
 *
 * Transaction一覧など、
 * Stripeとの接続済みAccount一覧が必要な画面で使用します。
 */
export async function fetchConnectedAccounts(): Promise<Account[]> {
  const accounts = await accountRepositoryHTTP.list();

  return accounts.filter(
    (account) =>
      account.status !== "deleted" &&
      Boolean(account.stripeAccountId),
  );
}

/**
 * 決済・送金先として利用可能なactive Accountのみ取得します。
 */
export async function fetchActiveAccounts(): Promise<Account[]> {
  const accounts = await accountRepositoryHTTP.list();

  return accounts.filter(
    (account) =>
      account.status === "active" &&
      Boolean(account.stripeAccountId),
  );
}