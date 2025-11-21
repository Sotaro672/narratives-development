// frontend/console/brand/src/presentation/hook/useBrandDetail.ts
import { useCallback, useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { brandRepositoryHTTP } from "../../infrastructure/http/brandRepositoryHTTP";

// ★ member 用のフックから ID → 「姓 名」を解決する関数を借りる
import { useMemberList } from "../../../../member/src/presentation/hooks/useMemberList";

export interface BrandDetailData {
  // ====== Brand スキーマ由来のフィールド ======
  id: string;
  companyId: string;

  name: string;
  description: string;

  /** WebサイトURL。空文字 or undefined の場合は未設定扱い */
  websiteUrl?: string;

  /** 有効フラグ（status は持たない） */
  isActive: boolean;

  /** ブランド責任者 Member ID（任意） */
  managerId: string;

  /** ブロックチェーン上のウォレットアドレス（必須） */
  walletAddress: string;

  /** 作成日時（ISO8601） */
  createdAt: string;
  /** 作成者（任意） */
  createdBy?: string | null;

  /** 更新日時（任意, ISO8601） */
  updatedAtRaw?: string | null;
  /** 更新者（任意） */
  updatedBy?: string | null;

  /** 論理削除日時（任意, ISO8601） */
  deletedAt?: string | null;
  /** 論理削除者（任意） */
  deletedBy?: string | null;

  // ====== 画面用に追加した派生フィールド ======
  managerName?: string; // 取得した責任者名（姓 名）

  /** 画面表示用のステータス文字列（"アクティブ" / "停止"） */
  status: string;

  /** 表示用にフォーマットした登録日（createdAt） */
  registeredAt: string;

  /** 表示用にフォーマットした最終更新日（updatedAtRaw） */
  updatedAt: string;
}

// ISO8601 → YYYY/MM/DD
const formatDateYmd = (iso?: string | null): string => {
  if (!iso) return "";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${y}/${m}/${day}`;
};

export function useBrandDetail() {
  const navigate = useNavigate();
  const { brandId } = useParams<{ brandId: string }>();

  // ★ ここで member 用フックを呼び、ID→表示名変換関数だけ使う
  const { getNameLastFirstByID } = useMemberList();

  const [brand, setBrand] = useState<BrandDetailData>(() => ({
    id: brandId ?? "",
    companyId: "",

    name: "",
    description: "",
    websiteUrl: "",

    isActive: false,
    managerId: "",
    walletAddress: "",

    createdAt: "",
    createdBy: null,
    updatedAtRaw: null,
    updatedBy: null,
    deletedAt: null,
    deletedBy: null,

    managerName: "",
    status: "",
    registeredAt: "",
    updatedAt: "",
  }));
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  // ブランド本体の取得＋責任者名の解決
  useEffect(() => {
    const load = async () => {
      if (!brandId) return;
      try {
        setLoading(true);
        setError(null);

        // backend: GET /brands/:id
        const data: any = await brandRepositoryHTTP.getById(brandId);

        const isActive = !!data.isActive;
        // backend 側のフィールド名ゆらぎに対応（manager / managerId）
        const managerId = String(data.manager ?? data.managerId ?? "").trim();

        // まずブランドの基本情報＋スキーマ全体をセット
        setBrand((prev) => ({
          ...prev,
          id: String(data.id ?? "").trim(),
          companyId: String(data.companyId ?? "").trim(),

          name: String(data.name ?? "").trim(),
          description: String(data.description ?? "").trim(),
          websiteUrl: data.websiteUrl ?? "",

          isActive,
          managerId,
          walletAddress: String(data.walletAddress ?? "").trim(),

          createdAt: String(data.createdAt ?? ""),
          createdBy: data.createdBy ?? null,
          updatedAtRaw: data.updatedAt ?? null,
          updatedBy: data.updatedBy ?? null,
          deletedAt: data.deletedAt ?? null,
          deletedBy: data.deletedBy ?? null,

          // 画面表示用の派生フィールド
          status: isActive ? "アクティブ" : "停止",
          registeredAt: formatDateYmd(data.createdAt),
          updatedAt: formatDateYmd(data.updatedAt),
        }));

        // つづいて managerId → 「姓 名」に解決（useMemberList 経由）
        if (managerId) {
          try {
            const dispName = await getNameLastFirstByID(managerId);
            // デバッグ用ログ
            // eslint-disable-next-line no-console
            console.log(
              "[useBrandDetail] resolved manager name:",
              managerId,
              "→",
              dispName,
            );

            setBrand((prev) => ({
              ...prev,
              managerName: dispName || prev.managerName || "",
            }));
          } catch (e) {
            // eslint-disable-next-line no-console
            console.error("[useBrandDetail] resolve managerName error:", e);
          }
        }
      } catch (e: any) {
        const err = e instanceof Error ? e : new Error(String(e));
        // eslint-disable-next-line no-console
        console.error("[useBrandDetail] load error:", err);
        setError(err);
      } finally {
        setLoading(false);
      }
    };

    void load();
  }, [brandId, getNameLastFirstByID]);

  // 戻るボタン処理
  const handleBack = useCallback(() => {
    navigate(-1);
  }, [navigate]);

  // ステータスの色分け
  const statusBadgeClass =
    brand.status === "アクティブ"
      ? "inline-flex items-center px-2 py-1 rounded-full bg-emerald-50 text-emerald-700 text-xs font-semibold"
      : "inline-flex items-center px-2 py-1 rounded-full bg-slate-50 text-slate-500 text-xs font-semibold";

  return {
    brand,
    handleBack,
    statusBadgeClass,
    loading,
    error,
  };
}
