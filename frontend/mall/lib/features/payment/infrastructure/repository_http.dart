// frontend\mall\lib\features\payment\infrastructure\repository_http.dart
import 'package:http/http.dart' as http;

import 'api.dart';

/// PaymentRepositoryHttp
/// - Backend の order_query.go（uid -> avatarId / shipping / billing）を叩いて
///   payment 画面が必要なコンテキストをまとめて取得する。
///
/// 想定エンドポイント:
/// - GET /mall/payment
///   Authorization: `Bearer <Firebase ID token>`
///   -> { uid, avatarId, userId, shippingAddress?, billingAddress? }
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
