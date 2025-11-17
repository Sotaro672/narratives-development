//frontend\console\brand\src\presentation\hook\useBrandDetail.ts
import { useCallback, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";

export interface BrandDetailData {
  id: string;
  name: string;
  code: string;
  category: string;
  description: string;
  owner: string;
  status: string;
  registeredAt: string;
  updatedAt: string;
}

export function useBrandDetail() {
  const navigate = useNavigate();
  const { brandId } = useParams<{ brandId: string }>();

  // ─────────────────────────────────────────────
  // モックデータ（API接続前）
  // ─────────────────────────────────────────────
  const [brand] = useState<BrandDetailData>({
    id: brandId ?? "brand_001",
    name: "LUMINA Fashion",
    code: "LUMINA01",
    category: "ファッション",
    description:
      "上質な素材とサステナブルな生産体制を重視した女性向けファッションブランド。",
    owner: "佐藤 美咲",
    status: "アクティブ",
    registeredAt: "2024/05/10",
    updatedAt: "2025/11/01",
  });

  // ─────────────────────────────────────────────
  // AdminCard 用モックデータ
  // ─────────────────────────────────────────────
  const [assignee, setAssignee] = useState("高橋 健太");
  const [creator] = useState("山田 太郎");
  const [createdAt] = useState("2024/05/10");

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
    assignee,
    creator,
    createdAt,
    setAssignee,
    handleBack,
    statusBadgeClass,
  };
}
