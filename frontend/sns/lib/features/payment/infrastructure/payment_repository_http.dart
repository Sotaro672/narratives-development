// frontend/sns/lib/features/payment/infrastructure/payment_repository_http.dart
import 'dart:convert';

import 'package:firebase_auth/firebase_auth.dart';
import 'package:http/http.dart' as http;

// ✅ API_BASE 解決ロジックは既存と揃える（inventory と同様）
import '../../inventory/infrastructure/inventory_repository_http.dart';

/// PaymentRepositoryHttp
/// - Backend の order_query.go（uid -> avatarId / shipping / billing）を叩いて
///   payment 画面が必要なコンテキストをまとめて取得する。
///
/// 想定エンドポイント:
/// - GET /sns/payment
///   Authorization: `Bearer <Firebase ID token>`
///   -> { uid, avatarId, userId, shippingAddress?, billingAddress? }
class PaymentRepositoryHttp {
  PaymentRepositoryHttp({http.Client? client, String? apiBase})
    : _client = client ?? http.Client(),
      _apiBase = (apiBase ?? const String.fromEnvironment('API_BASE')).trim();

  final http.Client _client;

  /// Optional override. If empty, resolveSnsApiBase() will be used.
  final String _apiBase;

  void dispose() {
    _client.close();
  }

  /// ✅ uid -> avatarId / shipping / billing を backend 側で解決して返してもらう
  Future<PaymentContextDTO> fetchPaymentContext() async {
    final token = await _requireFirebaseIdToken();

    final uri = _uri('/sns/payment');
    final res = await _client.get(uri, headers: _headersJsonAuth(token));

    if (res.statusCode >= 200 && res.statusCode < 300) {
      final m = _decodeJsonMap(res.body);

      // wrapper 吸収: {data:{...}} を許容
      final data = (m['data'] is Map)
          ? (m['data'] as Map).cast<String, dynamic>()
          : m;

      return PaymentContextDTO.fromJson(data);
    }

    _throwHttpError(res);
    throw StateError('unreachable');
  }

  // ------------------------------------------------------------
  // Auth
  // ------------------------------------------------------------

  Future<String> _requireFirebaseIdToken() async {
    final user = FirebaseAuth.instance.currentUser;
    if (user == null) {
      throw const PaymentHttpException(
        statusCode: 401,
        message: 'not_signed_in',
      );
    }

    // true で強制更新（キャッシュ不整合を避ける）
    final raw = await user.getIdToken(true); // String? 扱いになる環境がある
    final token = (raw ?? '').trim();

    if (token.isEmpty) {
      throw const PaymentHttpException(
        statusCode: 401,
        message: 'invalid_id_token',
      );
    }
    return token;
  }

  // ------------------------------------------------------------
  // JSON / HTTP helpers
  // ------------------------------------------------------------

  Map<String, dynamic> _decodeJsonMap(String body) {
    final raw = body.trim().isEmpty ? '{}' : body;
    final v = jsonDecode(raw);
    if (v is Map<String, dynamic>) return v;
    if (v is Map) return v.cast<String, dynamic>();
    throw const FormatException('invalid json response');
  }

  void _throwHttpError(http.Response res) {
    final status = res.statusCode;
    String msg = 'HTTP $status';
    try {
      final m = _decodeJsonMap(res.body);
      final e = (m['error'] ?? m['message'] ?? '').toString().trim();
      if (e.isNotEmpty) msg = e;
    } catch (_) {
      final s = res.body.trim();
      if (s.isNotEmpty) msg = s;
    }
    throw PaymentHttpException(statusCode: status, message: msg);
  }

  Uri _uri(String path, {Map<String, String>? qp}) {
    final base = (_apiBase.isNotEmpty ? _apiBase : resolveSnsApiBase()).trim();

    if (base.isEmpty) {
      throw StateError(
        'API_BASE is not set (use --dart-define=API_BASE=https://...)',
      );
    }

    final b = Uri.parse(base);
    final cleanPath = path.startsWith('/') ? path : '/$path';
    final joinedPath = _joinPaths(b.path, cleanPath);

    return Uri(
      scheme: b.scheme,
      userInfo: b.userInfo,
      host: b.host,
      port: b.hasPort ? b.port : null,
      path: joinedPath,
      queryParameters: (qp == null || qp.isEmpty) ? null : qp,
      fragment: b.fragment.isEmpty ? null : b.fragment,
    );
  }

  String _joinPaths(String a, String b) {
    final aa = a.trim();
    final bb = b.trim();
    if (aa.isEmpty || aa == '/') return bb;
    if (bb.isEmpty || bb == '/') return aa;
    if (aa.endsWith('/') && bb.startsWith('/')) return aa + bb.substring(1);
    if (!aa.endsWith('/') && !bb.startsWith('/')) return '$aa/$bb';
    return aa + bb;
  }

  Map<String, String> _headersJsonAuth(String idToken) => {
    'Content-Type': 'application/json; charset=utf-8',
    'Accept': 'application/json',
    'Authorization': 'Bearer $idToken',
  };
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

class PaymentHttpException implements Exception {
  const PaymentHttpException({required this.statusCode, required this.message});

  final int statusCode;
  final String message;

  @override
  String toString() =>
      'PaymentHttpException(statusCode=$statusCode, message=$message)';
}
