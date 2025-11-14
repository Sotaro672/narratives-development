// frontend/member/src/presentation/pages/memberCreate.tsx
import * as React from "react";
import { useNavigate } from "react-router-dom";
import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import { useMemberCreate } from "../../hooks/useMemberCreate";
import { Input } from "../../../../shell/src/shared/ui/input";
import {
  Popover,
  PopoverTrigger,
  PopoverContent,
} from "../../../../shell/src/shared/ui/popover";
import { Button } from "../../../../shell/src/shared/ui/button";
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from "../../../../shell/src/shared/ui/card";
import { Checkbox } from "../../../../shell/src/shared/ui/checkbox";
import { Badge } from "../../../../shell/src/shared/ui/badge";
import "../styles/member.css";

export default function MemberCreatePage() {
  const navigate = useNavigate();
  const formRef = React.useRef<HTMLFormElement>(null);

  const {
    email,
    // ▼ 役割（Permission Category）
    category,
    setCategory,
    submitting,
    error,
    setEmail,
    handleSubmit,
    // ▼ hooks に選択権限（name のカンマ区切り）/ ブランドID（カンマ区切り）を渡す
    setPermissionsText,
    setBrandsText,
    // ▼ 表示用
    permissionCategories,
    permissionCategoryList,
    brandRows,
  } = useMemberCreate({
    onSuccess: () => navigate("/member"),
  });

  const handleBack = () => navigate(-1);
  const handleCreate = () => formRef.current?.requestSubmit();

  const categoryLabel = (c?: string) => c ?? "";
  const closePopover = () =>
    document.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape" }));

  // ========= 権限カテゴリ ==========
  const selectedCategory = React.useMemo(
    () => permissionCategories.find((x) => x.key === category),
    [permissionCategories, category]
  );
  const currentPerms = selectedCategory?.permissions ?? [];

  const [selectedPermIds, setSelectedPermIds] = React.useState<Set<string>>(new Set());
  const togglePerm = (permId: string, checked: boolean) => {
    setSelectedPermIds((prev) => {
      const next = new Set(prev);
      if (checked) next.add(permId);
      else next.delete(permId);
      return next;
    });
  };
  const allSelectedInCategory =
    currentPerms.length > 0 && currentPerms.every((p) => selectedPermIds.has(p.id));
  const toggleAllInCategory = (checked: boolean) => {
    if (currentPerms.length === 0) return;
    setSelectedPermIds((prev) => {
      const next = new Set(prev);
      if (checked) currentPerms.forEach((p) => next.add(p.id));
      else currentPerms.forEach((p) => next.delete(p.id));
      return next;
    });
  };
  const allPerms = React.useMemo(
    () => permissionCategories.flatMap((c) => c.permissions),
    [permissionCategories]
  );
  const selectedPerms = React.useMemo(
    () => allPerms.filter((p) => selectedPermIds.has(p.id)),
    [allPerms, selectedPermIds]
  );

  // ========= ブランド選択（Popover 内にチェックボックス） ==========
  const [selectedBrandIds, setSelectedBrandIds] = React.useState<Set<string>>(new Set());
  const toggleBrand = (id: string, explicit?: boolean) => {
    setSelectedBrandIds((prev) => {
      const next = new Set(prev);
      const willCheck = explicit ?? !next.has(id);
      if (willCheck) next.add(id);
      else next.delete(id);
      return next;
    });
  };

  // ========= 送信処理 ==========
  const onSubmit = (e: React.FormEvent) => {
    const names = selectedPerms.map((p) => p.name);
    setPermissionsText(names.join(","));
    setBrandsText(Array.from(selectedBrandIds).join(","));
    handleSubmit(e);
  };

  return (
    <PageStyle title="メンバー追加" onBack={handleBack} onCreate={handleCreate}>
      <div className="p-4 max-w-3xl mx-auto">
        {error && <div className="mb-3 text-red-500">エラー: {error}</div>}

        <form ref={formRef} onSubmit={onSubmit} className="space-y-4" noValidate>
          {/* ===== ブランド選択（Popover + Checkbox、ボタンは固定表示） ===== */}
          <div>
            <label className="block text-sm text-slate-300 mb-1">
              ブランド（任意・複数選択可）
            </label>
            <Popover>
              <PopoverTrigger>
                <Button
                  type="button"
                  variant="outline"
                  className="w-full justify-start text-left"
                >
                  ブランドを選択
                </Button>
              </PopoverTrigger>
              <PopoverContent className="w-[320px] max-h-[400px] overflow-y-auto text-sm">
                <ul className="space-y-2">
                  {brandRows.map((b) => {
                    const checked = selectedBrandIds.has(b.id);
                    const inputId = `brand_${b.id}`;
                    return (
                      <li key={b.id} className="flex items-center gap-2">
                        <Checkbox
                          id={inputId}
                          checked={checked}
                          onCheckedChange={(v) => toggleBrand(b.id, v)}
                        />
                        <label
                          id={inputId}
                          className="cursor-pointer select-none"
                          onClick={() => toggleBrand(b.id)}
                        >
                          {b.name}
                        </label>
                      </li>
                    );
                  })}
                </ul>
              </PopoverContent>
            </Popover>

            {/* 選択済みブランドのバッジ表示 */}
            <div className="mt-2 flex flex-wrap gap-2">
              {Array.from(selectedBrandIds).length === 0 ? (
                <span className="text-xs text-[hsl(var(--muted-foreground))]">
                  選択したブランドがここに表示されます。
                </span>
              ) : (
                brandRows
                  .filter((b) => selectedBrandIds.has(b.id))
                  .map((b) => <Badge key={`brand_badge_${b.id}`}>{b.name}</Badge>)
              )}
            </div>
          </div>

          {/* ===== メールアドレス（必須） ===== */}
          <div>
            <label className="block text-sm text-slate-300 mb-1">
              メールアドレス（必須）
            </label>
            <Input
              type="email"
              required
              autoComplete="email"
              variant="default"
              className="w-full rounded border border-slate-600 bg-slate-800 px-3 py-2"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="taro@example.com"
            />
          </div>

          {/* ===== 役割 + 権限カード ===== */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 items-start">
            {/* 役割選択 */}
            <div>
              <label className="block text-sm text-slate-300 mb-1">役割（必須）</label>
              <Popover>
                <PopoverTrigger>
                  <Button
                    type="button"
                    variant="outline"
                    className="role-button w-full justify-start text-left"
                  >
                    {category ? (
                      categoryLabel(category)
                    ) : (
                      <span className="text-slate-400">役割を選択</span>
                    )}
                  </Button>
                </PopoverTrigger>
                <PopoverContent className="text-sm max-h-[320px] overflow-y-auto">
                  <div className="flex flex-col gap-1">
                    {permissionCategoryList.map((c) => (
                      <button
                        key={c}
                        type="button"
                        onClick={() => {
                          setCategory(c);
                          closePopover();
                        }}
                        className={
                          "w-full text-left px-3 py-2 rounded hover:bg-slate-100 " +
                          (category === c ? "bg-slate-100 font-semibold" : "")
                        }
                      >
                        {categoryLabel(c)}
                      </button>
                    ))}
                  </div>
                </PopoverContent>
              </Popover>
            </div>

            {/* 権限一覧 */}
            <div>
              <Card>
                <CardHeader>
                  <Checkbox
                    id="category-select-all"
                    checked={allSelectedInCategory}
                    onCheckedChange={(v) => toggleAllInCategory(v)}
                  />
                  <CardTitle style={{ marginLeft: 8 }}>
                    権限一覧（{categoryLabel(category)}）
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  {currentPerms.length === 0 ? (
                    <p className="text-sm text-[hsl(var(--muted-foreground))]">
                      この役割に紐づく権限はありません。
                    </p>
                  ) : (
                    <ul className="text-sm space-y-2">
                      {currentPerms.map((perm) => {
                        const checked = selectedPermIds.has(perm.id);
                        const inputId = `perm_${perm.id}`;
                        return (
                          <li key={perm.id} className="flex items-start gap-2">
                            <Checkbox
                              id={inputId}
                              checked={checked}
                              onCheckedChange={(v) => togglePerm(perm.id, v)}
                            />
                            <label
                              id={inputId}
                              className="cursor-pointer select-none"
                              onClick={() => togglePerm(perm.id, !checked)}
                            >
                              <span className="font-medium">{perm.name}</span>
                              <span className="text-[hsl(var(--muted-foreground))]">
                                {" — "}{perm.description}
                              </span>
                            </label>
                          </li>
                        );
                      })}
                    </ul>
                  )}
                </CardContent>
              </Card>

              {/* 選択済み権限バッジ */}
              <div className="mt-3 flex flex-wrap gap-2">
                {selectedPerms.length === 0 ? (
                  <span className="text-xs text-[hsl(var(--muted-foreground))]">
                    権限を選択するとここに表示されます。
                  </span>
                ) : (
                  selectedPerms.map((perm) => (
                    <Badge key={`badge_${perm.id}`} variant="secondary">
                      {perm.name}
                    </Badge>
                  ))
                )}
              </div>
            </div>
          </div>
        </form>
      </div>
    </PageStyle>
  );
}
