// frontend/sns/lib/features/order/infrastructure/order_repository_http.dart
import 'dart:convert';

import 'package:firebase_auth/firebase_auth.dart';
import 'package:http/http.dart' as http;

/// Orders API client (buyer-facing)
/// - POST /mall/orders
class OrderRepositoryHttp {
  OrderRepositoryHttp({http.Client? client, String? apiBase})
    : _client = client ?? http.Client(),
      _apiBase = (apiBase ?? '').trim();

  final http.Client _client;

  /// Optional override. If empty, resolved base will be used.
  final String _apiBase;

  void dispose() {
    _client.close();
  }

  // ------------------------------------------------------------
  // API base
  // ------------------------------------------------------------

  static const String _fallbackBaseUrl =
      'https://narratives-backend-871263659099.asia-northeast1.run.app';

  /// ✅ unify with other repos: --dart-define=API_BASE_URL=https://...
  static String _resolveApiBase() {
    const fromDefine = String.fromEnvironment('API_BASE_URL');
    final base = (fromDefine.isNotEmpty ? fromDefine : _fallbackBaseUrl).trim();
    return base.endsWith('/') ? base.substring(0, base.length - 1) : base;
  }

  // ------------------------------------------------------------
  // Public API
  // ------------------------------------------------------------

  Future<Map<String, dynamic>> createOrder({
    required String userId,
    required String avatarId,
    required String cartId,
    required Map<String, dynamic> shippingSnapshot,
    required Map<String, dynamic> billingSnapshot,
    required List<Map<String, dynamic>> items,
    String? invoiceId, // optional
    String? paymentId, // optional
    String? id,
    String? transferedDateRfc3339,
    String? updatedBy,
  }) async {
    final uri = _uri('/mall/orders');
    final headers = await _headersJsonAuthOptional();

    final body = <String, dynamic>{
      'id': _s(id),
      'userId': userId.trim(),
      'avatarId': avatarId.trim(),
      'cartId': cartId.trim(),
      'shippingSnapshot': shippingSnapshot,
      'billingSnapshot': billingSnapshot,
      'items': items,

      // ✅ only include when present (backend can treat as optional)
      if (_s(invoiceId).isNotEmpty) 'invoiceId': _s(invoiceId),
      if (_s(paymentId).isNotEmpty) 'paymentId': _s(paymentId),

      if (_s(transferedDateRfc3339).isNotEmpty)
        'transferedDate': _s(transferedDateRfc3339),
      if (_s(updatedBy).isNotEmpty) 'updatedBy': _s(updatedBy),
    };

    final res = await _client.post(
      uri,
      headers: headers,
      body: jsonEncode(body),
    );

    if (res.statusCode >= 200 && res.statusCode < 300) {
      final decoded = _decodeJson(res.body);

      // wrapper 吸収: {data:{...}} を許容
      if (decoded is Map<String, dynamic>) {
        final data = decoded['data'];
        if (data is Map<String, dynamic>) return data;
        if (data is Map) return Map<String, dynamic>.from(data);
        return decoded;
      }
      if (decoded is Map) {
        return Map<String, dynamic>.from(decoded);
      }

      return {'data': decoded};
    }

    _throwHttpError(res);
    throw StateError('unreachable');
  }

  // ------------------------------------------------------------
  // Auth
  // ------------------------------------------------------------

  Future<Map<String, String>> _headersJsonAuthOptional() async {
    final headers = <String, String>{
      'Content-Type': 'application/json; charset=utf-8',
      'Accept': 'application/json',
    };

    final user = FirebaseAuth.instance.currentUser;
    if (user != null) {
      // nullable 扱いになる SDK 差分に備えて明示
      final String? raw = await user.getIdToken(true);
      final token = (raw ?? '').trim();
      if (token.isNotEmpty) {
        headers['Authorization'] = 'Bearer $token';
      }
    }
    return headers;
  }

  // ------------------------------------------------------------
  // URL helpers
  // ------------------------------------------------------------

  Uri _uri(String path, {Map<String, String>? qp}) {
    final base = (_apiBase.isNotEmpty ? _apiBase : _resolveApiBase()).trim();

    if (base.isEmpty) {
      throw StateError(
        'API_BASE_URL is not set (use --dart-define=API_BASE_URL=https://...)',
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

  // ------------------------------------------------------------
  // JSON / HTTP helpers
  // ------------------------------------------------------------

  dynamic _decodeJson(String body) {
    final raw = body.trim().isEmpty ? 'null' : body;
    return jsonDecode(raw);
  }

  void _throwHttpError(http.Response res) {
    final status = res.statusCode;
    var msg = 'HTTP $status';

    try {
      final v = _decodeJson(res.body);
      if (v is Map) {
        final e = (v['error'] ?? v['message'] ?? '').toString().trim();
        if (e.isNotEmpty) msg = e;
      } else {
        final s = res.body.trim();
        if (s.isNotEmpty) msg = s;
      }
    } catch (_) {
      final s = res.body.trim();
      if (s.isNotEmpty) msg = s;
    }

    throw OrderHttpException(statusCode: status, message: msg, body: res.body);
  }

  static String _s(String? v) => (v ?? '').trim();
}

class OrderHttpException implements Exception {
  OrderHttpException({
    required this.statusCode,
    required this.message,
    required this.body,
  });

  final int statusCode;
  final String message;
  final String body;

  @override
  String toString() => 'OrderHttpException($statusCode): $message body=$body';
}
