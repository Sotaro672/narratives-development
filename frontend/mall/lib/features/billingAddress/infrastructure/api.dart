// frontend/mall/lib/features/billingAddress/infrastructure/api.dart
import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:http/http.dart' as http;
import 'package:firebase_auth/firebase_auth.dart';

// ✅ 共通 resolver を使う（fallback/環境変数名のブレを防ぐ）
import '../../../app/config/api_base.dart';

class ApiClient {
  ApiClient({
    required this.tag,
    http.Client? client,
    FirebaseAuth? auth,
    String? apiBase,
  }) : _client = client ?? http.Client(),
       _auth = auth ?? FirebaseAuth.instance {
    final resolvedRaw = (apiBase ?? '').trim().isNotEmpty
        ? apiBase!.trim()
        : resolveApiBase().trim();

    if (resolvedRaw.isEmpty) {
      throw Exception(
        'API base URL is empty (API_BASE_URL / API_BASE / fallback).',
      );
    }

    final normalized = resolvedRaw.replaceAll(RegExp(r'\/+$'), '');
    final u = Uri.parse(normalized);

    // ✅ origin（scheme/host/port）だけをベースにする
    _origin = u.replace(path: '', query: null, fragment: null);

    // ✅ baseUrl に /mall が含まれている注入も許容
    // - ".../mall" or ".../mall/..." なら caller 側では /mall を付けない
    final basePath = u.path.replaceAll(RegExp(r'\/+$'), '');
    _baseHasMall = basePath == '/mall' || basePath.endsWith('/mall');
  }

  final String tag;
  final http.Client _client;
  final FirebaseAuth _auth;

  late final Uri _origin;
  late final bool _baseHasMall;

  // ✅ release でもログを出したい場合: --dart-define=ENABLE_HTTP_LOG=true
  static const bool _envHttpLog = bool.fromEnvironment(
    'ENABLE_HTTP_LOG',
    defaultValue: false,
  );
  bool get _logEnabled => kDebugMode || _envHttpLog;

  void dispose() {
    _client.close();
  }

  // ------------------------------------------------------------
  // URI builder
  // ------------------------------------------------------------

  /// Build URI under origin + /mall prefix.
  ///
  /// - Accepts: "me/xxx", "/me/xxx", "mall/me/xxx", "/mall/me/xxx"
  /// - Ensures: "/mall" is added exactly once unless apiBase already contains "/mall"
  Uri uri(String path) {
    var p = path.trim();
    if (p.isEmpty) return _origin;

    // normalize leading slashes
    p = p.replaceAll(RegExp(r'^/+'), '');

    // if caller already passed mall prefix, don't double it
    final alreadyMall = p == 'mall' || p.startsWith('mall/');

    final needsMall = !_baseHasMall && !alreadyMall;

    final fullPath = needsMall ? '/mall/$p' : '/$p';

    // normalize duplicate slashes
    final normalizedPath = fullPath.replaceAll(RegExp(r'\/+'), '/');

    return _origin.replace(path: normalizedPath);
  }

  // ------------------------------------------------------------
  // HTTP helpers
  // ------------------------------------------------------------

  Future<http.Response> sendAuthed(
    String method,
    Uri uri, {
    Map<String, dynamic>? jsonBody,
    Map<String, dynamic>? logPayload,
  }) async {
    final m = method.toUpperCase();

    final headers = <String, String>{
      'Content-Type': 'application/json',
      'Accept': 'application/json',
    };

    // firebase token
    try {
      final u = _auth.currentUser;
      if (u != null) {
        final token = await u.getIdToken(false);
        final t = (token ?? '').toString().trim();
        if (t.isNotEmpty) {
          headers['Authorization'] = 'Bearer $t';
        }
      }
    } catch (e) {
      _log('[$tag] token error: $e');
    }

    final bodyStr = jsonBody == null ? null : jsonEncode(jsonBody);

    _logRequest(m, uri, headers, logPayload ?? jsonBody);

    switch (m) {
      case 'GET':
        return _client.get(uri, headers: headers);
      case 'POST':
        return _client.post(uri, headers: headers, body: bodyStr);
      case 'PATCH':
        return _client.patch(uri, headers: headers, body: bodyStr);
      case 'PUT':
        return _client.put(uri, headers: headers, body: bodyStr);
      case 'DELETE':
        return _client.delete(uri, headers: headers);
      default:
        throw Exception('unsupported method: $m');
    }
  }

  void ensureSuccess(http.Response res, Uri uri) {
    if (res.statusCode >= 200 && res.statusCode < 300) return;
    _log(
      '[$tag] HTTP ${res.statusCode} ${uri.toString()} body=${_truncate(res.body, 1200)}',
    );
    throw Exception('HTTP ${res.statusCode}: ${res.body}');
  }

  Map<String, dynamic> decodeObject(String body) {
    final s = body.trim();
    if (s.isEmpty) return <String, dynamic>{};
    final decoded = jsonDecode(s);
    if (decoded is Map<String, dynamic>) return decoded;
    if (decoded is Map) return Map<String, dynamic>.from(decoded);
    throw Exception('invalid json: expected object');
  }

  Map<String, dynamic> unwrapData(Map<String, dynamic> decoded) {
    final d = decoded['data'];
    if (d is Map<String, dynamic>) return d;
    if (d is Map) return Map<String, dynamic>.from(d);
    // data wrapper が無いサーバでも動くようにフォールバック
    return decoded;
  }

  // ------------------------------------------------------------
  // logging
  // ------------------------------------------------------------

  void _log(String msg) {
    if (!_logEnabled) return;
    debugPrint(msg);
  }

  void _logRequest(
    String method,
    Uri uri,
    Map<String, String> headers,
    Map<String, dynamic>? payload,
  ) {
    if (!_logEnabled) return;

    final maskedHeaders = <String, String>{...headers};
    if (maskedHeaders.containsKey('Authorization')) {
      maskedHeaders['Authorization'] = 'Bearer ***';
    }

    final b = StringBuffer();
    b.writeln('[$tag] request');
    b.writeln('  method=$method');
    b.writeln('  url=${uri.toString()}');
    b.writeln('  headers=${jsonEncode(maskedHeaders)}');
    if (payload != null && payload.isNotEmpty) {
      b.writeln('  body=${_truncate(jsonEncode(payload), 1500)}');
    }
    debugPrint(b.toString());
  }

  String _truncate(String s, int max) {
    final t = s.trim();
    if (t.length <= max) return t;
    return '${t.substring(0, max)}...(truncated ${t.length - max} chars)';
  }
}
