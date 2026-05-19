// frontend/amol/src/pages/BrandPage.tsx
import { useEffect, useMemo, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import "../styles/brand_page.css";
import Layout from "../components/layout/Layout";
import MediaIcon from "../components/ui/MediaIcon";

type ListPriceRow = {
  currency?: string;
  amount?: number;
  price?: number;
  [key: string]: unknown;
};

type MallListItem = {
  id: string;
  title: string;
  description: string;
  image: string;
  prices: ListPriceRow[];

  inventoryId?: string;
  productBlueprintId?: string;
  tokenBlueprintId?: string;
};

type BrandDetailDTO = {
  brandId: string;
  brandName: string;
  websiteUrl: string;
  brandIcon: string;
  brandBackgroundImage: string;
  description: string;
  companyId: string;
  companyName: string;
  inventoryIds: string[];
  listIds: string[];
};

type BrandPageState =
  | {
      status: "idle" | "loading";
      brand: null;
      listItems: MallListItem[];
      error: "";
    }
  | {
      status: "success";
      brand: BrandDetailDTO;
      listItems: MallListItem[];
      error: "";
    }
  | {
      status: "error";
      brand: null;
      listItems: MallListItem[];
      error: string;
    };

function normalizeBaseUrl(value: string): string {
  let v = value.trim();
  while (v.endsWith("/")) {
    v = v.slice(0, -1);
  }
  return v;
}

function resolveApiBase(): string {
  return normalizeBaseUrl(String(import.meta.env.VITE_API_BASE_URL || ""));
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === "object" && !Array.isArray(value);
}

function textValue(value: unknown): string {
  if (value == null) return "";
  return String(value).trim();
}

function numberValue(value: unknown): number | undefined {
  if (typeof value === "number" && Number.isFinite(value)) {
    return value;
  }

  if (typeof value === "string") {
    const n = Number(value);
    if (Number.isFinite(n)) {
      return n;
    }
  }

  return undefined;
}

function stringArrayValue(value: unknown): string[] {
  if (!Array.isArray(value)) return [];

  return value.map(textValue).filter((v) => v.length > 0);
}

function unwrapData(value: unknown): Record<string, unknown> {
  if (!isRecord(value)) {
    throw new Error("invalid response shape");
  }

  const data = value.data;
  if (isRecord(data)) {
    return unwrapData(data);
  }

  return value;
}

function unwrapListItem(value: unknown): Record<string, unknown> {
  const root = unwrapData(value);

  if (isRecord(root.item)) {
    return root.item;
  }

  if (isRecord(root.list)) {
    return root.list;
  }

  return root;
}

function priceRowsValue(value: unknown): ListPriceRow[] {
  if (!Array.isArray(value)) {
    return [];
  }

  return value
    .filter(isRecord)
    .map((row) => ({
      currency: textValue(row.currency),
      amount: numberValue(row.amount),
      price: numberValue(row.price),
      ...row,
    }));
}

function brandDetailFromJson(raw: unknown): BrandDetailDTO {
  const j = unwrapData(raw);

  return {
    brandId: textValue(j.brandId),
    brandName: textValue(j.brandName),
    websiteUrl: textValue(j.websiteUrl || j.url),
    brandIcon: textValue(j.brandIcon),
    brandBackgroundImage: textValue(j.brandBackgroundImage),
    description: textValue(j.description),
    companyId: textValue(j.companyId),
    companyName: textValue(j.companyName),
    inventoryIds: stringArrayValue(j.inventoryIds),
    listIds: stringArrayValue(j.listIds),
  };
}

function mallListItemFromJson(raw: unknown, fallbackId: string): MallListItem {
  const j = unwrapListItem(raw);

  return {
    id: textValue(j.id) || fallbackId,
    title: textValue(j.title),
    description: textValue(j.description),
    image: textValue(j.image || j.imageUrl || j.thumbnailUrl),
    prices: priceRowsValue(j.prices),
    inventoryId: textValue(j.inventoryId) || undefined,
    productBlueprintId: textValue(j.productBlueprintId) || undefined,
    tokenBlueprintId: textValue(j.tokenBlueprintId) || undefined,
  };
}

function formatPrice(prices: ListPriceRow[]): string {
  if (!Array.isArray(prices) || prices.length === 0) {
    return "価格未設定";
  }

  const first = prices[0];
  const rawAmount = first.amount ?? first.price;
  const amount =
    typeof rawAmount === "number"
      ? rawAmount
      : typeof rawAmount === "string"
        ? Number(rawAmount)
        : NaN;

  const currency =
    typeof first.currency === "string" && first.currency.trim() !== ""
      ? first.currency.toUpperCase()
      : "JPY";

  if (!Number.isFinite(amount)) {
    return "価格未設定";
  }

  if (currency === "JPY") {
    return `${amount.toLocaleString("ja-JP")}円`;
  }

  return `${amount.toLocaleString("ja-JP")} ${currency}`;
}

async function fetchBrandById(brandId: string): Promise<BrandDetailDTO> {
  const id = brandId.trim();
  if (!id) {
    throw new Error("brandId is empty");
  }

  const base = resolveApiBase();
  if (!base) {
    throw new Error("VITE_API_BASE_URL is not configured");
  }

  const url = `${base}/mall/brands/${encodeURIComponent(id)}`;
  const response = await fetch(url, {
    method: "GET",
    headers: {
      Accept: "application/json",
    },
  });

  const text = await response.text();

  if (!response.ok) {
    const body = text.length > 300 ? text.slice(0, 300) : text;
    throw new Error(`failed to load brand: ${response.status} body=${body}`);
  }

  let decoded: unknown;
  try {
    decoded = text ? JSON.parse(text) : {};
  } catch {
    throw new Error("failed to load brand: invalid json");
  }

  return brandDetailFromJson(decoded);
}

async function fetchMallListItemById(listId: string): Promise<MallListItem> {
  const id = listId.trim();
  if (!id) {
    throw new Error("listId is empty");
  }

  const base = resolveApiBase();
  if (!base) {
    throw new Error("VITE_API_BASE_URL is not configured");
  }

  const url = `${base}/mall/lists/${encodeURIComponent(id)}`;
  const response = await fetch(url, {
    method: "GET",
    headers: {
      Accept: "application/json",
    },
    credentials: "include",
  });

  const text = await response.text();

  if (!response.ok) {
    const body = text.length > 300 ? text.slice(0, 300) : text;
    throw new Error(`failed to load list: ${response.status} body=${body}`);
  }

  let decoded: unknown;
  try {
    decoded = text ? JSON.parse(text) : {};
  } catch {
    throw new Error("failed to load list: invalid json");
  }

  return mallListItemFromJson(decoded, id);
}

async function fetchMallListItemsByIds(
  listIds: string[],
): Promise<MallListItem[]> {
  const uniqueIds = Array.from(
    new Set(listIds.map((id) => id.trim()).filter(Boolean)),
  );

  const results = await Promise.allSettled(
    uniqueIds.map((id) => fetchMallListItemById(id)),
  );

  return results
    .filter(
      (result): result is PromiseFulfilledResult<MallListItem> =>
        result.status === "fulfilled",
    )
    .map((result) => result.value)
    .filter((item) => item.id.trim().length > 0);
}

function buildInitial(name: string): string {
  const n = name.trim();
  if (!n) return "B";
  return n.slice(0, 1).toUpperCase();
}

function BrandIcon(props: { brand: BrandDetailDTO }) {
  const { brand } = props;

  return (
    <MediaIcon
      src={brand.brandIcon}
      alt={brand.brandName || "ブランドアイコン"}
      fallback={buildInitial(brand.brandName)}
      size="lg"
      shape="circle"
      className="brand-page-icon"
    />
  );
}

function BrandBackground(props: { brand: BrandDetailDTO }) {
  const { brand } = props;
  const [failed, setFailed] = useState(false);

  if (!brand.brandBackgroundImage || failed) {
    return null;
  }

  return (
    <div className="brand-page-hero">
      <img
        className="brand-page-hero-image"
        src={brand.brandBackgroundImage}
        alt={`${brand.brandName || "ブランド"}の背景画像`}
        loading="lazy"
        onError={() => setFailed(true)}
      />
    </div>
  );
}

function ExternalWebsiteLink(props: { url: string }) {
  const url = props.url.trim();
  if (!url) return null;

  const href =
    url.startsWith("http://") || url.startsWith("https://")
      ? url
      : `https://${url}`;

  return (
    <a
      className="brand-page-link"
      href={href}
      target="_blank"
      rel="noreferrer"
    >
      公式サイトを見る
    </a>
  );
}

function ListItemCards(props: {
  listIds: string[];
  listItems: MallListItem[];
}) {
  const { listIds, listItems } = props;

  if (listIds.length === 0) {
    return (
      <section className="brand-page-section">
        <h2>出品中のリスト</h2>
        <div className="brand-page-empty">
          現在このブランドの出品中リストはありません。
        </div>
      </section>
    );
  }

  if (listItems.length === 0) {
    return (
      <section className="brand-page-section">
        <div className="brand-page-section-header">
          <h2>出品中のリスト</h2>
          <span>{listIds.length}件</span>
        </div>

        <div className="brand-page-empty">
          リスト情報を取得できませんでした。
        </div>
      </section>
    );
  }

  return (
    <section className="brand-page-section">
      <div className="brand-page-section-header">
        <h2>出品中のリスト</h2>
        <span>{listItems.length}件</span>
      </div>

      <div className="lists-page-grid brand-page-list-grid">
        {listItems.map((item) => {
          const title = item.title || item.id;

          return (
            <Link
              key={item.id}
              className="lists-page-card brand-page-list-card"
              to={`/lists/${encodeURIComponent(item.id)}`}
            >
              <div className="lists-page-card-image-wrap">
                {item.image ? (
                  <img
                    src={item.image}
                    alt={title}
                    className="lists-page-card-image"
                    loading="lazy"
                  />
                ) : (
                  <div className="lists-page-card-image-placeholder">
                    No Image
                  </div>
                )}
              </div>

              <div className="lists-page-card-body">
                <h2 className="lists-page-card-title">{title}</h2>

                {item.description ? (
                  <p className="lists-page-card-description">
                    {item.description}
                  </p>
                ) : null}

                <div className="lists-page-card-footer">
                  <span className="lists-page-card-price">
                    {formatPrice(item.prices)}
                  </span>
                </div>
              </div>
            </Link>
          );
        })}
      </div>
    </section>
  );
}

function BrandContent(props: {
  brand: BrandDetailDTO;
  listItems: MallListItem[];
}) {
  const { brand, listItems } = props;

  const hasDescription = brand.description.trim().length > 0;
  const hasCompanyName = brand.companyName.trim().length > 0;
  const hasWebsite = brand.websiteUrl.trim().length > 0;

  return (
    <div className="brand-page">
      <BrandBackground brand={brand} />

      <section className="brand-page-profile">
        <BrandIcon brand={brand} />

        <div className="brand-page-profile-body">
          <p className="brand-page-label">Brand</p>
          <h1>{brand.brandName || "名称未設定のブランド"}</h1>

          {hasCompanyName ? (
            <p className="brand-page-company">{brand.companyName}</p>
          ) : null}

          {hasWebsite ? <ExternalWebsiteLink url={brand.websiteUrl} /> : null}
        </div>
      </section>

      {hasDescription ? (
        <section className="brand-page-section">
          <h2>説明</h2>
          <p className="brand-page-description">{brand.description}</p>
        </section>
      ) : null}

      <ListItemCards listIds={brand.listIds} listItems={listItems} />
    </div>
  );
}

export default function BrandPage() {
  const params = useParams();
  const navigate = useNavigate();

  const brandId = useMemo(() => {
    return String(params.brandId || "").trim();
  }, [params.brandId]);

  const [state, setState] = useState<BrandPageState>({
    status: "idle",
    brand: null,
    listItems: [],
    error: "",
  });

  useEffect(() => {
    let cancelled = false;

    async function run() {
      if (!brandId) {
        setState({
          status: "error",
          brand: null,
          listItems: [],
          error: "brandId is empty",
        });
        return;
      }

      setState({
        status: "loading",
        brand: null,
        listItems: [],
        error: "",
      });

      try {
        const brand = await fetchBrandById(brandId);
        const listItems = await fetchMallListItemsByIds(brand.listIds);

        if (cancelled) return;

        setState({
          status: "success",
          brand,
          listItems,
          error: "",
        });
      } catch (error) {
        if (cancelled) return;

        setState({
          status: "error",
          brand: null,
          listItems: [],
          error:
            error instanceof Error ? error.message : "failed to load brand",
        });
      }
    }

    void run();

    return () => {
      cancelled = true;
    };
  }, [brandId]);

  if (state.status === "loading" || state.status === "idle") {
    return (
      <Layout
        title="ブランド"
        mode="landing"
        showHeader
        showBackButton
        backTo="/"
        showFooter={false}
        hideHamburgerMenu={false}
        hideSettingsButton
        mainClassName="brand-page-main"
      >
        <div className="brand-page brand-page-centered">
          <div className="brand-page-loading">ブランド情報を読み込み中...</div>
        </div>
      </Layout>
    );
  }

  if (state.status === "error") {
    return (
      <Layout
        title="ブランド"
        mode="landing"
        showHeader
        showBackButton
        backTo="/"
        showFooter={false}
        hideHamburgerMenu={false}
        hideSettingsButton
        mainClassName="brand-page-main"
      >
        <div className="brand-page brand-page-centered">
          <div className="brand-page-error-card">
            <h1>ブランド情報を取得できませんでした</h1>
            <p>{state.error}</p>
            <div className="brand-page-error-actions">
              <button type="button" onClick={() => navigate(-1)}>
                戻る
              </button>
              <Link to="/">トップへ</Link>
            </div>
          </div>
        </div>
      </Layout>
    );
  }

  if (!state.brand) {
    return (
      <Layout
        title="ブランド"
        mode="landing"
        showHeader
        showBackButton
        backTo="/"
        showFooter={false}
        hideHamburgerMenu={false}
        hideSettingsButton
        mainClassName="brand-page-main"
      >
        <div className="brand-page brand-page-centered">
          <div className="brand-page-error-card">
            <h1>ブランド情報を取得できませんでした</h1>
            <p>brand data is empty</p>
            <div className="brand-page-error-actions">
              <button type="button" onClick={() => navigate(-1)}>
                戻る
              </button>
              <Link to="/">トップへ</Link>
            </div>
          </div>
        </div>
      </Layout>
    );
  }

  return (
    <Layout
      title={state.brand.brandName || "ブランド"}
      mode="landing"
      showHeader
      showBackButton
      backTo="/"
      showFooter={false}
      hideHamburgerMenu={false}
      hideSettingsButton
      mainClassName="brand-page-main"
    >
      <BrandContent brand={state.brand} listItems={state.listItems} />
    </Layout>
  );
}