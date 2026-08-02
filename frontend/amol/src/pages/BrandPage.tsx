// frontend/amol/src/pages/BrandPage.tsx

import {
  useEffect,
  useMemo,
  useState,
} from "react";
import {
  Link,
  useNavigate,
  useParams,
} from "react-router-dom";

import Layout from "../components/layout/Layout";
import MediaIcon from "../components/ui/MediaIcon";
import { formatPrice } from "../features/shared/utils/price";
import { getApiBaseUrl } from "../lib/apiBaseUrl";
import { isRecord } from "../features/shared/utils/typeGuards";

import "../styles/brand_page.css";

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

function textValue(
  value: unknown,
): string {
  if (value == null) {
    return "";
  }

  return String(value).trim();
}

function numberValue(
  value: unknown,
): number | undefined {
  if (
    typeof value === "number" &&
    Number.isFinite(value)
  ) {
    return value;
  }

  if (typeof value === "string") {
    const number = Number(value);

    if (Number.isFinite(number)) {
      return number;
    }
  }

  return undefined;
}

function stringArrayValue(
  value: unknown,
): string[] {
  if (!Array.isArray(value)) {
    return [];
  }

  return value
    .map(textValue)
    .filter(
      (item) => item.length > 0,
    );
}

function unwrapData(
  value: unknown,
): Record<string, unknown> {
  if (
    !isRecord(value) ||
    Array.isArray(value)
  ) {
    throw new Error(
      "invalid response shape",
    );
  }

  const data = value.data;

  if (
    isRecord(data) &&
    !Array.isArray(data)
  ) {
    return unwrapData(data);
  }

  return value;
}

function unwrapListItem(
  value: unknown,
): Record<string, unknown> {
  const root = unwrapData(value);

  if (
    isRecord(root.item) &&
    !Array.isArray(root.item)
  ) {
    return root.item;
  }

  if (
    isRecord(root.list) &&
    !Array.isArray(root.list)
  ) {
    return root.list;
  }

  return root;
}

function priceRowsValue(
  value: unknown,
): ListPriceRow[] {
  if (!Array.isArray(value)) {
    return [];
  }

  return value
    .filter(
      (
        row,
      ): row is Record<string, unknown> =>
        isRecord(row) &&
        !Array.isArray(row),
    )
    .map((row) => ({
      currency: textValue(
        row.currency,
      ),
      amount: numberValue(
        row.amount,
      ),
      price: numberValue(
        row.price,
      ),
      ...row,
    }));
}

function brandDetailFromJson(
  raw: unknown,
): BrandDetailDTO {
  const json = unwrapData(raw);

  return {
    brandId: textValue(
      json.brandId,
    ),
    brandName: textValue(
      json.brandName,
    ),
    websiteUrl: textValue(
      json.websiteUrl ||
        json.url,
    ),
    brandIcon: textValue(
      json.brandIcon,
    ),
    brandBackgroundImage:
      textValue(
        json.brandBackgroundImage,
      ),
    description: textValue(
      json.description,
    ),
    companyId: textValue(
      json.companyId,
    ),
    companyName: textValue(
      json.companyName,
    ),
    inventoryIds:
      stringArrayValue(
        json.inventoryIds,
      ),
    listIds: stringArrayValue(
      json.listIds,
    ),
  };
}

function mallListItemFromJson(
  raw: unknown,
  fallbackId: string,
): MallListItem {
  const json = unwrapListItem(raw);

  return {
    id:
      textValue(json.id) ||
      fallbackId,
    title: textValue(
      json.title,
    ),
    description: textValue(
      json.description,
    ),
    image: textValue(
      json.image ||
        json.imageUrl ||
        json.thumbnailUrl,
    ),
    prices: priceRowsValue(
      json.prices,
    ),
    inventoryId:
      textValue(
        json.inventoryId,
      ) || undefined,
    productBlueprintId:
      textValue(
        json.productBlueprintId,
      ) || undefined,
    tokenBlueprintId:
      textValue(
        json.tokenBlueprintId,
      ) || undefined,
  };
}

function formatListPrice(
  prices: ListPriceRow[],
): string {
  const firstPrice =
    Array.isArray(prices)
      ? prices[0]
      : undefined;

  const amount =
    firstPrice?.amount ??
    firstPrice?.price;

  return formatPrice(amount, {
    currency:
      firstPrice?.currency,
  });
}

async function fetchBrandById(
  brandId: string,
): Promise<BrandDetailDTO> {
  const id = brandId.trim();

  if (!id) {
    throw new Error(
      "brandId is empty",
    );
  }

  const base = getApiBaseUrl();

  if (!base) {
    throw new Error(
      "VITE_API_BASE_URL is not configured",
    );
  }

  const url =
    `${base}/mall/brands/${encodeURIComponent(
      id,
    )}`;

  const response = await fetch(
    url,
    {
      method: "GET",
      headers: {
        Accept: "application/json",
      },
    },
  );

  const text =
    await response.text();

  if (!response.ok) {
    const body =
      text.length > 300
        ? text.slice(0, 300)
        : text;

    throw new Error(
      `failed to load brand: ${response.status} body=${body}`,
    );
  }

  let decoded: unknown;

  try {
    decoded = text
      ? JSON.parse(text)
      : {};
  } catch {
    throw new Error(
      "failed to load brand: invalid json",
    );
  }

  return brandDetailFromJson(
    decoded,
  );
}

