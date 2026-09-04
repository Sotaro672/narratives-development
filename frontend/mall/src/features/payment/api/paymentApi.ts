// frontend/amol/src/features/payment/api/paymentApi.ts 
 
import { 
  HttpError, 
  requestJson, 
} from "../../../lib/http"; 
 
import type { 
  CreateOrderRequest, 
  CreatedOrder, 
  PaymentContext, 
} from "../../shared/types/payment"; 
import type { 
  CardPaymentMethod, 
  PaymentMethodDefaultResponse, 
  PaymentMethodListResponse, 
} from "../../shared/types/paymentMethods"; 
 
export async function fetchPaymentContext(): Promise<PaymentContext> { 
  return requestJson<PaymentContext>( 
    "/mall/me/payments", 
    { 
      method: "GET", 
      auth: "required", 
      credentials: "include", 
    }, 
  ); 
} 
 
export async function fetchPaymentMethods(): Promise<{ 
  methods: CardPaymentMethod[]; 
  defaultMethod: 
    CardPaymentMethod | null; 
}> { 
  const [ 
    listBody, 
    defaultBody, 
  ] = await Promise.all([ 
    requestJson<PaymentMethodListResponse>( 
      "/mall/me/payment-methods", 
      { 
        method: "GET", 
        auth: "required", 
        credentials: "include", 
        messages: { 
          requestErrorMessage: 
            "支払い方法の取得に失敗しました。", 
        }, 
      }, 
    ), 
 
    requestJson<PaymentMethodDefaultResponse>( 
      "/mall/me/payment-methods/default", 
      { 
        method: "GET", 
        auth: "required", 
        credentials: "include", 
        messages: { 
          requestErrorMessage: 
            "既定の支払い方法の取得に失敗しました。", 
        }, 
      }, 
    ).catch( 
      ( 
        error: unknown, 
      ) => { 
        if ( 
          error instanceof HttpError && 
          error.status === 404 
        ) { 
          return null; 
        } 
 
        throw error; 
      }, 
    ), 
  ]); 
 
  return { 
    methods: 
      Array.isArray( 
        listBody?.data, 
      ) 
        ? listBody.data 
        : [], 
 
    defaultMethod: 
      defaultBody?.data ?? 
      null, 
  }; 
} 
 
export async function createOrder( 
  input: CreateOrderRequest, 
): Promise<CreatedOrder> { 
  const order = 
    await requestJson<CreatedOrder>( 
      "/mall/me/orders", 
      { 
        method: "POST", 
        auth: "required", 
        credentials: "include", 
        json: input, 
      }, 
    ); 
 
  return { 
    ...order, 
 
    id: 
      order.id ?? 
      input.id, 
 
    paid: 
      order.paid ?? 
      false, 
  }; 
}