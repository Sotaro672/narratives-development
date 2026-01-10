// frontend/mall/lib/features/payment/infrastructure/api.dart
import 'dart:convert';

import 'package:firebase_auth/firebase_auth.dart';
import 'package:http/http.dart' as http;

// ✅ API_BASE 解決ロジックは共通（single source of truth）
import '../../../app/config/api_base.dart';

/// PaymentApi
/// - 認証（Firebase ID token）
/// - URI 組み立て（API_BASE_URL / fallback）
/// - JSON decode / HTTP error mapping
class PaymentApi {
  PaymentApi({http.Client? client, String? apiBase})
    : client = client ?? http.Client(),
      apiBase = (apiBase ?? '').trim();

  final http.Client client;

  /// Optional override. If empty, resolveApiBase() will be used.
  final String apiBase;

  void dispose() {
    client.close();
  }

  // ------------------------------------------------------------
  // HTTP
  // ------------------------------------------------------------

  Future<Map<String, dynamic>> getJsonAuth(
    String path, {
    Map<String, String>? qp,
  }) async {
    final token = await requireFirebaseIdToken();
    final uri = buildUri(path, qp: qp);
    final headers = headersJsonAuth(token);

    final res = await client.get(uri, headers: headers);

    if (res.statusCode >= 200 && res.statusCode < 300) {
      // 204 等の空レスポンスを許容
      if (res.body.trim().isEmpty) return <String, dynamic>{};

      final m = decodeJsonMap(res.body);

      // wrapper 吸収: {data:{...}} を許容
      final data = (m['data'] is Map)
          ? (m['data'] as Map).cast<String, dynamic>()
          : m;

      return data;
    }

    throwHttpError(res);
    throw StateError('unreachable');
  }

  /// ✅ POST (auth + JSON) helper
  /// - 2xx: JSON を返す場合は Map で返す
  /// - 204: {} を返す
  /// - wrapper {data:{...}} を吸収
  Future<Map<String, dynamic>> postJsonAuth(
    String path,
    Map<String, dynamic> body, {
    Map<String, String>? qp,
  }) async {
    final token = await requireFirebaseIdToken();
    final uri = buildUri(path, qp: qp);
    final headers = headersJsonAuth(token);

    final res = await client.post(
      uri,
      headers: headers,
      body: jsonEncode(body),
    );

    if (res.statusCode >= 200 && res.statusCode < 300) {
      // 204 No Content 等
      if (res.body.trim().isEmpty) return <String, dynamic>{};

      final m = decodeJsonMap(res.body);

      // wrapper 吸収: {data:{...}} を許容
      final data = (m['data'] is Map)
          ? (m['data'] as Map).cast<String, dynamic>()
          : m;

      return data;
    }

    throwHttpError(res);
    throw StateError('unreachable');
  }

  // ------------------------------------------------------------
  // Auth
  // ------------------------------------------------------------

  Future<String> requireFirebaseIdToken() async {
    final user = FirebaseAuth.instance.currentUser;
    if (user == null) {
      throw const PaymentHttpException(
        statusCode: 401,
        message: 'not_signed_in',
      );
    }

    // true で強制更新（キャッシュ不整合を避ける）
    final String? raw = await user.getIdToken(true); // String? 扱いになる環境がある
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

  Map<String, dynamic> decodeJsonMap(String body) {
    final raw = body.trim().isEmpty ? '{}' : body;
    final v = jsonDecode(raw);
    if (v is Map<String, dynamic>) return v;
    if (v is Map) return v.cast<String, dynamic>();
    throw const FormatException('invalid json response');
  }

  void throwHttpError(http.Response res) {
    final status = res.statusCode;
    String msg = 'HTTP $status';
    try {
      final m = decodeJsonMap(res.body);
      final e = (m['error'] ?? m['message'] ?? '').toString().trim();
      if (e.isNotEmpty) msg = e;
    } catch (_) {
      final s = res.body.trim();
      if (s.isNotEmpty) msg = s;
    }

    throw PaymentHttpException(statusCode: status, message: msg);
  }

  Uri buildUri(String path, {Map<String, String>? qp}) {
    final base = (apiBase.isNotEmpty ? apiBase : resolveApiBase()).trim();

    if (base.isEmpty) {
      throw StateError(
        'API_BASE_URL is not set (use --dart-define=API_BASE_URL=https://...)',
      );
    }

    final b = Uri.parse(base);
    final cleanPath = path.startsWith('/') ? path : '/$path';
    final joinedPath = joinPaths(b.path, cleanPath);

    return Uri(
      scheme: b.scheme,
      userInfo: b.userInfo,
      host: b.host,
      port: b.hasPort ? b.port : null,
      path: joinedPath,
      queryParameters: (qp == null || qp.isEmpty) ? null : qp,
      fragment: b.fragment.isEmpty ? null : b.fragment,
    );
    // Note: Uri(...) の port は int? ではなく int なので
    // b.hasPort ? b.port : null が IDE で警告になる場合は削ってOKです（DartのSDK差分）。
  }

  String joinPaths(String a, String b) {
    final aa = a.trim();
    final bb = b.trim();
    if (aa.isEmpty || aa == '/') return bb;
    if (bb.isEmpty || bb == '/') return aa;
    if (aa.endsWith('/') && bb.startsWith('/')) return aa + bb.substring(1);
    if (!aa.endsWith('/') && !bb.startsWith('/')) return '$aa/$bb';
    return aa + bb;
  }

  Map<String, String> headersJsonAuth(String idToken) => {
    'Content-Type': 'application/json; charset=utf-8',
    'Accept': 'application/json',
    'Authorization': 'Bearer $idToken',
  };
}

class PaymentHttpException implements Exception {
  const PaymentHttpException({required this.statusCode, required this.message});

  final int statusCode;
  final String message;

  @override
  String toString() =>
      'PaymentHttpException(statusCode=$statusCode, message=$message)';
}
