// frontend\console\shell\src\pages\accountManagement.tsx   
   
import { useNavigate } from "react-router-dom";   
import List from "../layout/List/List";   
import { useAccountManagement } from "../features/account/presentation/hook/useAccountManagement";   
import "../styles/account.css";   
   
export default function AccountManagementPage() {   
  const navigate = useNavigate();   
   
  const {   
    accounts,   
    loading,   
    error,   
  } = useAccountManagement();   
   
  if (loading) {   
    return (   
      <div className="p-4">   
        読み込み中...   
      </div>   
    );   
  }   
   
  if (error) {   
    return (   
      <div className="p-4 text-red-500">   
        データ取得エラー: {error.message}   
      </div>   
    );   
  }   
   
  const headers = [   
    "銀行名",   
    "支店名",   
    "口座番号",   
    "種別",   
    "ステータス",   
    "登録日",   
    "更新日",   
  ];   
   
  return (   
    <div className="p-0">   
      <List   
        title="口座管理"   
        headerCells={headers}   
        showCreateButton   
        createLabel="口座追加"   
        showResetButton={false}   
        onCreate={() => navigate("/account/create")}   
      >   
        {accounts.length === 0 ? (   
          <tr>   
            <td colSpan={7}>   
              <div className="account-empty">   
                登録されている口座はありません。   
              </div>   
            </td>   
          </tr>   
        ) : (   
          accounts.map((account) => (   
            <tr key={account.id}>   
              <td>{account.bankName}</td>   
              <td>{account.branchName}</td>   
              <td>{account.accountNumberLabel}</td>   
              <td>{account.accountTypeLabel}</td>   
              <td>{account.statusLabel}</td>   
              <td>{account.registeredAt}</td>   
              <td>{account.updatedAt}</td>   
            </tr>   
          ))   
        )}   
      </List>   
    </div>   
  );   
}