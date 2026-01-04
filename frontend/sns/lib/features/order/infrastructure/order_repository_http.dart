// frontend/sns/lib/features/order/infrastructure/order_repository_http.dart
import 'dart:convert';

import 'package:firebase_auth/firebase_auth.dart';
import 'package:http/http.dart' as http;

import '../../../app/config/api_base.dart';

/// Orders API client (buyer-facing)
/// - POST /sns/orders
class OrderRepositoryHttp {
  OrderRepositoryHttp({http.Client? client, String? apiBase})
    : _client = client ?? http.Client(),
      _apiBase = (apiBase ?? '').trim();

  final http.Client _client;

  /// Optional override. If empty, resolveSnsApiBase() will be used.
  final String _apiBase;

  void dispose() {
    _client.close();
  }

  // ------------------------------------------------------------
  // Public API
  // ------------------------------------------------------------

  Future<Map<String, dynamic>> createOrder({
    required String userId,
    required String cartId,
    required Map<String, dynamic> shippingSnapshot,
    required Map<String, dynamic> billingSnapshot,
    required List<Map<String, dynamic>> items, // ✅ snapshot items
    required String invoiceId,
    required String paymentId,
    String? id,
    String? transferedDateRfc3339,
    String? updatedBy,
  }) async {
    final uri = _uri('/sns/orders');
    final headers = await _headersJsonAuthOptional();

    final body = <String, dynamic>{
      'id': _s(id),
      'userId': userId.trim(),
      'cartId': cartId.trim(),
      'shippingSnapshot': shippingSnapshot,
      'billingSnapshot': billingSnapshot,
      'items': items,
      'invoiceId': invoiceId.trim(),
      'paymentId': paymentId.trim(),
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
      // ✅ 返り値が nullable 扱いになる SDK 差分に備えて明示
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
