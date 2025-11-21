// frontend/console/member/src/presentation/pages/memberCreate.tsx
import * as React from "react";
import { useNavigate } from "react-router-dom";
import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import { useMemberCreate } from "../hooks/useMemberCreate";
import { Input } from "../../../../shell/src/shared/ui/input";

import { BrandSelect } from "../components/BrandSelect";
import { PermissionSelect } from "../components/PermissionSelect";

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
    // ▼ 表示用
    permissionCategories,
    permissionCategoryList,
    brandRows,
  } = useMemberCreate({
    onSuccess: () => navigate("/member"),
  });

  const handleBack = () => navigate(-1);
  const handleCreate = () => formRef.current?.requestSubmit();

  // ========= 権限選択状態 ==========
  const [selectedPermIds, setSelectedPermIds] = React.useState<Set<string>>(
    new Set(),
  );

  const allPerms = React.useMemo(
    () => permissionCategories.flatMap((c) => c.permissions as any[]),
    [permissionCategories],
  );

  const selectedPerms = React.useMemo(
    () => allPerms.filter((p) => selectedPermIds.has(p.id)),
    [allPerms, selectedPermIds],
  );

  // ========= ブランド選択（BrandSelect コンポーネントに委譲） ==========
  const [selectedBrandIds, setSelectedBrandIds] = React.useState<Set<string>>(
    new Set(),
  );
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
    const permissionNames = selectedPerms.map((p: any) => p.name as string);
    const brandIdsArray = Array.from(selectedBrandIds);

    // handleSubmit に override として渡す
    handleSubmit(e, {
      permissions: permissionNames,
      assignedBrandIds: brandIdsArray,
    });
  };

  return (
    <PageStyle
      title="メンバー追加"
      onBack={handleBack}
      onCreate={handleCreate}
    >
      <div className="p-4 max-w-3xl mx-auto">
        {error && <div className="mb-3 text-red-500">エラー: {error}</div>}

        <form
          ref={formRef}
          onSubmit={onSubmit}
          className="space-y-4"
          noValidate
        >
          {/* ===== ブランド選択 ===== */}
          <BrandSelect
            brandRows={brandRows}
            selectedBrandIds={selectedBrandIds}
            onToggleBrand={toggleBrand}
          />

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
              disabled={submitting}
            />
          </div>

          {/* ===== 役割 + 権限カード（PermissionSelect に委譲） ===== */}
          <PermissionSelect
            category={category}
            setCategory={setCategory}
            permissionCategories={permissionCategories}
            permissionCategoryList={permissionCategoryList}
            selectedPermIds={selectedPermIds}
            setSelectedPermIds={setSelectedPermIds}
          />
        </form>
      </div>
    </PageStyle>
  );
}
