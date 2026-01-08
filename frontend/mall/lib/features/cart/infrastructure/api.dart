import 'dart:convert';

import 'package:firebase_auth/firebase_auth.dart';
import 'package:http/http.dart' as http;

import '../../../app/config/api_base.dart';

/// Shared HTTP API helper for Cart repository.
///
/// - Resolves base URL via resolveApiBase() unless overridden by apiBase.
/// - Adds Firebase ID token as Bearer auth header.
/// - Retries once on 401 with forceRefreshToken=true.
/// - Provides JSON decode + error extraction.
/// - CORS: does NOT add custom headers (only Authorization + JSON headers).
class CartApiClient {
  CartApiClient({http.Client? client, String? apiBase})
    : _client = client ?? http.Client(),
      _apiBase = (apiBase ?? '').trim() {
    final base = _resolveBase();
    if (base.isEmpty) {
      throw StateError(
        'API base URL is empty (API_BASE_URL / API_BASE / fallback).',
      );
    }
  }

  final http.Client _client;

  /// Optional override. If empty, resolveApiBase() will be used.
  final String _apiBase;

  void dispose() {
    _client.close();
  }

  // ----------------------------
  // Public (used by repository)
  // ----------------------------

  Uri uri(String path, {Map<String, String>? qp}) {
    final baseRaw = (_apiBase.isNotEmpty ? _apiBase : resolveApiBase()).trim();
    if (baseRaw.isEmpty) {
      throw StateError(
        'API base URL is empty (API_BASE_URL / API_BASE / fallback).',
      );
    }

    final base = baseRaw.replaceAll(RegExp(r'\/+$'), '');
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

  /// Sends request with Firebase Authorization header.
  /// - If 401, retry once with forceRefreshToken=true.
  Future<http.Response> sendAuthed(
    String method,
    Uri uri, {
    String? body,
  }) async {
    final h1 = await _headersJsonAuthed(forceRefreshToken: false);
    final res1 = await _sendRaw(method, uri, headers: h1, body: body);
    if (res1.statusCode != 401) return res1;

    final h2 = await _headersJsonAuthed(forceRefreshToken: true);
    return _sendRaw(method, uri, headers: h2, body: body);
  }

  Map<String, dynamic> decodeJsonMap(String body) {
    final raw = body.trim().isEmpty ? '{}' : body;
    final v = jsonDecode(raw);
    if (v is Map<String, dynamic>) return v;
    if (v is Map) return v.cast<String, dynamic>();
    throw const FormatException('invalid json response');
  }

  /// wrapper 吸収: {data:{...}} を許容
  Map<String, dynamic> unwrapData(Map<String, dynamic> map) {
    final data = map['data'];
    if (data is Map) return data.cast<String, dynamic>();
    return map;
  }

  Never throwHttpError(http.Response res) {
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

    throw CartHttpException(statusCode: status, message: msg);
  }

  // ----------------------------
  // Internal
  // ----------------------------

  Future<Map<String, String>> _headersJsonAuthed({
    required bool forceRefreshToken,
  }) async {
    final u = FirebaseAuth.instance.currentUser;
    if (u == null) {
      throw CartHttpException(statusCode: 401, message: 'not_signed_in');
    }

    final idToken = await u.getIdToken(forceRefreshToken);
    final tok = (idToken ?? '').trim(); // ✅ null-safe
    if (tok.isEmpty) {
      throw CartHttpException(statusCode: 401, message: 'empty_id_token');
    }

    // ✅ token は絶対にログに出さない
    return <String, String>{'Authorization': 'Bearer $tok', ..._headersJson()};
  }

  /// ✅ CORS 的に “simple headers” 寄りにする（x- 系などカスタムは入れない）
  /// NOTE: Authorization はここには入れない（authed 時は _headersJsonAuthed で追加）
  Map<String, String> _headersJson() => const {
    'Content-Type': 'application/json; charset=utf-8',
    'Accept': 'application/json',
  };

  Future<http.Response> _sendRaw(
    String method,
    Uri uri, {
    required Map<String, String> headers,
    String? body,
  }) async {
    final m = method.trim().toUpperCase();

    // body なし
    if (body == null) {
      switch (m) {
        case 'GET':
          return _client.get(uri, headers: headers);
        case 'DELETE':
          return _client.delete(uri, headers: headers);
        case 'POST':
          return _client.post(uri, headers: headers);
        case 'PUT':
          return _client.put(uri, headers: headers);
        default:
          final req = http.Request(m, uri);
          req.headers.addAll(headers);
          final streamed = await _client.send(req);
          return http.Response.fromStream(streamed);
      }
    }

    // body あり
    switch (m) {
      case 'POST':
        return _client.post(uri, headers: headers, body: body);
      case 'PUT':
        return _client.put(uri, headers: headers, body: body);
      case 'DELETE':
        // ✅ http.delete(body) が環境/バージョンで不安定なため Request で送る
        final req = http.Request('DELETE', uri);
        req.headers.addAll(headers);
        req.body = body;
        final streamed = await _client.send(req);
        return http.Response.fromStream(streamed);
      default:
        final req = http.Request(m, uri);
        req.headers.addAll(headers);
        req.body = body;
        final streamed = await _client.send(req);
        return http.Response.fromStream(streamed);
    }
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

  String _resolveBase() {
    final baseRaw = (_apiBase.isNotEmpty ? _apiBase : resolveApiBase()).trim();
    return baseRaw.replaceAll(RegExp(r'\/+$'), '');
  }
}

class CartHttpException implements Exception {
  CartHttpException({required this.statusCode, required this.message});

  final int statusCode;
  final String message;

  @override
  String toString() =>
      'CartHttpException(statusCode=$statusCode, message=$message)';
}
