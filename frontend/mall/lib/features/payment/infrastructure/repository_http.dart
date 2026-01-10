import 'package:http/http.dart' as http;

import 'api.dart';

/// PaymentRepositoryHttp
/// - Backend の /mall/me/payment を叩いて payment 画面が必要なコンテキストを取得
/// - Case A（責務分離）では /mall/me/payments で payment 起票を行う
class PaymentRepositoryHttp {
  PaymentRepositoryHttp({http.Client? client, String? apiBase})
    : _api = PaymentApi(client: client, apiBase: apiBase);

  final PaymentApi _api;

  void dispose() {
    _api.dispose();
  }

  // ------------------------------------------------------------
  // Public API
  // ------------------------------------------------------------

  /// ✅ uid -> avatarId / shipping / billing を backend 側で解決して返してもらう
  Future<PaymentContextDTO> fetchPaymentContext() async {
    final data = await _api.getJsonAuth('/mall/me/payment');
    return PaymentContextDTO.fromJson(data);
  }

  /// ✅ Case A: /mall/me/payments で payment 起票（+ dev では backend 側で自己 webhook する想定）
  ///
  /// Backend 側の受け口が:
  /// - POST /mall/me/payments
  /// body: { invoiceId, billingAddressId, amount }
  /// を想定。
  ///
  /// 204 を返す実装でも動くように、戻り値は Map を返す（空なら {}）
  Future<Map<String, dynamic>> startPayment({
    required String invoiceId,
    required String billingAddressId,
    required int amount,
  }) async {
    final body = <String, dynamic>{
      'invoiceId': invoiceId.trim(),
      'billingAddressId': billingAddressId.trim(),
      'amount': amount,
    };

    // ここで postJsonAuth を使う（PaymentApi に追加済み前提）
    final data = await _api.postJsonAuth('/mall/me/payments', body);
    return data;
  }
}

// ============================================================
// DTOs
// ============================================================

class PaymentContextDTO {
  PaymentContextDTO({
    required this.uid,
    required this.avatarId,
    required this.userId,
    required this.shippingAddress,
    required this.billingAddress,
    required this.debug,
  });

  final String uid;
  final String avatarId;
  final String userId;

  /// raw map（キー差分を吸収するため）
  final Map<String, dynamic>? shippingAddress;
  final Map<String, dynamic>? billingAddress;

  /// backend の debug フィールド（任意）
  final Map<String, String>? debug;

  static String _s(dynamic v) => (v ?? '').toString().trim();

  static Map<String, dynamic>? _mapAny(dynamic v) {
    if (v == null) return null;
    if (v is Map<String, dynamic>) return v;
    if (v is Map) return v.cast<String, dynamic>();
    return null;
  }

  static Map<String, String>? _mapString(dynamic v) {
    if (v == null) return null;
    if (v is Map<String, String>) return v;
    if (v is Map) {
      final out = <String, String>{};
      for (final e in v.entries) {
        out[e.key.toString()] = (e.value ?? '').toString();
      }
      return out;
    }
    return null;
  }

  factory PaymentContextDTO.fromJson(Map<String, dynamic> json) {
    return PaymentContextDTO(
      uid: _s(json['uid']),
      avatarId: _s(json['avatarId']),
      userId: _s(json['userId']),
      shippingAddress: _mapAny(json['shippingAddress']),
      billingAddress: _mapAny(json['billingAddress']),
      debug: _mapString(json['debug']),
    );
  }
}