async function fetchMallListItemById(
  listId: string,
): Promise<MallListItem> {
  const id = listId.trim();

  if (!id) {
    throw new Error(
      "listId is empty",
    );
  }

  const base = getApiBaseUrl();

  if (!base) {
    throw new Error(
      "VITE_API_BASE_URL is not configured",
    );
  }

  const url =
    `${base}/mall/lists/${encodeURIComponent(
      id,
    )}`;

  const response = await fetch(
    url,
    {
      method: "GET",
      headers: {
        Accept: "application/json",
      },
      credentials: "include",
    },
  );

  const text =
    await response.text();

  if (!response.ok) {
    const body =
      text.length > 300
        ? text.slice(0, 300)
        : text;

    throw new Error(
      `failed to load list: ${response.status} body=${body}`,
    );
  }

  let decoded: unknown;

  try {
    decoded = text
      ? JSON.parse(text)
      : {};
  } catch {
    throw new Error(
      "failed to load list: invalid json",
    );
  }

  return mallListItemFromJson(
    decoded,
    id,
  );
}

async function fetchMallListItemsByIds(
  listIds: string[],
): Promise<MallListItem[]> {
  const uniqueIds =
    Array.from(
      new Set(
        listIds
          .map((id) => id.trim())
          .filter(Boolean),
      ),
    );

  const results =
    await Promise.allSettled(
      uniqueIds.map((id) =>
        fetchMallListItemById(id),
      ),
    );

  return results
    .filter(
      (
        result,
      ): result is PromiseFulfilledResult<MallListItem> =>
        result.status ===
        "fulfilled",
    )
    .map(
      (result) => result.value,
    )
    .filter(
      (item) =>
        item.id.trim().length >
        0,
    );
}

function buildInitial(
  name: string,
): string {
  const normalizedName =
    name.trim();

  if (!normalizedName) {
    return "B";
  }

  return normalizedName
    .slice(0, 1)
    .toUpperCase();
}

function BrandIcon({
  brand,
}: {
  brand: BrandDetailDTO;
}) {
  return (
    <MediaIcon
      src={brand.brandIcon}
      alt={
        brand.brandName ||
        "ブランドアイコン"
      }
      fallback={buildInitial(
        brand.brandName,
      )}
      size="lg"
      shape="circle"
      className="brand-page-icon"
    />
  );
}

function BrandBackground({
  brand,
}: {
  brand: BrandDetailDTO;
}) {
  const [failed, setFailed] =
    useState(false);

  if (
    !brand.brandBackgroundImage ||
    failed
  ) {
    return null;
  }

  return (
    <div className="brand-page-hero">
      <img
        className="brand-page-hero-image"
        src={
          brand.brandBackgroundImage
        }
        alt={`${
          brand.brandName ||
          "ブランド"
        }の背景画像`}
        loading="lazy"
        onError={() =>
          setFailed(true)
        }
      />
    </div>
  );
}

