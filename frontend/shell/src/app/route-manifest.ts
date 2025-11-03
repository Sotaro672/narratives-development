import React, { lazy } from "react";

/**
 * Remote route entry type
 */
export type RemoteRouteEntry = {
  /** ルーティングのベースパス（例: "/org/*"） */
  path: string;

  /** モジュールフェデレーションで公開された routes を読み込むファクトリ */
  loader: () => Promise<{ default: React.ComponentType<any> }>;

  /** サイドバーや権限制御などのメタ情報 */
  meta: {
    /** サイドバーの表示名 */
    label: string;
    /** グループ（セクション）名：サイドバー見出しのまとまり */
    section:
      | "組織"
      | "問い合わせ"
      | "出品・運用・プレビュー"
      | "生産・設計"
      | "ミント・注文"
      | "広告・財務";
    /** 機能キー（ABテスト/Feature Flag用） */
    featureKey?: string;
    /** アイコン名（任意のIconセットに合わせてマッピング） */
    icon?: string;
    /** サイドバーに表示するか */
    showInSidebar?: boolean;
    /** 必要ロールや最小ロールレベルなど（必要なら拡張） */
    requiredRoleLevel?: number;
  };
};

/**
 * 遅延ロード用の薄いヘルパー
 * 例）const OrgModule = toLazy(REMOTE_ROUTES[0].loader)
 */
export const toLazy = (loader: RemoteRouteEntry["loader"]) =>
  lazy(loader);

/**
 * Remote routes manifest
 * 画像（Console画面一覧）に基づく全モジュールのベースルートを定義
 */
export const REMOTE_ROUTES: RemoteRouteEntry[] = [
  // ────────────── 組織系 ──────────────
  {
    path: "/org/*",
    loader: () => import("org/routes"),
    meta: {
      label: "組織",
      section: "組織",
      icon: "Building",
      showInSidebar: true,
      featureKey: "org",
    },
  },

  // ────────────── 問い合わせ ──────────────
  {
    path: "/inquiries/*",
    loader: () => import("inquiries/routes"),
    meta: {
      label: "問い合わせ",
      section: "問い合わせ",
      icon: "MessageSquare",
      showInSidebar: true,
      featureKey: "inquiries",
    },
  },

  // ────────────── 出品・運用・プレビュー ──────────────
  {
    path: "/listings/*",
    loader: () => import("listings/routes"),
    meta: {
      label: "出品",
      section: "出品・運用・プレビュー",
      icon: "PackageSearch",
      showInSidebar: true,
      featureKey: "listings",
    },
  },
  {
    path: "/operations/*",
    loader: () => import("operations/routes"),
    meta: {
      label: "運用",
      section: "出品・運用・プレビュー",
      icon: "Wrench",
      showInSidebar: true,
      featureKey: "operations",
    },
  },
  {
    path: "/preview/*",
    loader: () => import("preview/routes"),
    meta: {
      label: "プレビュー",
      section: "出品・運用・プレビュー",
      icon: "Eye",
      showInSidebar: true,
      featureKey: "preview",
    },
  },

  // ────────────── 生産・設計 ──────────────
  {
    path: "/production/*",
    loader: () => import("production/routes"),
    meta: {
      label: "生産計画",
      section: "生産・設計",
      icon: "CalendarRange",
      showInSidebar: true,
      featureKey: "production",
    },
  },
  {
    path: "/design/*",
    loader: () => import("design/routes"),
    meta: {
      label: "設計",
      section: "生産・設計",
      icon: "PenTool",
      showInSidebar: true,
      featureKey: "design",
    },
  },

  // ────────────── ミント・注文 ──────────────
  {
    path: "/mint/*",
    loader: () => import("mint/routes"),
    meta: {
      label: "ミント申請",
      section: "ミント・注文",
      icon: "Sparkles",
      showInSidebar: true,
      featureKey: "mint",
    },
  },
  {
    path: "/orders/*",
    loader: () => import("orders/routes"),
    meta: {
      label: "注文",
      section: "ミント・注文",
      icon: "ShoppingCart",
      showInSidebar: true,
      featureKey: "orders",
    },
  },

  // ────────────── 広告・財務 ──────────────
  {
    path: "/ads/*",
    loader: () => import("ads/routes"),
    meta: {
      label: "広告",
      section: "広告・財務",
      icon: "Megaphone",
      showInSidebar: true,
      featureKey: "ads",
    },
  },
  {
    path: "/accounts/*",
    loader: () => import("accounts/routes"),
    meta: {
      label: "口座",
      section: "広告・財務",
      icon: "Banknote",
      showInSidebar: true,
      featureKey: "accounts",
    },
  },
  {
    path: "/transactions/*",
    loader: () => import("transactions/routes"),
    meta: {
      label: "取引履歴",
      section: "広告・財務",
      icon: "Receipt",
      showInSidebar: true,
      featureKey: "transactions",
    },
  },
];

/**
 * サイドバー用の整形済みデータ（任意）
 * 例：セクションごとにグループ化して表示
 */
export const SIDEBAR_SECTIONS = (() => {
  const grouped = new Map<string, RemoteRouteEntry[]>();
  for (const r of REMOTE_ROUTES) {
    if (!r.meta.showInSidebar) continue;
    const key = r.meta.section;
    const arr = grouped.get(key) ?? [];
    arr.push(r);
    grouped.set(key, arr);
  }
  return grouped;
})();

/**
 * ルーターで使うための遅延コンポーネントを生成
 * 例：
 *   const routes = toLazyRoutes();
 *   <Route path="/org/*" element={<routes.org />} />
 */
export const toLazyRoutes = () => {
  const map: Record<string, React.LazyExoticComponent<React.ComponentType<any>>> = {};
  for (const r of REMOTE_ROUTES) {
    map[r.path] = toLazy(r.loader);
  }
  return map;
};
