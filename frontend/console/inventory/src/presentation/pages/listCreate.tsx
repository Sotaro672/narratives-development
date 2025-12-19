// frontend/console/inventory/src/presentation/pages/listCreate.tsx

import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";
import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";

import { Card, CardContent } from "../../../../shell/src/shared/ui/card";
import { Button } from "../../../../shell/src/shared/ui/button";

import {
  Popover,
  PopoverTrigger,
  PopoverContent,
} from "../../../../shell/src/shared/ui/popover";

// Firebase Auth（IDトークン）
import { auth } from "../../../../shell/src/auth/infrastructure/config/firebaseClient";

// ★ Admin 用 hook（担当者候補の取得・選択）
import { useAdminCard as useAdminCardHook } from "../../../../admin/src/presentation/hook/useAdminCard";

function s(v: unknown): string {
  return String(v ?? "").trim();
}

type ListingDecision = "list" | "hold";

/**
 * backend/internal/application/query/dto/list_create_dto.go と対応
 */
type ListCreateDTO = {
  inventoryId?: string;
  productBlueprintId?: string;
  tokenBlueprintId?: string;

  productBrandName: string;
  productName: string;

  tokenBrandName: string;
  tokenName: string;
};

async function fetchListCreateDTO(input: {
  inventoryId?: string;
  productBlueprintId?: string;
  tokenBlueprintId?: string;
}): Promise<ListCreateDTO> {
  const token = await auth.currentUser?.getIdToken();
  if (!token) throw new Error("unauthenticated");

  // ✅ 想定エンドポイント（Inventory ドメイン配下に寄せる）
  // - /inventory/list-create/:inventoryId
  // - /inventory/list-create/:productBlueprintId/:tokenBlueprintId
  //
  // ※ backend 側の handler ルーティングに合わせて調整してください。
  let path = "";
  if (input.inventoryId) {
    path = `/inventory/list-create/${encodeURIComponent(input.inventoryId)}`;
  } else if (input.productBlueprintId && input.tokenBlueprintId) {
    path =
      `/inventory/list-create/${encodeURIComponent(
        input.productBlueprintId,
      )}/${encodeURIComponent(input.tokenBlueprintId)}`;
  } else {
    throw new Error("missing params");
  }

  const ENV_BASE =
    (
      (import.meta as any).env?.VITE_BACKEND_BASE_URL as string | undefined
    )?.replace(/\/+$/g, "") ?? "";
  const FALLBACK_BASE =
    "https://narratives-backend-871263659099.asia-northeast1.run.app";
  const API_BASE = ENV_BASE || FALLBACK_BASE;

  const res = await fetch(`${API_BASE}${path}`, {
    method: "GET",
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });

  if (!res.ok) {
    const text = await res.text().catch(() => "");
    throw new Error(`request failed: ${res.status} ${text}`);
  }

  return (await res.json()) as ListCreateDTO;
}

