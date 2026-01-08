import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:http/http.dart' as http;
import 'package:firebase_auth/firebase_auth.dart';

import '../../../app/config/api_base.dart';

/// Shared HTTP API helper for Mall feature repositories.
///
/// - Resolves API base URL via resolveApiBase() unless overridden by apiBase.
/// - Adds Firebase ID token as Bearer auth header.
/// - Retries once on 401 with forceRefreshToken=true.
/// - Provides JSON decode helpers and safe logging (never logs token).
class ApiClient {
  ApiClient({
    required String tag,
    http.Client? client,
    FirebaseAuth? auth,
    String? apiBase,
  }) : _tag = tag,
       _client = client ?? http.Client(),
       _auth = auth ?? FirebaseAuth.instance,
       _apiBaseOverride = (apiBase ?? '').trim() {
    final base = _resolveBase();
    if (base.isEmpty) {
      throw StateError(
        'API base URL is empty (API_BASE_URL / API_BASE / fallback).',
      );
    }
    _log('[$_tag] init baseUrl=$base');
  }

  final String _tag;
  final http.Client _client;
  final FirebaseAuth _auth;

  /// Optional override. If empty, resolveApiBase() will be used.
  final String _apiBaseOverride;

  // ✅ release でもログを出したい場合: --dart-define=ENABLE_HTTP_LOG=true
  static const bool _envHttpLog = bool.fromEnvironment(
    'ENABLE_HTTP_LOG',
    defaultValue: false,
  );

  bool get _logEnabled => kDebugMode || _envHttpLog;

  // ---------------------------------------------------------------------------
  // URL
  // ---------------------------------------------------------------------------

  String _resolveBase() {
    final baseRaw =
        (_apiBaseOverride.isNotEmpty ? _apiBaseOverride : resolveApiBase())
            .trim();
    return baseRaw.replaceAll(RegExp(r'\/+$'), '');
  }

  Uri uri(String path, [Map<String, String>? query]) {
    final base = _resolveBase();
    if (base.isEmpty) {
      throw StateError(
        'API base URL is empty (API_BASE_URL / API_BASE / fallback).',
      );
    }
    final p = path.startsWith('/') ? path : '/$path';
    return Uri.parse('$base$p').replace(queryParameters: query);
  }

  // ---------------------------------------------------------------------------
  // Auth / headers
  // ---------------------------------------------------------------------------

  Map<String, String> _headersJson() => const {
    'Accept': 'application/json',
    'Content-Type': 'application/json; charset=utf-8',
  };

  Future<Map<String, String>> _authHeaders({
    bool forceRefreshToken = false,
  }) async {
    final u = _auth.currentUser;
    if (u == null) {
      throw const HttpException(
        statusCode: 401,
        message: 'not_signed_in',
        url: '',
      );
    }

    final raw = await u.getIdToken(forceRefreshToken);
    final token = (raw ?? '').trim();
    if (token.isEmpty) {
      throw const HttpException(
        statusCode: 401,
        message: 'empty_id_token',
        url: '',
      );
    }

    // ✅ token は絶対にログに出さない
    return <String, String>{
      ..._headersJson(),
      'Authorization': 'Bearer $token',
    };
  }

  // ---------------------------------------------------------------------------
  // Send (authed) with 401 retry
  // ---------------------------------------------------------------------------

  Future<http.Response> sendAuthed(
    String method,
    Uri uri, {
    Map<String, dynamic>? jsonBody,
    Map<String, dynamic>? logPayload,
  }) async {
    http.Response res;

    final h1 = await _authHeaders(forceRefreshToken: false);
    _logRequest(method, uri, headers: h1, payload: logPayload ?? jsonBody);

    res = await _sendRaw(method, uri, headers: h1, jsonBody: jsonBody);
    _logResponse(method, uri, res.statusCode, res.body);

    if (res.statusCode != 401) return res;

    final h2 = await _authHeaders(forceRefreshToken: true);
    _logRequest(method, uri, headers: h2, payload: logPayload ?? jsonBody);

    final res2 = await _sendRaw(method, uri, headers: h2, jsonBody: jsonBody);
    _logResponse(method, uri, res2.statusCode, res2.body);

    return res2;
  }

