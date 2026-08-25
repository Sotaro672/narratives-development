// frontend/console/shell/src/features/account/presentation/hook/useAccountConnect.tsx  
  
import {  
  useCallback,  
  useEffect,  
  useMemo,  
  useState,  
} from "react";  
import { useNavigate } from "react-router-dom";  
  
import { useAuthContext } from "../../../../auth/application/AuthContext";  
import { accountRepositoryHTTP } from "../../infrastructure/http/accountRepositoryHTTP";  
  
const DEFAULT_BANK_NAME = "テスト銀行";  
const DEFAULT_BRANCH_NAME = "テスト支店";  
const DEFAULT_ACCOUNT_NUMBER = "1234567";  
  
const ACCOUNT_CONNECT_STORAGE_KEY =  
  "amol:account-connect:account-id";  
  
function getErrorMessage(error: unknown): string {  
  if (  
    error instanceof Error &&  
    error.message  
  ) {  
    return error.message;  
  }  
  
  return "Stripe口座との接続に失敗しました。";  
}  
  
export function useAccountConnect() {  
  const navigate = useNavigate();  
  const { currentMember } = useAuthContext();  
  
  const [  
    bankName,  
    setBankName,  
  ] = useState(  
    DEFAULT_BANK_NAME,  
  );  
  
  const [  
    branchName,  
    setBranchName,  
  ] = useState(  
    DEFAULT_BRANCH_NAME,  
  );  
  
  const [  
    accountNumber,  
    setAccountNumber,  
  ] = useState(  
    DEFAULT_ACCOUNT_NUMBER,  
  );  
  
  const [  
    createdAccountId,  
    setCreatedAccountId,  
  ] = useState("");  
  
  const [  
    submitting,  
    setSubmitting,  
  ] = useState(false);  
  
  const [  
    error,  
    setError,  
  ] = useState<string | null>(null);  
  
  const [  
    completed,  
    setCompleted,  
  ] = useState(false);  
  
  useEffect(() => {  
    let cancelled = false;  
  
    const params =  
      new URLSearchParams(  
        window.location.search,  
      );  
  
    const isCompleted =  
      params.get("completed") === "1";  
  
    const storedAccountId =  
      String(  
        window.sessionStorage.getItem(  
          ACCOUNT_CONNECT_STORAGE_KEY,  
        ) ?? "",  
      ).trim();  
  
    setCompleted(  
      isCompleted,  
    );  
  
    if (!storedAccountId) {  
      return () => {  
        cancelled = true;  
      };  
    }  
  
    setCreatedAccountId(  
      storedAccountId,  
    );  
  
    const load = async () => {  
      try {  
        setError(null);  
  
        const account =  
          isCompleted  
            ? await accountRepositoryHTTP.syncStripeStatus(  
                storedAccountId,  
              )  
            : await accountRepositoryHTTP.getById(  
                storedAccountId,  
              );  
  
        if (cancelled) {  
          return;  
        }  
  
        if (account.bankName) {  
          setBankName(  
            account.bankName,  
          );  
        }  
  
        if (account.branchName) {  
          setBranchName(  
            account.branchName,  
          );  
        }  
  
        if (  
          Number.isInteger(  
            account.accountNumber,  
          ) &&  
          account.accountNumber > 0  
        ) {  
          setAccountNumber(  
            String(  
              account.accountNumber,  
            ),  
          );  
        }  
  
        if (isCompleted) {  
          window.sessionStorage.removeItem(  
            ACCOUNT_CONNECT_STORAGE_KEY,  
          );  
        }  
      } catch (  
        caughtError: unknown  
      ) {  
        if (cancelled) {  
          return;  
        }  
  
        setError(  
          getErrorMessage(  
            caughtError,  
          ),  
        );  
      }  
    };  
  
    void load();  
  
    return () => {  
      cancelled = true;  
    };  
  }, []);  
  
  const canConnect = useMemo(  
    () =>  
      !submitting &&  
      Boolean(bankName.trim()) &&  
      Boolean(branchName.trim()) &&  
      Boolean(accountNumber.trim()),  
    [  
      submitting,  
      bankName,  
      branchName,  
      accountNumber,  
    ],  
  );  
  
  const handleBankNameChange =  
    useCallback(  
      (value: string) => {  
        setBankName(value);  
        setError(null);  
      },  
      [],  
    );  
  
  const handleBranchNameChange =  
    useCallback(  
      (value: string) => {  
        setBranchName(value);  
        setError(null);  
      },  
      [],  
    );  
  
  const handleAccountNumberChange =  
    useCallback(  
      (value: string) => {  
        const digits =  
          value.replace(  
            /[^0-9]/g,  
            "",  
          );  
  
        setAccountNumber(  
          digits.slice(0, 8),  
        );  
  
        setError(null);  
      },  
      [],  
    );  
  
  const handleBack =  
    useCallback(() => {  
      navigate(-1);  
    }, [navigate]);  
  
  const handleConnect =  
    useCallback(async () => {  
      if (submitting) {  
        return;  
      }  
  
      const memberEmail =  
        String(  
          currentMember?.email ?? "",  
        ).trim();  
  
      const normalizedBankName =  
        bankName.trim();  
  
      const normalizedBranchName =  
        branchName.trim();  
  
      const normalizedAccountNumber =  
        accountNumber.trim();  
  
      if (!memberEmail) {  
        setError(  
          "Memberのメールアドレスを取得できません。",  
        );  
        return;  
      }  
  
      if (!normalizedBankName) {  
        setError(  
          "銀行名を入力してください。",  
        );  
        return;  
      }  
  
      if (!normalizedBranchName) {  
        setError(  
          "支店名を入力してください。",  
        );  
        return;  
      }  
  
      if (  
        !/^\d{1,8}$/.test(  
          normalizedAccountNumber,  
        )  
      ) {  
        setError(  
          "口座番号は8桁以内の数字で入力してください。",  
        );  
        return;  
      }  
  
      try {  
        setSubmitting(true);  
        setError(null);  
  
        const origin =  
          window.location.origin;  
  
        const response =  
          await accountRepositoryHTTP.connect({  
            accountId:  
              createdAccountId ||  
              undefined,  
            contactEmail:  
              memberEmail,  
            country:  
              "JP",  
            returnUrl:  
              `${origin}/account/connect?completed=1`,  
            refreshUrl:  
              `${origin}/account/connect`,  
          });  
  
        const accountId =  
          response.account?.id ?? "";  
  
        if (!accountId) {  
          throw new Error(  
            "Account IDを取得できませんでした。",  
          );  
        }  
  
        setCreatedAccountId(  
          accountId,  
        );  
  
        window.sessionStorage.setItem(  
          ACCOUNT_CONNECT_STORAGE_KEY,  
          accountId,  
        );  
  
        await accountRepositoryHTTP.update(  
          accountId,  
          {  
            bankName:  
              normalizedBankName,  
            branchName:  
              normalizedBranchName,  
            accountNumber:  
              Number(  
                normalizedAccountNumber,  
              ),  
          },  
        );  
  
        const onboardingUrl =  
          response.onboardingUrl?.trim();  
  
        if (!onboardingUrl) {  
          throw new Error(  
            "Stripe onboarding URLを取得できませんでした。",  
          );  
        }  
  
        window.location.assign(  
          onboardingUrl,  
        );  
      } catch (  
        caughtError: unknown  
      ) {  
        setError(  
          getErrorMessage(  
            caughtError,  
          ),  
        );  
      } finally {  
        setSubmitting(false);  
      }  
    }, [  
      currentMember?.email,  
      bankName,  
      branchName,  
      accountNumber,  
      createdAccountId,  
      submitting,  
    ]);  
  
  return {  
    bankName,  
    branchName,  
    accountNumber,  
  
    submitting,  
    error,  
    completed,  
    canConnect,  
  
    handleBankNameChange,  
    handleBranchNameChange,  
    handleAccountNumberChange,  
  
    handleBack,  
    handleConnect,  
  };  
}