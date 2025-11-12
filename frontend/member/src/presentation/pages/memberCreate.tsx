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
    // 氏名/かなは InvitationPage に移譲したため UI からは削除
    firstName, lastName, firstNameKana, lastNameKana,
    email,
    // ▼ 役割（Permission Category）
    category, setCategory,
    submitting, error,
    setFirstName, setLastName, setFirstNameKana, setLastNameKana,
    setEmail,
    handleSubmit,
    // ▼ hooks に選択権限（name のカンマ区切り）を渡す
    setPermissionsText,
    // ▼ 表示用
    permissionCategories,
    permissionCategoryList,
  } = useMemberCreate({
    onSuccess: () => navigate("/member"),
  });

  const handleBack = () => navigate(-1);
  const handleCreate = () => formRef.current?.requestSubmit();

  const categoryLabel = (c?: string) => c ?? "";
  const closePopover = () =>
    document.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape" }));

  const selectedCategory = React.useMemo(
    () => permissionCategories.find((x) => x.key === category),
    [permissionCategories, category]
  );
  const currentPerms = selectedCategory?.permissions ?? [];

  // ▼ 権限のチェック状態（IDベース）
  const [selectedPermIds, setSelectedPermIds] = React.useState<Set<string>>(new Set());

  const togglePerm = (permId: string, checked: boolean) => {
    setSelectedPermIds((prev) => {
      const next = new Set(prev);
      if (checked) next.add(permId);
      else next.delete(permId);
      return next;
    });
  };

  // ▼ カテゴリ配下の全権限が選択されているか
  const allSelectedInCategory =
    currentPerms.length > 0 &&
    currentPerms.every((p) => selectedPermIds.has(p.id));

  // ▼ ヘッダーの全選択チェックボックス切り替え
  const toggleAllInCategory = (checked: boolean) => {
    if (currentPerms.length === 0) return;
    setSelectedPermIds((prev) => {
      const next = new Set(prev);
      if (checked) {
        currentPerms.forEach((p) => next.add(p.id));
      } else {
        currentPerms.forEach((p) => next.delete(p.id));
      }
      return next;
    });
  };

  // ▼ 画面下部に並べる「選択済み権限バッジ」用（全カテゴリ横断）
  const allPerms = React.useMemo(
    () => permissionCategories.flatMap((c) => c.permissions),
    [permissionCategories]
  );
  const selectedPerms = React.useMemo(
    () => allPerms.filter((p) => selectedPermIds.has(p.id)),
    [allPerms, selectedPermIds]
  );

  // 送信時に hooks 側へ "name" のカンマ区切りで渡す
  const onSubmit = (e: React.FormEvent) => {
    const names = selectedPerms.map((p) => p.name);
    setPermissionsText(names.join(","));
    handleSubmit(e);
  };

  return (
    <PageStyle title="メンバー追加" onBack={handleBack} onCreate={handleCreate}>
      <div className="p-4 max-w-3xl mx-auto">
        {error && <div className="mb-3 text-red-500">エラー: {error}</div>}

        <form ref={formRef} onSubmit={onSubmit} className="space-y-4" noValidate>
          {/* 氏名/かなの入力欄は InvitationPage に移譲 */}

          {/* メールアドレス（必須） */}
          <div>
            <label className="block text-sm text-slate-300 mb-1">メールアドレス（必須）</label>
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

          {/* 役割 + 権限カード（横並び） */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 items-start">
            {/* 役割ボタン（Permission Category） */}
            <div>
              <label className="block text-sm text-slate-300 mb-1">役割（必須）</label>

              <Popover>
                <PopoverTrigger>
                  <Button
                    type="button"
                    variant="outline"
                    className="w-full justify-start border-slate-600 bg-slate-800 text-left"
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

              {/* hidden select for HTML バリデーション互換 */}
              <select
                aria-hidden="true"
                tabIndex={-1}
                required
                value={category}
                onChange={(e) => setCategory(e.target.value as typeof category)}
                style={{
                  position: "absolute",
                  opacity: 0,
                  pointerEvents: "none",
                  height: 0,
                  width: 0,
                }}
              >
                {permissionCategoryList.map((c) => (
                  <option key={c} value={c}>
                    {categoryLabel(c)}
                  </option>
                ))}
              </select>
            </div>

            {/* 権限カード（選択中の permissionCategory に属する権限を表示 + チェックボックス） */}
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

              {/* ▼ 選択した権限をバッジで一覧表示（全カテゴリ横断） */}
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