  Future<http.Response> _sendRaw(
    String method,
    Uri uri, {
    required Map<String, String> headers,
    Map<String, dynamic>? jsonBody,
  }) async {
    final m = method.trim().toUpperCase();

    if (jsonBody == null) {
      switch (m) {
        case 'GET':
          return _client.get(uri, headers: headers);
        case 'DELETE':
          return _client.delete(uri, headers: headers);
        case 'POST':
          return _client.post(uri, headers: headers);
        case 'PATCH':
          return _client.patch(uri, headers: headers);
        case 'PUT':
          return _client.put(uri, headers: headers);
        default:
          final req = http.Request(m, uri);
          req.headers.addAll(headers);
          final streamed = await _client.send(req);
          return http.Response.fromStream(streamed);
      }
    }

    final body = jsonEncode(jsonBody);

    switch (m) {
      case 'POST':
        return _client.post(uri, headers: headers, body: body);
      case 'PATCH':
        return _client.patch(uri, headers: headers, body: body);
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

  // ---------------------------------------------------------------------------
  // JSON helpers
  // ---------------------------------------------------------------------------

  Map<String, dynamic> decodeObject(String body) {
    final b = body.trim();
    if (b.isEmpty) {
      throw const FormatException('Empty response body (expected object)');
    }
    final decoded = jsonDecode(b);
    if (decoded is Map<String, dynamic>) return decoded;
    if (decoded is Map) return Map<String, dynamic>.from(decoded);
    throw const FormatException('Invalid JSON shape (expected object)');
  }

  Map<String, dynamic> unwrapData(Map<String, dynamic> decoded) {
    final data = decoded['data'];
    if (data is Map<String, dynamic>) return data;
    if (data is Map) return Map<String, dynamic>.from(data);
    return decoded;
  }

  String? extractError(String body) {
    try {
      final decoded = jsonDecode(body);
      if (decoded is Map) {
        final e = decoded['error'] ?? decoded['message'];
        if (e != null) {
          final s = e.toString().trim();
          return s.isEmpty ? null : s;
        }
      }
    } catch (_) {
      // ignore
    }
    final s = body.trim();
    return s.isEmpty ? null : s;
  }

  void ensureSuccess(http.Response res, Uri uri) {
    if (res.statusCode >= 200 && res.statusCode < 300) return;
    throw HttpException(
      statusCode: res.statusCode,
      message: extractError(res.body) ?? 'request failed',
      url: uri.toString(),
    );
  }

  // ---------------------------------------------------------------------------
  // Logging
  // ---------------------------------------------------------------------------

  void _log(String msg) {
    if (!_logEnabled) return;
    debugPrint(msg);
  }

  void _logRequest(
    String method,
    Uri uri, {
    required Map<String, String> headers,
    required Map<String, dynamic>? payload,
  }) {
    if (!_logEnabled) return;

    // Authorization は伏せる
    final safeHeaders = <String, String>{};
    headers.forEach((k, v) {
      if (k.toLowerCase() == 'authorization') {
        safeHeaders[k] = 'Bearer ***';
      } else {
        safeHeaders[k] = v;
      }
    });

    final b = StringBuffer();
    b.writeln('[$_tag] request');
    b.writeln('  method=$method');
    b.writeln('  url=$uri');
    b.writeln('  headers=${jsonEncode(safeHeaders)}');
    if (payload != null) {
      b.writeln('  payload=${_truncate(jsonEncode(payload), 1500)}');
    }
    debugPrint(b.toString());
  }

  void _logResponse(String method, Uri uri, int status, String body) {
    if (!_logEnabled) return;

    final truncated = _truncate(body, 1500);
    final b = StringBuffer();
    b.writeln('[$_tag] response');
    b.writeln('  method=$method');
    b.writeln('  url=$uri');
    b.writeln('  status=$status');
    if (truncated.isNotEmpty) {
      b.writeln('  body=$truncated');
    }
    debugPrint(b.toString());
  }

  String _truncate(String s, int max) {
    final t = s.trim();
    if (t.length <= max) return t;
    return '${t.substring(0, max)}...(truncated ${t.length - max} chars)';
  }

  void dispose() {
    _client.close();
  }
}

@immutable
class HttpException implements Exception {
  const HttpException({
    required this.statusCode,
    required this.message,
    required this.url,
  });

  final int statusCode;
  final String message;
  final String url;

  @override
  String toString() => 'HttpException($statusCode) $message ($url)';
}
