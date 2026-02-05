// frontend/mall/lib/features/avatar/infrastructure/api.dart
import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:firebase_auth/firebase_auth.dart';
import 'package:http/http.dart' as http;

// ✅ 共通 API base resolver（env名/fallbackのブレを防ぐ）
import '../../../app/config/api_base.dart';

/// Shared authed HTTP client for Mall APIs.
///
/// - Adds Firebase ID token (Bearer)
/// - Retries once on 401 with forceRefreshToken=true
/// - Safe logging (never logs token; can mask signed URLs)
class MallAuthedApi {
  MallAuthedApi({http.Client? client, FirebaseAuth? auth, String? baseUrl})
    : _client = client ?? http.Client(),
      _auth = auth ?? FirebaseAuth.instance,
      _base = _normalizeBase(
        (baseUrl ?? '').trim().isNotEmpty ? baseUrl!.trim() : resolveApiBase(),
      ) {
    if (_base.trim().isEmpty) {
      throw Exception(
        'API base URL is empty (API_BASE_URL / API_BASE / fallback).',
      );
    }
    _log('[MallAuthedApi] init baseUrl=$_base');
  }

  final http.Client _client;
  final FirebaseAuth _auth;
  final String _base;

  // ✅ release でもログを出したい場合: --dart-define=ENABLE_HTTP_LOG=true
  static const bool _envHttpLog = bool.fromEnvironment(
    'ENABLE_HTTP_LOG',
    defaultValue: false,
  );

  bool get _logEnabled => kDebugMode || _envHttpLog;

  // ------------------------------------------------------------
  // URL / JSON helpers
  // ------------------------------------------------------------

  static String _normalizeBase(String base) {
    var b = base.trim();
    while (b.endsWith('/')) {
      b = b.substring(0, b.length - 1);
    }
    return b;
  }

  Uri uri(String path, [Map<String, String>? query]) {
    final p = path.startsWith('/') ? path : '/$path';
    return Uri.parse('$_base$p').replace(queryParameters: query);
  }

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

