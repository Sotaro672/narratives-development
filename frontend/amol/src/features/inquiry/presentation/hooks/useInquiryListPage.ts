// frontend/amol/src/features/inquiry/presentation/hooks/useInquiryListPage.ts 
 
import { useCallback, useEffect, useMemo, useState } from "react"; 
import { useNavigate } from "react-router-dom"; 
 
import { 
  listMeInquiries, 
  markInquiryAsRead, 
  type InquiryListItem, 
} from "../../api/inquiryApi"; 
 
export type InquiryChatListItem = InquiryListItem & { 
  chatKind: "inquiry"; 
}; 
 
function getErrorMessage(caught: unknown, fallbackMessage: string): string { 
  return caught instanceof Error ? caught.message : fallbackMessage; 
} 
 
function getComparableTime(value: string): number { 
  const date = new Date(value); 
  return Number.isNaN(date.getTime()) ? 0 : date.getTime(); 
} 
 
async function loadInquiryItems(signal?: AbortSignal): Promise<InquiryChatListItem[]> { 
  const result = await listMeInquiries({ 
    page: 1, 
    perPage: 100, 
    signal, 
  }); 
 
  if (signal?.aborted) { 
    return []; 
  } 
 
  return result.items.map((inquiry) => ({ 
    ...inquiry, 
    chatKind: "inquiry", 
  })); 
} 
 
export function useInquiryListPage() { 
  const navigate = useNavigate(); 
 
  const [items, setItems] = useState<InquiryChatListItem[]>([]); 
  const [loading, setLoading] = useState(true); 
  const [navigatingId, setNavigatingId] = useState<string | null>(null); 
  const [error, setError] = useState(""); 
 
  const sortedItems = useMemo(() => { 
    return [...items].sort((firstItem, secondItem) => { 
      const firstTime = getComparableTime(firstItem.latestActivityAt); 
      const secondTime = getComparableTime(secondItem.latestActivityAt); 
      return secondTime - firstTime; 
    }); 
  }, [items]); 
 
  const loadChats = useCallback(async (signal?: AbortSignal) => { 
    setLoading(true); 
    setError(""); 
 
    try { 
      const nextItems = await loadInquiryItems(signal); 
      if (signal?.aborted) { 
        return; 
      } 
      setItems(nextItems); 
    } catch (caught) { 
      if (signal?.aborted) { 
        return; 
      } 
 
      setItems([]); 
      setError( 
        getErrorMessage( 
          caught, 
          "チャット一覧の取得に失敗しました。", 
        ), 
      ); 
    } finally { 
      if (!signal?.aborted) { 
        setLoading(false); 
      } 
    } 
  }, []); 
 
  useEffect(() => { 
    const controller = new AbortController(); 
    void loadChats(controller.signal); 
 
    return () => { 
      controller.abort(); 
    }; 
  }, [loadChats]); 
 
  const handleOpenChat = useCallback( 
    async (item: InquiryChatListItem) => { 
      if (navigatingId) { 
        return; 
      } 
 
      const inquiryId = item.id; 
      setNavigatingId(inquiryId); 
      setError(""); 
 
      try { 
        let nextItem = item; 
 
        if (item.unreadReplyCount > 0) { 
          const updatedInquiry = await markInquiryAsRead(inquiryId); 
 
          nextItem = { 
            ...item, 
            ...updatedInquiry, 
            unreadReplyCount: 0, 
            chatKind: "inquiry", 
          }; 
 
          setItems((currentItems) => 
            currentItems.map((currentItem) => 
              currentItem.id === inquiryId ? nextItem : currentItem, 
            ), 
          ); 
        } 
 
        navigate(`/chats/${encodeURIComponent(inquiryId)}`, { 
          state: { 
            inquiry: nextItem, 
          }, 
        }); 
      } catch (caught) { 
        setError( 
          getErrorMessage( 
            caught, 
            "チャットを開く処理に失敗しました。", 
          ), 
        ); 
      } finally { 
        setNavigatingId(null); 
      } 
    }, 
    [navigate, navigatingId], 
  ); 
 
  return { 
    items, 
    sortedItems, 
    loading, 
    navigatingId, 
    error, 
    loadChats, 
    handleOpenChat, 
  }; 
}