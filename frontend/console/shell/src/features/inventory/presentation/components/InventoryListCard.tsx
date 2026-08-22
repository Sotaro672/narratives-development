// frontend/console/shell/src/features/inventory/presentation/components/InventoryListCard.tsx 
 
import * as React from "react"; 
import { Tag } from "lucide-react"; 
 
import { 
  Card, 
  CardContent, 
  CardHeader, 
  CardTitle, 
} from "../../../../shared/ui/card"; 
import { Button } from "../../../../shared/ui/button"; 
 
export type InventoryListCardItem = { 
  id: string; 
  readableId: string; 
}; 
 
export type InventoryListCardProps = { 
  items: InventoryListCardItem[]; 
  loading?: boolean; 
  error?: string | null; 
  onList: () => void; 
  onOpenList: (listId: string) => void; 
}; 
 
const InventoryListCard: React.FC<InventoryListCardProps> = ({ 
  items, 
  loading = false, 
  error = null, 
  onList, 
  onOpenList, 
}) => { 
  return ( 
    <Card> 
      <CardHeader className="flex flex-row items-center justify-between gap-3"> 
        <CardTitle>出品</CardTitle> 
 
        <Button type="button" onClick={onList}> 
          <Tag size={16} className="mr-2" /> 
          出品 
        </Button> 
      </CardHeader> 
 
      <CardContent> 
        {loading ? ( 
          <div className="text-sm text-slate-500"> 
            出品情報を読み込み中です... 
          </div> 
        ) : error ? ( 
          <div className="text-sm text-red-600"> 
            出品情報の取得に失敗しました: {error} 
          </div> 
        ) : items.length === 0 ? ( 
          <div className="text-sm text-slate-500"> 
            この在庫の出品はまだありません。 
          </div> 
        ) : ( 
          <div className="divide-y divide-slate-200 rounded-md border border-slate-200"> 
            {items.map((item) => ( 
              <div 
                key={item.id} 
                className="px-3 py-2 text-sm text-slate-700" 
              > 
                <button 
                  type="button" 
                  className="text-blue-600 hover:underline" 
                  onClick={() => onOpenList(item.id)} 
                > 
                  {item.readableId || item.id} 
                </button> 
              </div> 
            ))} 
          </div> 
        )} 
      </CardContent> 
    </Card> 
  ); 
}; 
 
export default InventoryListCard;