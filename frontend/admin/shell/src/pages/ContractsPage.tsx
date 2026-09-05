// frontend/admin/shell/src/pages/ContractsPage.tsx

import { useEffect, useMemo, useState } from "react";
import { getAuthHeaders } from "../shared/http/authHeaders";
import type { Company, CompanyListResponse } from "../shared/type/company";
import Page from "../shared/ui/Page/Page";
import Table, { type TableColumn } from "../shared/ui/Table/Table";
import { formatDateTime } from "../shared/util/dateFormat";

const BACKEND_BASE_URL = import.meta.env.VITE_BACKEND_BASE_URL?.trim().replace(/\/+$/, "");

function requireBackendBaseUrl(): string {
  if (!BACKEND_BASE_URL) {
    throw new Error("VITE_BACKEND_BASE_URL is not configured.");
  }
  return BACKEND_BASE_URL;
}

async function listCompanies(): Promise<Company[]> {
  const backendBaseUrl = requireBackendBaseUrl();
  const authHeaders = await getAuthHeaders();

  const response = await fetch(`${backendBaseUrl}/admin/companies`, {
    method: "GET",
    headers: {
      ...authHeaders,
      Accept: "application/json",
    },
  });

  if (!response.ok) {
    let detail = "";

    try {
      const body = (await response.json()) as { error?: string };
      detail = body.error ? ` error=${body.error}` : "";
    } catch {
      // Response body may not be JSON.
    }

    throw new Error(`Failed to load companies. status=${response.status}${detail}`);
  }

  const body = (await response.json()) as CompanyListResponse;
  return Array.isArray(body.items) ? body.items : [];
}

export default function ContractsPage() {
  const [companies, setCompanies] = useState<Company[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    const loadCompanies = async () => {
      try {
        setLoading(true);
        setError(null);

        const result = await listCompanies();

        if (!cancelled) {
          setCompanies(result);
        }
      } catch (cause) {
        if (!cancelled) {
          setError(
            cause instanceof Error
              ? cause.message
              : "企業一覧の取得に失敗しました。",
          );
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    };

    void loadCompanies();

    return () => {
      cancelled = true;
    };
  }, []);

  const columns = useMemo<TableColumn<Company>[]>(
    () => [
      {
        key: "createdAt",
        header: "登録日時",
        render: (company) => formatDateTime(company.createdAt),
        sortValue: (company) => new Date(company.createdAt).getTime(),
        nowrap: true,
      },
      {
        key: "name",
        header: "企業名",
        render: (company) => company.name,
        sortValue: (company) => company.name,
        filter: {
          getValue: (company) => company.name,
          placeholder: "企業名で絞り込み",
        },
        nowrap: true,
      },
      {
        key: "representativeName",
        header: "代表者",
        render: (company) => company.representativeName || "-",
        sortValue: (company) => company.representativeName,
        filter: {
          getValue: (company) => company.representativeName,
          placeholder: "代表者名で絞り込み",
        },
        nowrap: true,
      },
      {
        key: "isActive",
        header: "契約状態",
        render: (company) => company.isActive ? "契約中" : "停止中",
        sortValue: (company) => company.isActive,
        filter: {
          getValue: (company) => company.isActive ? "契約中" : "停止中",
          options: [
            { value: "契約中", label: "契約中" },
            { value: "停止中", label: "停止中" },
          ],
        },
        nowrap: true,
      },
      {
        key: "updatedAt",
        header: "最終更新日時",
        render: (company) => formatDateTime(company.updatedAt),
        sortValue: (company) => new Date(company.updatedAt).getTime(),
        nowrap: true,
      },
    ],
    [],
  );

  return (
    <Page>
      <h1>契約</h1>

      {loading && <p>企業一覧を読み込んでいます。</p>}

      {!loading && error && (
        <p role="alert">
          企業一覧の取得に失敗しました。{error}
        </p>
      )}

      {!loading && !error && (
        <Table
          columns={columns}
          rows={companies}
          getRowKey={(company) => company.id}
          emptyMessage="登録企業はありません。"
          filteredEmptyMessage="条件に一致する企業はありません。"
        />
      )}
    </Page>
  );
}