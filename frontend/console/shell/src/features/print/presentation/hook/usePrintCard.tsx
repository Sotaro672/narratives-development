// frontend/console/product/src/presentation/hook/usePrintCard.tsx  
  
import * as React from "react";  
import {  
  outputProductIdsCsvForProduction,  
  outputQrForProduction,  
  preparePrintForProduction,  
  type PrintLogForPrint,  
} from "../../application/printService";  
  
type UsePrintCardParams = {  
  productionId: string | null;  
  hasProduction: boolean;  
  onCompleted?: (logs: PrintLogForPrint[]) => void;  
};  
  
type OutputType = "print" | "qr" | "csv";  
  
/**  
 * 商品IDタグ用 Product 発行・出力ロジックをまとめた Hook。  
 */  
export function usePrintCard({  
  productionId,  
  hasProduction,  
  onCompleted,  
}: UsePrintCardParams) {  
  const [printLogs, setPrintLogs] = React.useState<PrintLogForPrint[]>([]);  
  const [outputting, setOutputting] = React.useState<OutputType | null>(null);  
  const [error, setError] = React.useState<string | null>(null);  
  
  const outputtingRef = React.useRef(false);  
  
  const printing = outputting === "print";  
  const qrOutputting = outputting === "qr";  
  const csvOutputting = outputting === "csv";  
  const busy = outputting !== null;  
  
  /**  
   * 初回印刷処理。  
   *  
   * 既存 print_log がある場合は GET 結果を返す。  
   * 既存 print_log が無い場合だけ、products / print_log / inspections を作成する。  
   * QR PDF / CSV の出力は行わない。  
   */  
  const onPrint = React.useCallback(async (): Promise<PrintLogForPrint[]> => {  
    if (!productionId || !hasProduction || outputtingRef.current) return [];  
  
    try {  
      outputtingRef.current = true;  
      setOutputting("print");  
      setError(null);  
  
      const logs = await preparePrintForProduction({ productionId });  
      setPrintLogs(logs);  
      onCompleted?.(logs);  
  
      return logs;  
    } catch {  
      const message = "印刷用のデータ作成に失敗しました";  
      setError(message);  
      alert(message);  
      return [];  
    } finally {  
      outputtingRef.current = false;  
      setOutputting(null);  
    }  
  }, [productionId, hasProduction, onCompleted]);  
  
  /**  
   * QR 出力処理。  
   *  
   * 既存 print_log がある場合は GET 結果から QR PDF を表示する。  
   * 既存 print_log が無い場合だけ、初回作成として POST 処理に進む。  
   */  
  const onQrOutput = React.useCallback(async (): Promise<PrintLogForPrint[]> => {  
    if (!productionId || !hasProduction || outputtingRef.current) return [];  
  
    try {  
      outputtingRef.current = true;  
      setOutputting("qr");  
      setError(null);  
  
      const logs = await outputQrForProduction({ productionId });  
      setPrintLogs(logs);  
      onCompleted?.(logs);  
  
      return logs;  
    } catch {  
      const message = "QR出力用のデータ作成に失敗しました";  
      setError(message);  
      alert(message);  
      return [];  
    } finally {  
      outputtingRef.current = false;  
      setOutputting(null);  
    }  
  }, [productionId, hasProduction, onCompleted]);  
  
  /**  
   * CSV 出力処理。  
   *  
   * 既存 print_log がある場合は GET 結果から productId を CSV 出力する。  
   * 既存 print_log が無い場合だけ、初回作成として POST 処理に進む。  
   */  
  const onCsvOutput = React.useCallback(async (): Promise<PrintLogForPrint[]> => {  
    if (!productionId || !hasProduction || outputtingRef.current) return [];  
  
    try {  
      outputtingRef.current = true;  
      setOutputting("csv");  
      setError(null);  
  
      const logs = await outputProductIdsCsvForProduction({ productionId });  
      setPrintLogs(logs);  
      onCompleted?.(logs);  
  
      return logs;  
    } catch {  
      const message = "CSV出力用のデータ作成に失敗しました";  
      setError(message);  
      alert(message);  
      return [];  
    } finally {  
      outputtingRef.current = false;  
      setOutputting(null);  
    }  
  }, [productionId, hasProduction, onCompleted]);  
  
  return {  
    onPrint,  
    onQrOutput,  
    onCsvOutput,  
    printLogs,  
    printing,  
    qrOutputting,  
    csvOutputting,  
    busy,  
    error,  
  };  
}