function ExternalWebsiteLink({
  url: sourceUrl,
}: {
  url: string;
}) {
  const url = sourceUrl.trim();

  if (!url) {
    return null;
  }

  const href =
    url.startsWith("http://") ||
    url.startsWith("https://")
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

function ListItemCards({
  listIds,
  listItems,
}: {
  listIds: string[];
  listItems: MallListItem[];
}) {
  if (listIds.length === 0) {
    return (
      <section className="brand-page-section">
        <h2>
          出品中のリスト
        </h2>

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
          <h2>
            出品中のリスト
          </h2>

          <span>
            {listIds.length}件
          </span>
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
        <h2>
          出品中のリスト
        </h2>

        <span>
          {listItems.length}件
        </span>
      </div>

      <div className="lists-page-grid brand-page-list-grid">
        {listItems.map((item) => {
          const title =
            item.title ||
            item.id;

          return (
            <Link
              key={item.id}
              className="lists-page-card brand-page-list-card"
              to={`/lists/${encodeURIComponent(
                item.id,
              )}`}
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
                <h2 className="lists-page-card-title">
                  {title}
                </h2>

                {item.description ? (
                  <p className="lists-page-card-description">
                    {
                      item.description
                    }
                  </p>
                ) : null}

                <div className="lists-page-card-footer">
                  <span className="lists-page-card-price">
                    {formatListPrice(
                      item.prices,
                    )}
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

function BrandContent({
  brand,
  listItems,
}: {
  brand: BrandDetailDTO;
  listItems: MallListItem[];
}) {
  const hasDescription =
    brand.description.trim()
      .length > 0;

  const hasCompanyName =
    brand.companyName.trim()
      .length > 0;

  const hasWebsite =
    brand.websiteUrl.trim()
      .length > 0;

  return (
    <div className="brand-page">
      <BrandBackground
        brand={brand}
      />

      <section className="brand-page-profile">
        <BrandIcon
          brand={brand}
        />

        <div className="brand-page-profile-body">
          <h1>
            {brand.brandName ||
              "名称未設定のブランド"}
          </h1>

          {hasCompanyName ? (
            <p className="brand-page-company">
              {
                brand.companyName
              }
            </p>
          ) : null}

          {hasWebsite ? (
            <ExternalWebsiteLink
              url={
                brand.websiteUrl
              }
            />
          ) : null}
        </div>
      </section>

      {hasDescription ? (
        <section className="brand-page-section">
          <h2>説明</h2>

          <p className="brand-page-description">
            {brand.description}
          </p>
        </section>
      ) : null}

      <ListItemCards
        listIds={brand.listIds}
        listItems={listItems}
      />
    </div>
  );
}

export default function BrandPage() {
  const params = useParams();
  const navigate = useNavigate();

  const brandId = useMemo(() => {
    return String(
      params.brandId || "",
    ).trim();
  }, [params.brandId]);

  const [state, setState] =
    useState<BrandPageState>({
      status: "idle",
      brand: null,
      listItems: [],
      error: "",
    });

  function handleHeaderBackButtonClick() {
    navigate(-1);
  }

  useEffect(() => {
    let cancelled = false;

    async function run() {
      if (!brandId) {
        setState({
          status: "error",
          brand: null,
          listItems: [],
          error:
            "brandId is empty",
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
        const brand =
          await fetchBrandById(
            brandId,
          );

        const listItems =
          await fetchMallListItemsByIds(
            brand.listIds,
          );

        if (cancelled) {
          return;
        }

        setState({
          status: "success",
          brand,
          listItems,
          error: "",
        });
      } catch (error) {
        if (cancelled) {
          return;
        }

        setState({
          status: "error",
          brand: null,
          listItems: [],
          error:
            error instanceof Error
              ? error.message
              : "failed to load brand",
        });
      }
    }

    void run();

    return () => {
      cancelled = true;
    };
  }, [brandId]);

  if (
    state.status === "loading" ||
    state.status === "idle"
  ) {
    return (
      <Layout
        title="ブランド"
        titleClickable={false}
        mode="landing"
        showHeader
        showBackButton
        backTo="/lists"
        onBackButtonClick={
          handleHeaderBackButtonClick
        }
        showFooter={false}
        hideHamburgerMenu={false}
        hideSettingsButton
        mainClassName="brand-page-main"
      >
        <div className="brand-page brand-page-centered">
          <div className="brand-page-loading">
            ブランド情報を読み込み中...
          </div>
        </div>
      </Layout>
    );
  }

  if (state.status === "error") {
    return (
      <Layout
        title="ブランド"
        titleClickable={false}
        mode="landing"
        showHeader
        showBackButton
        backTo="/lists"
        onBackButtonClick={
          handleHeaderBackButtonClick
        }
        showFooter={false}
        hideHamburgerMenu={false}
        hideSettingsButton
        mainClassName="brand-page-main"
      >
        <div className="brand-page brand-page-centered">
          <div className="brand-page-error-card">
            <h1>
              ブランド情報を取得できませんでした
            </h1>

            <p>{state.error}</p>

            <div className="brand-page-error-actions">
              <button
                type="button"
                onClick={
                  handleHeaderBackButtonClick
                }
              >
                戻る
              </button>

              <Link to="/">
                トップへ
              </Link>
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
        titleClickable={false}
        mode="landing"
        showHeader
        showBackButton
        backTo="/lists"
        onBackButtonClick={
          handleHeaderBackButtonClick
        }
        showFooter={false}
        hideHamburgerMenu={false}
        hideSettingsButton
        mainClassName="brand-page-main"
      >
        <div className="brand-page brand-page-centered">
          <div className="brand-page-error-card">
            <h1>
              ブランド情報を取得できませんでした
            </h1>

            <p>
              brand data is empty
            </p>

            <div className="brand-page-error-actions">
              <button
                type="button"
                onClick={
                  handleHeaderBackButtonClick
                }
              >
                戻る
              </button>

              <Link to="/">
                トップへ
              </Link>
            </div>
          </div>
        </div>
      </Layout>
    );
  }

  return (
    <Layout
      title={
        state.brand.brandName ||
        "ブランド"
      }
      titleClickable={false}
      mode="landing"
      showHeader
      showBackButton
      backTo="/lists"
      onBackButtonClick={
        handleHeaderBackButtonClick
      }
      showFooter={false}
      hideHamburgerMenu={false}
      hideSettingsButton
      mainClassName="brand-page-main"
    >
      <BrandContent
        brand={state.brand}
        listItems={
          state.listItems
        }
      />
    </Layout>
  );
}