export default function InventoryListCreate() {
  const navigate = useNavigate();

  // ✅ routes.tsx で定義した param を受け取る（inventoryId or pb/tb）
  const params = useParams<{
    inventoryId?: string;
    productBlueprintId?: string;
    tokenBlueprintId?: string;
  }>();

  const inventoryId = s(params.inventoryId);
  const productBlueprintId = s(params.productBlueprintId);
  const tokenBlueprintId = s(params.tokenBlueprintId);

  // ✅ PageHeader（title）には pb/tb を出さない
  const title = inventoryId ? `出品作成（inventoryId: ${inventoryId}）` : "出品作成";

  // ✅ 戻るは inventoryDetail へ絶対遷移
  const onBack = React.useCallback(() => {
    if (productBlueprintId && tokenBlueprintId) {
      navigate(`/inventory/detail/${productBlueprintId}/${tokenBlueprintId}`);
      return;
    }
    navigate("/inventory");
  }, [navigate, productBlueprintId, tokenBlueprintId]);

  // ✅ 作成ボタン（PageHeader）
  const onCreate = React.useCallback(() => {
    // TODO: 出品作成APIを呼ぶ（今は仮）
    alert("作成しました（仮）");

    if (productBlueprintId && tokenBlueprintId) {
      navigate(`/inventory/detail/${productBlueprintId}/${tokenBlueprintId}`);
      return;
    }
    navigate("/inventory");
  }, [navigate, productBlueprintId, tokenBlueprintId]);

  // ============================================================
  // ✅ listCreate 用 DTO を取得（pb/tb または inventoryId から）
  // ============================================================
  const [dto, setDTO] = React.useState<ListCreateDTO | null>(null);
  const [loadingDTO, setLoadingDTO] = React.useState(false);
  const [dtoError, setDTOError] = React.useState<string>("");

  React.useEffect(() => {
    let cancelled = false;

    const run = async () => {
      // inventoryId が無い場合は pb/tb が必須
      const canFetch =
        Boolean(inventoryId) ||
        (Boolean(productBlueprintId) && Boolean(tokenBlueprintId));
      if (!canFetch) return;

      setLoadingDTO(true);
      setDTOError("");

      try {
        const data = await fetchListCreateDTO({
          inventoryId: inventoryId || undefined,
          productBlueprintId: productBlueprintId || undefined,
          tokenBlueprintId: tokenBlueprintId || undefined,
        });
        if (!cancelled) setDTO(data);
      } catch (e) {
        if (!cancelled) setDTOError(String(e instanceof Error ? e.message : e));
      } finally {
        if (!cancelled) setLoadingDTO(false);
      }
    };

    void run();
    return () => {
      cancelled = true;
    };
  }, [inventoryId, productBlueprintId, tokenBlueprintId]);

  // ============================================================
  // ✅ 右カラム：担当者選択（ボタンのみ表示）
  // ============================================================
  const { assigneeName, assigneeCandidates, loadingMembers, onSelectAssignee } =
    useAdminCardHook();

  const handleSelectAssignee = React.useCallback(
    (id: string) => {
      onSelectAssignee(id);
    },
    [onSelectAssignee],
  );

  // ============================================================
  // ✅ 出品｜保留 選択
  // ============================================================
  const [decision, setDecision] = React.useState<ListingDecision>("list");

  return (
    <PageStyle
      layout="grid-2"
      title={title}
      onBack={onBack}
      onCreate={onCreate} // ✅ PageHeader に「作成」ボタンを表示
    >
      {/* 左カラム：空（grid-2 のレイアウト維持） */}
      <div />

      {/* 右カラム */}
      <div className="space-y-4">
        {/* DTO 読み込み状態（style elements only） */}
        {loadingDTO && (
          <div className="text-sm text-[hsl(var(--muted-foreground))]">
            読み込み中...
          </div>
        )}
        {dtoError && (
          <div className="text-sm text-red-600">
            読み込みに失敗しました: {dtoError}
          </div>
        )}

        {/* ✅ 担当者（title: 担当者） */}
        <Card>
          <CardContent className="p-4">
            <div className="text-sm font-medium mb-2">担当者</div>

            <Popover>
              <PopoverTrigger>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  className="w-full justify-between"
                >
                  <span>{assigneeName || "未設定"}</span>
                  <span className="text-[11px] text-slate-400" />
                </Button>
              </PopoverTrigger>

              <PopoverContent className="p-2 space-y-1">
                {loadingMembers && (
                  <p className="text-xs text-slate-400">
                    担当者を読み込み中です…
                  </p>
                )}

                {!loadingMembers && assigneeCandidates.length > 0 && (
                  <div className="space-y-1">
                    {assigneeCandidates.map((c) => (
                      <button
                        key={c.id}
                        type="button"
                        className="block w-full text-left px-2 py-1 rounded hover:bg-slate-100 text-sm"
                        onClick={() => handleSelectAssignee(c.id)}
                      >
                        {c.name}
                      </button>
                    ))}
                  </div>
                )}

                {!loadingMembers && assigneeCandidates.length === 0 && (
                  <p className="text-xs text-slate-400">
                    担当者候補がありません。
                  </p>
                )}
              </PopoverContent>
            </Popover>
          </CardContent>
        </Card>

        {/* ✅ 選択商品カード：productName / brandName（DTO） */}
        <Card>
          <CardContent className="p-4">
            <div className="text-sm font-medium mb-2">選択商品</div>
            <div className="text-sm text-slate-800 break-all">
              {s(dto?.productBrandName) || "未選択"}
            </div>
            <div className="text-sm text-slate-800 break-all">
              {s(dto?.productName) || "未選択"}
            </div>
          </CardContent>
        </Card>

        {/* ✅ 選択トークンカード：tokenName / brandName（DTO） */}
        <Card>
          <CardContent className="p-4">
            <div className="text-sm font-medium mb-2">選択トークン</div>
            <div className="text-sm text-slate-800 break-all">
              {s(dto?.tokenBrandName) || "未選択"}
            </div>
            <div className="text-sm text-slate-800 break-all">
              {s(dto?.tokenName) || "未選択"}
            </div>
          </CardContent>
        </Card>

        {/* ✅ 出品｜保留 選択カード */}
        <Card>
          <CardContent className="p-4">
            <div className="text-sm font-medium mb-2">出品｜保留</div>

            <div className="flex gap-2">
              <Button
                type="button"
                variant={decision === "list" ? "default" : "outline"}
                size="sm"
                className="flex-1"
                onClick={() => setDecision("list")}
              >
                出品
              </Button>

              <Button
                type="button"
                variant={decision === "hold" ? "default" : "outline"}
                size="sm"
                className="flex-1"
                onClick={() => setDecision("hold")}
              >
                保留
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>
    </PageStyle>
  );
}