  /// ✅ wrapper 吸収: {data:{...}} を許容
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
        final e = (decoded['error'] ?? decoded['message'] ?? '')
            .toString()
            .trim();
        return e.isEmpty ? null : e;
      }
    } catch (_) {
      // ignore
    }
    final s = body.trim();
    return s.isEmpty ? null : s;
  }

  // ------------------------------------------------------------
  // Auth / sending
  // ------------------------------------------------------------

  Map<String, String> _headersJson() => const {
    'Accept': 'application/json',
    'Content-Type': 'application/json; charset=utf-8',
  };

  Future<Map<String, String>> _headersJsonAuthed({
    bool forceRefreshToken = false,
    Uri? urlForError,
  }) async {
    final u = _auth.currentUser;
    if (u == null) {
      throw HttpException(
        statusCode: 401,
        message: 'not_signed_in',
        url: urlForError?.toString() ?? '',
      );
    }

    final raw = await u.getIdToken(forceRefreshToken);
    final token = (raw ?? '').trim();
    if (token.isEmpty) {
      throw HttpException(
        statusCode: 401,
        message: 'empty_id_token',
        url: urlForError?.toString() ?? '',
      );
    }

    // ✅ token は絶対にログに出さない
    return <String, String>{
      ..._headersJson(),
      'Authorization': 'Bearer $token',
    };
  }

  /// Sends request with Firebase Authorization header.
  /// - If 401, retry once with forceRefreshToken=true.
  Future<http.Response> sendAuthed(
    String method,
    Uri uri, {
    Map<String, dynamic>? jsonBody,
    bool allowEmptyBody = false,
  }) async {
    http.Response res;

    final h1 = await _headersJsonAuthed(
      forceRefreshToken: false,
      urlForError: uri,
    );
    _logRequest(method, uri, headers: h1, payload: jsonBody);

    res = await _sendRaw(
      method,
      uri,
      headers: h1,
      jsonBody: jsonBody,
      allowEmptyBody: allowEmptyBody,
    );
    _logResponse(method, uri, res.statusCode, res.body);

    if (res.statusCode != 401) return res;

    final h2 = await _headersJsonAuthed(
      forceRefreshToken: true,
      urlForError: uri,
    );
    _logRequest(method, uri, headers: h2, payload: jsonBody);

    final res2 = await _sendRaw(
      method,
      uri,
      headers: h2,
      jsonBody: jsonBody,
      allowEmptyBody: allowEmptyBody,
    );
    _logResponse(method, uri, res2.statusCode, res2.body);

    return res2;
  }

  Future<http.Response> _sendRaw(
    String method,
    Uri uri, {
    required Map<String, String> headers,
    Map<String, dynamic>? jsonBody,
    bool allowEmptyBody = false,
  }) async {
    final m = method.trim().toUpperCase();

    // body なし
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
    if (!allowEmptyBody && body.trim().isEmpty) {
      // 通常ここには来ないが、念のため
      return http.Response('', 400);
    }

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

  void throwHttpError(http.Response res, Uri uri) {
    final status = res.statusCode;
    final msg = extractError(res.body) ?? 'request failed';
    throw HttpException(statusCode: status, message: msg, url: uri.toString());
  }

  /// PUT to signed URL (upload bytes)
  ///
  /// - Signed URL は Authorization 不要（署名が認証）
  /// - headers は Content-Type を合わせる（署名と一致しないと 403 になる）
  Future<void> uploadToSignedUrl({
    required String uploadUrl,
    required Uint8List bytes,
    required String contentType,
  }) async {
    final u = uploadUrl.trim();
    if (u.isEmpty) throw ArgumentError('uploadUrl is empty');
    if (bytes.isEmpty) throw ArgumentError('bytes is empty');

    final ct = contentType.trim().isEmpty
        ? 'application/octet-stream'
        : contentType.trim();
    final uri = Uri.parse(u);

    // ✅ 署名付きURLは OAuth ヘッダ等を付けない
    final headers = <String, String>{'Content-Type': ct};

    // ✅ signed url はログでクエリを落とす（URL自体が権限になり得る）
    _logRequest(
      'PUT',
      uri,
      headers: headers,
      payload: {'bytes': bytes.lengthInBytes, 'contentType': ct},
      forceMaskUrl: true,
    );

    final res = await _client.put(uri, headers: headers, body: bytes);

    _logResponse('PUT', uri, res.statusCode, res.body, forceMaskUrl: true);

    if (res.statusCode < 200 || res.statusCode >= 300) {
      throw HttpException(
        statusCode: res.statusCode,
        message: extractError(res.body) ?? 'upload failed',
        url: _maskUrl(uri).toString(),
      );
    }
  }

  void dispose() {
    _client.close();
  }

  // ---------------------------------------------------------------------------
  // Logging
  // ---------------------------------------------------------------------------

  void _log(String msg) {
    if (!_logEnabled) return;
    debugPrint(msg);
  }

  Uri _maskUrl(Uri uri) {
    // signed-url を推定するキーがある、または query が長い場合は query を落とす
    final qp = uri.queryParameters;
    final hasSignedKey = qp.keys.any((k) {
      final kk = k.toLowerCase();
      return kk.contains('x-goog-signature') ||
          kk.contains('signature') ||
          kk.contains('x-amz-signature') ||
          kk.contains('x-goog-credential') ||
          kk.contains('x-amz-credential');
    });

    if (hasSignedKey || uri.query.length > 80) {
      return uri.replace(query: '');
    }
    return uri;
  }

  void _logRequest(
    String method,
    Uri uri, {
    required Map<String, String> headers,
    required Map<String, dynamic>? payload,
    bool forceMaskUrl = false,
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

    final safeUri = forceMaskUrl ? _maskUrl(uri) : uri;

    final b = StringBuffer();
    b.writeln('[MallAuthedApi] request');
    b.writeln('  method=$method');
    b.writeln('  url=$safeUri');
    b.writeln('  headers=${jsonEncode(safeHeaders)}');
    if (payload != null) {
      b.writeln('  payload=${_truncate(jsonEncode(payload), 1500)}');
    }
    debugPrint(b.toString());
  }

  void _logResponse(
    String method,
    Uri uri,
    int status,
    String body, {
    bool forceMaskUrl = false,
  }) {
    if (!_logEnabled) return;

    final safeUri = forceMaskUrl ? _maskUrl(uri) : uri;
    final truncated = _truncate(body, 1500);

    final b = StringBuffer();
    b.writeln('[MallAuthedApi] response');
    b.writeln('  method=$method');
    b.writeln('  url=$safeUri');
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
