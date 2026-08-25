//frontend\console\shell\src\pages\transactionList.tsx  
  
import React from "react";  
import List from "../layout/List/List";  
import { useTransactionList } from "../features/transaction/presentation/hook/useTransactionList";  
import "../styles/transaction.css";  
  
export default function TransactionListPage() {  
  const {  
    transactions,  
    loading,  
    isResetting,  
    error,  
    reload,  
  } = useTransactionList();  
  
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
  
  const headers: React.ReactNode[] = [  
    "日時",  
    "口座",  
    "種別",  
    "説明",  
    "金額",  
    "ステータス",  
  ];  
  
  return (  
    <div className="p-0">  
      <List  
        title="取引履歴"  
        headerCells={headers}  
        showCreateButton={false}  
        showResetButton  
        isResetting={isResetting}  
        onReset={reload}  
      >  
        {transactions.length === 0 ? (  
          <tr>  
            <td colSpan={6}>  
              <div className="transaction-empty">  
                取引履歴はありません。  
              </div>  
            </td>  
          </tr>  
        ) : (  
          transactions.map((transaction) => (  
            <tr key={transaction.id}>  
              <td>{transaction.timestampLabel}</td>  
              <td>{transaction.accountLabel}</td>  
              <td>{transaction.typeLabel}</td>  
              <td>{transaction.description}</td>  
              <td>{transaction.amountLabel}</td>  
              <td>{transaction.statusLabel}</td>  
            </tr>  
          ))  
        )}  
      </List>  
    </div>  
  );  
}  