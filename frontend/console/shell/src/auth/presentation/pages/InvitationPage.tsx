// frontend/shell/src/pages/InvitationPage.tsx
import * as React from "react";
import PageStyle from "../../../layout/PageStyle/PageStyle";
import { Input } from "../../../shared/ui/input";

/**
 * 招待ページ（氏名・かな + パスワード入力）
 * - メンバー作成時に設定された companyId / assignedBrands / permissions の表示欄を追加
 */
export default function InvitationPage() {
  const formRef = React.useRef<HTMLFormElement>(null);

  const [lastName, setLastName] = React.useState("");
  const [lastNameKana, setLastNameKana] = React.useState("");
  const [firstName, setFirstName] = React.useState("");
  const [firstNameKana, setFirstNameKana] = React.useState("");

  // ★ 追加：パスワード
  const [password, setPassword] = React.useState("");
  const [passwordConfirm, setPasswordConfirm] = React.useState("");

  // ★ 追加：表示用の companyId / assignedBrandId / permissions
  //   - 実際には API や AuthContext から取得して setXXX する想定
  const [companyId] = React.useState<string>("");
  const [assignedBrandIds] = React.useState<string>(""); // カンマ区切りなどで表示想定
  const [permissions] = React.useState<string>(""); // カンマ区切りなどで表示想定

  const handleBack = () => history.back();
  const handleCreate = () => formRef.current?.requestSubmit();

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    // ここで API 呼び出しなどを行う
    // console.log({
    //   lastName,
    //   lastNameKana,
    //   firstName,
    //   firstNameKana,
    //   password,
    //   passwordConfirm,
    //   companyId,
    //   assignedBrandIds,
    //   permissions,
    // });
  };

  return (
    <PageStyle title="メンバー招待" onBack={handleBack} onCreate={handleCreate}>
      <div className="p-4 max-w-3xl mx-auto">
        <form ref={formRef} onSubmit={handleSubmit} className="space-y-4" noValidate>
          {/* 姓 → 姓（かな） */}
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-sm text-slate-300 mb-1">姓</label>
              <Input
                variant="default"
                className="w-full rounded border border-slate-600 bg-slate-800 px-3 py-2"
                value={lastName}
                onChange={(e) => setLastName(e.target.value)}
                placeholder="山田"
              />
            </div>
            <div>
              <label className="block text-sm text-slate-300 mb-1">姓（かな）</label>
              <Input
                variant="default"
                className="w-full rounded border border-slate-600 bg-slate-800 px-3 py-2"
                value={lastNameKana}
                onChange={(e) => setLastNameKana(e.target.value)}
                placeholder="やまだ"
              />
            </div>
          </div>

          {/* 名 → 名（かな） */}
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-sm text-slate-300 mb-1">名</label>
              <Input
                variant="default"
                className="w-full rounded border border-slate-600 bg-slate-800 px-3 py-2"
                value={firstName}
                onChange={(e) => setFirstName(e.target.value)}
                placeholder="太郎"
              />
            </div>
            <div>
              <label className="block text-sm text-slate-300 mb-1">名（かな）</label>
              <Input
                variant="default"
                className="w-full rounded border border-slate-600 bg-slate-800 px-3 py-2"
                value={firstNameKana}
                onChange={(e) => setFirstNameKana(e.target.value)}
                placeholder="たろう"
              />
            </div>
          </div>

          {/* ★ 追加：パスワード */}
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-sm text-slate-300 mb-1">パスワード</label>
              <Input
                type="password"
                variant="default"
                className="w-full rounded border border-slate-600 bg-slate-800 px-3 py-2"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="8文字以上"
              />
            </div>

            <div>
              <label className="block text-sm text-slate-300 mb-1">
                パスワード（確認用）
              </label>
              <Input
                type="password"
                variant="default"
                className="w-full rounded border border-slate-600 bg-slate-800 px-3 py-2"
                value={passwordConfirm}
                onChange={(e) => setPasswordConfirm(e.target.value)}
                placeholder="もう一度入力"
              />
            </div>
          </div>

          {/* ★ 追加：割り当て情報（companyId / assignedBrandId / permissions の表示） */}
          <div className="mt-4 space-y-3">
            <h2 className="text-sm font-semibold text-slate-200">割り当て情報</h2>

            <div>
              <label className="block text-sm text-slate-300 mb-1">会社ID（companyId）</label>
              <Input
                variant="default"
                className="w-full rounded border border-slate-600 bg-slate-900 px-3 py-2 text-slate-200"
                value={companyId}
                readOnly
              />
            </div>

            <div>
              <label className="block text-sm text-slate-300 mb-1">
                割り当てブランドID（assignedBrandId）
              </label>
              <Input
                variant="default"
                className="w-full rounded border border-slate-600 bg-slate-900 px-3 py-2 text-slate-200"
                value={assignedBrandIds}
                readOnly
                placeholder="例：brand-001, brand-002"
              />
            </div>

            <div>
              <label className="block text-sm text-slate-300 mb-1">権限（permissions）</label>
              <Input
                variant="default"
                className="w-full rounded border border-slate-600 bg-slate-900 px-3 py-2 text-slate-200"
                value={permissions}
                readOnly
                placeholder="例：member.read, member.write ..."
              />
            </div>
          </div>
        </form>
      </div>
    </PageStyle>
  );
}
