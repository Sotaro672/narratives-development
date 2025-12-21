// frontend/list/src/pages/listDetail.tsx

import * as React from "react";
import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";

import { Card, CardContent } from "../../../../shell/src/shared/ui/card";
import { Input } from "../../../../shell/src/shared/ui/input";
import { Button } from "../../../../shell/src/shared/ui/button";

// ✅ PriceCard（list app 側のコンポーネント）
import PriceCard from "../../../../list/src/presentation/components/priceCard";

// ✅ hook
import { useListDetail } from "../../../../list/src/presentation/hook/useListDetail";

// ✅ NEW: PUT /lists/{id} を叩く（hook 側が未実装でもここで直接更新できる）
import { updateListByIdHTTP } from "../../../../../console/list/src/infrastructure/http/listRepositoryHTTP";

function ImageIcon() {
  return (
    <svg
      width="28"
      height="28"
      viewBox="0 0 24 24"
      fill="none"
      className="text-slate-400"
    >
      <path
        d="M21 19V5a2 2 0 0 0-2-2H5a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2Z"
        stroke="currentColor"
        strokeWidth="1.6"
      />
      <path
        d="M8.5 10.5a1.5 1.5 0 1 0 0-3 1.5 1.5 0 0 0 0 3Z"
        stroke="currentColor"
        strokeWidth="1.6"
      />
      <path
        d="M21 16l-5.5-5.5a2 2 0 0 0-2.8 0L5 18"
        stroke="currentColor"
        strokeWidth="1.6"
      />
    </svg>
  );
}

// local trim helper (UI-only)
function s(v: unknown): string {
  return String(v ?? "").trim();
}

type DraftImage = {
  url: string;
  isNew: boolean;
  file?: File;
};

function revokeDraftBlobUrls(items: DraftImage[]) {
  for (const x of items) {
    if (x?.isNew && typeof x?.url === "string" && x.url.startsWith("blob:")) {
      try {
        URL.revokeObjectURL(x.url);
      } catch {
        // noop
      }
    }
  }
}

export default function ListDetail() {
  // ✅ listDetail hook の返却型差分に強くする（UI 側で必要な情報を any で吸収）
  const vm = useListDetail() as any;

  // -----------------------------
  // ViewModel normalization (best-effort)
  // -----------------------------
  const loading = !!vm?.loading;
  const error = s(vm?.error || vm?.dtoError);

  // ✅ NEW: listId を best-effort で抽出（PUT /lists/{id} のため）
  const listId = React.useMemo(() => {
    return s(
      vm?.listId ||
        vm?.id ||
        vm?.dto?.id ||
        vm?.dto?.listId ||
        vm?.dto?.ID ||
        vm?.detail?.id ||
        vm?.detail?.listId ||
        vm?.list?.id ||
        vm?.list?.listId ||
        vm?.resolved?.listId ||
        vm?.resolved?.id,
    );
  }, [vm]);

  const listingTitle =
    s(vm?.listingTitle) || s(vm?.title) || s(vm?.list?.title) || "";
  const description =
    s(vm?.description) ||
    s(vm?.list?.description) ||
    s(vm?.detail?.description) ||
    "";

  const assigneeName =
    s(vm?.assigneeName) || s(vm?.admin?.assigneeName) || "未設定";

  const productBrandName =
    s(vm?.productBrandName) ||
    s(vm?.product?.brandName) ||
    s(vm?.productBlueprint?.brandName);
  const productName =
    s(vm?.productName) ||
    s(vm?.product?.name) ||
    s(vm?.productBlueprint?.productName);

  const tokenBrandName =
    s(vm?.tokenBrandName) ||
    s(vm?.token?.brandName) ||
    s(vm?.tokenBlueprint?.brandName);
  const tokenName =
    s(vm?.tokenName) ||
    s(vm?.token?.name) ||
    s(vm?.tokenBlueprint?.tokenName);

  // decision/status (view)
  const decision =
    s(vm?.decision) || s(vm?.status) || s(vm?.list?.status) || "";

  // price rows (view)
  const basePriceRows = (vm?.priceRows ||
    vm?.prices ||
    vm?.list?.priceRows ||
    vm?.list?.prices ||
    []) as any[];

  // images (view)
  // - detail 画面は URL 配列 or image objects を想定（best-effort で拾う）
  const baseImageUrls: string[] = React.useMemo(() => {
    const urls =
      (vm?.imagePreviewUrls as string[]) ||
      (vm?.imageUrls as string[]) ||
      (vm?.images as any[])
        ?.map((x: any) => s(x?.url || x?.src || x?.publicUrl || x?.downloadUrl))
        .filter(Boolean) ||
      (vm?.list?.images as any[])
        ?.map((x: any) => s(x?.url || x?.src || x?.publicUrl || x?.downloadUrl))
        .filter(Boolean) ||
      [];
    return urls.filter((u) => !!s(u));
  }, [vm]);

  // ============================================================
  // ✅ pageHeader edit mode (single source of truth, best-effort)
  // - hook 側に isEdit / onEdit / onCancel / onSave があれば使う
  // - 無ければ local state でフォールバック
  // ============================================================
  const externalIsEdit =
    typeof vm?.isEdit === "boolean" ? (vm.isEdit as boolean) : undefined;

  const [localIsEdit, setLocalIsEdit] = React.useState(false);
  const isEdit = externalIsEdit ?? localIsEdit;

  // drafts（hook に draft があればそれを優先）
  const [draftTitle, setDraftTitle] = React.useState(listingTitle);
  const [draftDescription, setDraftDescription] = React.useState(description);
  const [draftPriceRows, setDraftPriceRows] =
    React.useState<any[]>(basePriceRows);

  const [draftImages, setDraftImages] = React.useState<DraftImage[]>(() =>
    (baseImageUrls ?? []).map((u) => ({ url: u, isNew: false })),
  );

  // 保存状態（ページ側で持つ：hook 側未実装でも UI が破綻しない）
  const [saving, setSaving] = React.useState(false);
  const [saveError, setSaveError] = React.useState("");

  // edit 開始時に、最新の base から draft を再初期化
  React.useEffect(() => {
    if (!isEdit) return;

    // hook 側に draft があれば尊重（無ければ base をコピー）
    const t = s(vm?.draftListingTitle) || listingTitle;
    const d = s(vm?.draftDescription) || description;

    setDraftTitle(t);
    setDraftDescription(d);

    const pr =
      (vm?.draftPriceRows as any[]) ||
      (vm?.draftPrices as any[]) ||
      basePriceRows;
    setDraftPriceRows(Array.isArray(pr) ? pr.map((x) => ({ ...x })) : []);

    const imgs = (vm?.draftImages as any[]) || null;
    if (Array.isArray(imgs) && imgs.length > 0) {
      // draftImages が hook から来る場合は url を拾う（best-effort）
      const next = imgs
        .map((x) => s(x?.url || x?.src || x?.publicUrl || x?.downloadUrl))
        .filter(Boolean)
        .map((u) => ({ url: u, isNew: false as const }));
      setDraftImages(next);
    } else {
      setDraftImages((baseImageUrls ?? []).map((u) => ({ url: u, isNew: false })));
    }

    setSaveError("");
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isEdit]);

  const handleEdit = React.useCallback(() => {
    if (typeof vm?.onEdit === "function") {
      try {
        vm.onEdit();
        return;
      } catch {
        // fallthrough
      }
    }
    setLocalIsEdit(true);
  }, [vm]);

  const handleCancel = React.useCallback(() => {
    // hook 側キャンセルがあれば呼ぶ
    if (typeof vm?.onCancel === "function") {
      try {
        vm.onCancel();
      } catch {
        // noop
      }
    } else if (typeof vm?.onCancelEdit === "function") {
      try {
        vm.onCancelEdit();
      } catch {
        // noop
      }
    }

    // ✅ 追加した blob URL を解放
    revokeDraftBlobUrls(draftImages);

    // local draft を base に戻す
    setDraftTitle(listingTitle);
    setDraftDescription(description);
    setDraftPriceRows(
      Array.isArray(basePriceRows) ? basePriceRows.map((x) => ({ ...x })) : [],
    );
    setDraftImages((baseImageUrls ?? []).map((u) => ({ url: u, isNew: false })));

    setSaveError("");

    // local edit を終了
    if (externalIsEdit === undefined) {
      setLocalIsEdit(false);
    }
  }, [
    vm,
    listingTitle,
    description,
    basePriceRows,
    baseImageUrls,
    externalIsEdit,
    draftImages,
  ]);

  const handleSave = React.useCallback(async () => {
    setSaveError("");
    setSaving(true);

    // ✅ 画像はこの PUT では更新しない（サーバ側の対応が無い前提）
    // ✅ priceRows は id (= modelId) / modelId を含む shape をそのまま渡す
    const payloadForHook = {
      title: draftTitle,
      description: draftDescription,
      priceRows: draftPriceRows,
      keepImageUrls: draftImages.filter((x) => !x.isNew).map((x) => x.url),
      newImageFiles: draftImages
        .filter((x) => x.isNew && x.file)
        .map((x) => x.file as File),
    };

    try {
      // 1) hook 側保存が実装されているなら、それを優先して呼ぶ
      if (typeof vm?.onSaveEdit === "function") {
        await vm.onSaveEdit(payloadForHook);
      } else if (typeof vm?.onSave === "function") {
        await vm.onSave(payloadForHook);
      } else if (typeof vm?.save === "function") {
        await vm.save(payloadForHook);
      } else {
        // 2) ✅ hook 側が未実装なら、このページから直接 PUT /lists/{id} を叩く
        const id = s(listId);
        if (!id) {
          throw new Error("missing_list_id_for_update");
        }

        await updateListByIdHTTP({
          listId: id,
          title: draftTitle,
          description: draftDescription,
          priceRows: draftPriceRows,
          // decision はこの画面では編集していないため送らない
        });

        // 3) 更新後に再取得できる関数があれば呼ぶ（best-effort）
        if (typeof vm?.reload === "function") {
          try {
            await vm.reload();
          } catch {
            // noop
          }
        } else if (typeof vm?.refetch === "function") {
          try {
            await vm.refetch();
          } catch {
            // noop
          }
        } else if (typeof vm?.refresh === "function") {
          try {
            await vm.refresh();
          } catch {
            // noop
          }
        }
      }

      // ✅ 追加した blob URL を解放（edit 終了で参照されなくなるため）
      revokeDraftBlobUrls(draftImages);

      // local edit を終了
      if (externalIsEdit === undefined) {
        setLocalIsEdit(false);
      }
    } catch (e) {
      const msg = s(e instanceof Error ? e.message : e) || "save_failed";
      setSaveError(msg);
      return;
    } finally {
      setSaving(false);
    }
  }, [
    vm,
    draftTitle,
    draftDescription,
    draftPriceRows,
    draftImages,
    externalIsEdit,
    listId,
  ]);

  // ============================================================
  // ✅ Effective view based on mode
  // ============================================================
  const effectiveTitle = isEdit ? s(draftTitle) : listingTitle;
  const effectiveDescription = isEdit ? s(draftDescription) : description;
  const effectivePriceRows = isEdit ? draftPriceRows : basePriceRows;

  const effectiveImageUrls = isEdit
    ? draftImages.map((x) => x.url).filter((u) => !!s(u))
    : baseImageUrls;

  const hasImages = effectiveImageUrls.length > 0;

  // メイン画像はローカル state で切替（view/edit 共通）
  const [mainImageIndex, setMainImageIndex] = React.useState(0);

  React.useEffect(() => {
    if (!hasImages) {
      setMainImageIndex(0);
      return;
    }
    if (mainImageIndex >= effectiveImageUrls.length) {
      setMainImageIndex(0);
    }
  }, [hasImages, effectiveImageUrls.length, mainImageIndex]);

  const mainUrl = hasImages ? effectiveImageUrls[mainImageIndex] : "";

  const thumbIndices = React.useMemo(() => {
    if (!hasImages) return [];
    return effectiveImageUrls
      .map((_, idx) => idx)
      .filter((idx) => idx !== mainImageIndex);
  }, [hasImages, effectiveImageUrls, mainImageIndex]);

  // ============================================================
  // ✅ Edit handlers inside cards
  // ============================================================
  const onChangePrice = React.useCallback(
    (index: number, price: number | null, row: any) => {
      // hook 側ハンドラがあるなら先に委譲
      if (typeof vm?.onChangePrice === "function") {
        try {
          vm.onChangePrice(index, price, row);
        } catch {
          // noop
        }
      }
      setDraftPriceRows((prev) => {
        const next = Array.isArray(prev) ? prev.map((x) => ({ ...x })) : [];
        if (index < 0 || index >= next.length) return next;
        next[index] = { ...next[index], price };
        return next;
      });
    },
    [vm],
  );

  const onAddImages = React.useCallback((files: FileList | null) => {
    if (!files || files.length === 0) return;

    const next: DraftImage[] = [];
    for (let i = 0; i < files.length; i++) {
      const f = files.item(i);
      if (!f) continue;
      const url = URL.createObjectURL(f);
      next.push({ url, file: f, isNew: true });
    }

    setDraftImages((prev) => [...prev, ...next]);
  }, []);

  const onRemoveImageAt = React.useCallback((idx: number) => {
    setDraftImages((prev) => {
      if (!Array.isArray(prev)) return prev;
      if (idx < 0 || idx >= prev.length) return prev;

      const target = prev[idx];
      // 新規 blob URL は revoke しておく
      if (target?.isNew && target?.url?.startsWith("blob:")) {
        try {
          URL.revokeObjectURL(target.url);
        } catch {
          // noop
        }
      }

      const next = prev.slice(0, idx).concat(prev.slice(idx + 1));
      return next;
    });
  }, []);

  return (
    <PageStyle
      layout="grid-2"
      // ✅ header には title のみ（id は出さない）
      title={effectiveTitle || "出品詳細"}
      onBack={vm?.onBack}
      // ✅ view 中は編集、edit 中はキャンセル/保存
      onEdit={!isEdit ? handleEdit : undefined}
      onCancel={isEdit ? handleCancel : undefined}
      onSave={isEdit ? handleSave : undefined}
      onCreate={undefined}
    >
      {/* =========================
          左カラム
          - 商品画像（edit 対応）
          - タイトル（edit 対応）
          - 説明（edit 対応）
          - 価格（PriceCard: edit 対応）
          ========================= */}
      <div className="space-y-4">
        {/* 状態表示（任意） */}
        {loading && (
          <div className="text-sm text-[hsl(var(--muted-foreground))]">
            読み込み中...
          </div>
        )}
        {error && (
          <div className="text-sm text-red-600">
            読み込みに失敗しました: {error}
          </div>
        )}

        {/* ✅ 保存エラー（ページ側フォールバック時も表示できる） */}
        {isEdit && saveError && (
          <div className="text-sm text-red-600">
            保存に失敗しました: {saveError}
          </div>
        )}
        {isEdit && saving && (
          <div className="text-xs text-[hsl(var(--muted-foreground))]">
            保存中...
          </div>
        )}

        {/* ✅ 商品画像カード（view/edit） */}
        <Card>
          <CardContent className="p-4 space-y-3">
            <div className="text-sm font-medium flex items-center justify-between gap-2">
              <div className="flex items-center gap-2">
                <span className="inline-flex items-center justify-center w-6 h-6 rounded-md bg-slate-50 border border-slate-200">
                  <ImageIcon />
                </span>
                商品画像
              </div>

              {isEdit && (
                <div className="flex items-center gap-2">
                  <label className="cursor-pointer">
                    <input
                      type="file"
                      accept="image/*"
                      multiple
                      className="hidden"
                      onChange={(e) => onAddImages(e.target.files)}
                    />
                    <Button type="button" variant="outline" className="h-8" disabled={saving}>
                      追加
                    </Button>
                  </label>
                </div>
              )}
            </div>

            {!hasImages && (
              <div className="rounded-xl border border-dashed border-slate-300 bg-slate-50/30 w-full aspect-[16/9] flex flex-col items-center justify-center gap-3 select-none">
                <div className="w-12 h-12 rounded-lg bg-white border border-slate-200 flex items-center justify-center">
                  <ImageIcon />
                </div>
                <div className="text-sm text-slate-700">画像は未設定です</div>
                <div className="text-xs text-[hsl(var(--muted-foreground))]">
                  {isEdit
                    ? "右上の「追加」から画像を追加できます。"
                    : "画像を追加する場合は「画像」機能（別画面/別操作）から追加してください。"}
                </div>
              </div>
            )}

            {hasImages && (
              <>
                {/* メイン（大） */}
                <div className="relative rounded-xl overflow-hidden border border-slate-200 bg-white">
                  <div className="w-full aspect-[16/9] bg-slate-50">
                    {mainUrl && (
                      <img
                        src={mainUrl}
                        alt="main"
                        className="w-full h-full object-cover"
                      />
                    )}
                  </div>

                  <div className="px-3 py-2 border-t border-slate-200 flex items-center justify-between">
                    <div className="text-xs text-[hsl(var(--muted-foreground))]">
                      {effectiveImageUrls.length} 枚
                      {isEdit
                        ? "（サムネから削除できます）"
                        : "（クリックでサブ画像をメインにできます）"}
                    </div>

                    {isEdit && effectiveImageUrls.length > 0 && (
                      <Button
                        type="button"
                        variant="outline"
                        className="h-8"
                        onClick={() => onRemoveImageAt(mainImageIndex)}
                        disabled={saving}
                      >
                        この画像を削除
                      </Button>
                    )}
                  </div>
                </div>

                {/* サブ（小） */}
                {thumbIndices.length > 0 && (
                  <div className="grid grid-cols-2 sm:grid-cols-3 gap-3">
                    {thumbIndices.map((idx) => {
                      const url = effectiveImageUrls[idx];
                      return (
                        <div key={`${url}-${idx}`} className="space-y-2">
                          <div
                            className="relative rounded-xl overflow-hidden border border-slate-200 bg-white cursor-pointer"
                            onClick={() => setMainImageIndex(idx)}
                            role="button"
                            tabIndex={0}
                            title="クリックでメインに設定"
                          >
                            <div className="w-full aspect-square bg-slate-50">
                              {url && (
                                <img
                                  src={url}
                                  alt={`sub-${idx}`}
                                  className="w-full h-full object-cover"
                                />
                              )}
                            </div>
                          </div>

                          {isEdit && (
                            <Button
                              type="button"
                              variant="outline"
                              className="h-8 w-full"
                              onClick={() => onRemoveImageAt(idx)}
                              disabled={saving}
                            >
                              削除
                            </Button>
                          )}
                        </div>
                      );
                    })}
                  </div>
                )}
              </>
            )}
          </CardContent>
        </Card>

        {/* ✅ タイトル（view/edit） */}
        <Card>
          <CardContent className="p-4 space-y-2">
            <div className="text-sm font-medium">タイトル</div>

            {!isEdit && (
              <div className="text-sm text-slate-800 break-words">
                {listingTitle || "未設定"}
              </div>
            )}

            {isEdit && (
              <Input
                value={draftTitle}
                placeholder="タイトルを入力"
                onChange={(e) => setDraftTitle(e.target.value)}
                disabled={saving}
              />
            )}
          </CardContent>
        </Card>

        {/* ✅ 説明（view/edit） */}
        <Card>
          <CardContent className="p-4 space-y-2">
            <div className="text-sm font-medium">説明</div>

            {!isEdit && (
              <div className="text-sm text-slate-800 whitespace-pre-wrap break-words">
                {description || "未設定"}
              </div>
            )}

            {isEdit && (
              <textarea
                value={draftDescription}
                placeholder="説明を入力"
                onChange={(e) => setDraftDescription(e.target.value)}
                className="w-full min-h-[120px] rounded-md border border-slate-200 bg-white px-3 py-2 text-sm outline-none"
                disabled={saving}
              />
            )}
          </CardContent>
        </Card>

        {/* ✅ PriceCard（view/edit） */}
        <PriceCard
          title="価格"
          rows={effectivePriceRows as any}
          mode={isEdit ? "edit" : "view"}
          currencySymbol="¥"
          onChangePrice={isEdit ? onChangePrice : undefined}
        />

        {Array.isArray(effectivePriceRows) && effectivePriceRows.length === 0 && (
          <div className="text-xs text-[hsl(var(--muted-foreground))]">
            価格情報がありません。
          </div>
        )}
      </div>

      {/* =========================
          右カラム（view-only）
          - 担当者
          - 選択商品
          - 選択トークン
          - 出品｜保留（表示のみ）
          ========================= */}
      <div className="space-y-4">
        {/* ✅ 担当者（view-only） */}
        <Card>
          <CardContent className="p-4">
            <div className="text-sm font-medium mb-2">担当者</div>
            <div className="text-sm text-slate-800 break-all">{assigneeName}</div>

            {/* 管理情報（あれば） */}
            {(vm?.admin?.createdByName || vm?.admin?.createdAt) && (
              <div className="mt-3 text-xs text-[hsl(var(--muted-foreground))] space-y-1">
                {vm?.admin?.createdByName && (
                  <div>作成者: {s(vm.admin.createdByName)}</div>
                )}
                {vm?.admin?.createdAt && <div>作成日時: {s(vm.admin.createdAt)}</div>}
              </div>
            )}
          </CardContent>
        </Card>

        {/* ✅ 選択商品（view-only） */}
        <Card>
          <CardContent className="p-4">
            <div className="text-sm font-medium mb-2">選択商品</div>
            <div className="text-sm text-slate-800 break-all">
              {productBrandName || "未選択"}
            </div>
            <div className="text-sm text-slate-800 break-all">
              {productName || "未選択"}
            </div>
          </CardContent>
        </Card>

        {/* ✅ 選択トークン（view-only） */}
        <Card>
          <CardContent className="p-4">
            <div className="text-sm font-medium mb-2">選択トークン</div>
            <div className="text-sm text-slate-800 break-all">
              {tokenBrandName || "未選択"}
            </div>
            <div className="text-sm text-slate-800 break-all">
              {tokenName || "未選択"}
            </div>
          </CardContent>
        </Card>

        {/* ✅ 出品｜保留（view-only） */}
        <Card>
          <CardContent className="p-4">
            <div className="text-sm font-medium mb-2">出品｜保留</div>

            <div className="flex gap-2">
              <div
                className={[
                  "flex-1 h-9 rounded-md border text-sm flex items-center justify-center",
                  s(decision).toLowerCase() === "list"
                    ? "bg-slate-900 text-white border-slate-900"
                    : "bg-white text-slate-700 border-slate-200",
                ].join(" ")}
              >
                出品
              </div>

              <div
                className={[
                  "flex-1 h-9 rounded-md border text-sm flex items-center justify-center",
                  s(decision).toLowerCase() === "hold"
                    ? "bg-slate-900 text-white border-slate-900"
                    : "bg-white text-slate-700 border-slate-200",
                ].join(" ")}
              >
                保留
              </div>
            </div>

            {decision && (
              <div className="mt-2 text-xs text-[hsl(var(--muted-foreground))]">
                現在: {decision}
              </div>
            )}
          </CardContent>
        </Card>

        {/* ✅ 編集中のヒント */}
        {isEdit && (
          <div className="text-xs text-[hsl(var(--muted-foreground))]">
            編集モードです（保存すると反映されます）。
            {!s(listId) && (
              <span className="block text-red-600 mt-1">
                ※ listId が取得できていないため、ページ側の直接PUT更新はできません（hook 側 onSaveEdit 実装が必要）。
              </span>
            )}
          </div>
        )}
      </div>
    </PageStyle>
  );
}
