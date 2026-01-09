// frontend/mall/lib/features/wallet/infrastructure/api.dart
import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:firebase_auth/firebase_auth.dart';
import 'package:http/http.dart' as http;

// ✅ 共通 resolver（API_BASE / API_BASE_URL / fallback を吸収）
import '../../../app/config/api_base.dart';

/// Shared authed API client for Mall wallet-related endpoints.
/// - Firebase Bearer auth
/// - 401 -> retry once with forceRefreshToken=true
/// - JSON decode + {data:{...}} unwrap
/// - Safe logging (never prints token)
class MallAuthedApi {
  MallAuthedApi({
    http.Client? client,
    FirebaseAuth? auth,
    String? baseUrl,
    void Function(String s)? logger,
  }) : _client = client ?? http.Client(),
       _auth = auth ?? FirebaseAuth.instance,
       _logger = logger,
       _base = _normalizeBase(
         (baseUrl ?? '').trim().isNotEmpty
             ? baseUrl!.trim()
             : resolveMallApiBase(),
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
  final void Function(String s)? _logger;
  final String _base;

  // ✅ release でもログを出したい場合: --dart-define=ENABLE_HTTP_LOG=true
  static const bool _envHttpLog = bool.fromEnvironment(
    'ENABLE_HTTP_LOG',
    defaultValue: false,
  );

  bool get _logEnabled => kDebugMode || _envHttpLog;

  int? _lastStatusCode;
  int? get lastStatusCode => _lastStatusCode;

  void dispose() {
    _client.close();
  }

  void _log(String s) {
    if (!_logEnabled) return;
    if (_logger != null) {
      _logger.call(s);
    } else {
      debugPrint(s);
    }
  }

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

  Future<Map<String, String>> _authHeaders({bool forceRefresh = false}) async {
    final headers = <String, String>{};

    final user = _auth.currentUser;
    if (user == null) {
      return headers;
    }

    final token = await user.getIdToken(forceRefresh);
    final t = (token ?? '').toString().trim();
    if (t.isNotEmpty) headers['Authorization'] = 'Bearer $t';

    return headers;
  }

  /// GET with Firebase Authorization.
  /// - If 401, retry once with forceRefresh=true.
  Future<http.Response> getAuthed(Uri uri) async {
    // first try
    final h1 = <String, String>{
      'Accept': 'application/json',
      ...await _authHeaders(forceRefresh: false),
    };
    _logRequest('GET', uri, headers: h1);

    final res1 = await _client.get(uri, headers: h1);
    _lastStatusCode = res1.statusCode;
    _logResponse('GET', uri, res1.statusCode, res1.body);

    if (res1.statusCode != 401) return res1;

    // retry
    final h2 = <String, String>{
      'Accept': 'application/json',
      ...await _authHeaders(forceRefresh: true),
    };
    _logRequest('GET', uri, headers: h2);

    final res2 = await _client.get(uri, headers: h2);
    _lastStatusCode = res2.statusCode;
    _logResponse('GET', uri, res2.statusCode, res2.body);

    return res2;
  }

  void _logRequest(
    String method,
    Uri uri, {
    required Map<String, String> headers,
  }) {
    if (!_logEnabled) return;

    final safeHeaders = <String, String>{};
    headers.forEach((k, v) {
      if (k.toLowerCase() == 'authorization') {
        safeHeaders[k] = 'Bearer ***';
      } else {
        safeHeaders[k] = v;
      }
    });

    _log(
      '[MallAuthedApi] request method=$method url=$uri headers=${jsonEncode(safeHeaders)}',
    );
  }

  void _logResponse(String method, Uri uri, int status, String body) {
    if (!_logEnabled) return;

    final truncated = _truncate(body, 1500);
    if (truncated.isEmpty) {
      _log('[MallAuthedApi] response method=$method url=$uri status=$status');
      return;
    }
    _log(
      '[MallAuthedApi] response method=$method url=$uri status=$status body=$truncated',
    );
  }

  String _truncate(String s, int max) {
    final t = s.trim();
    if (t.isEmpty) return '';
    if (t.length <= max) return t;
    return '${t.substring(0, max)}...(truncated ${t.length - max} chars)';
  }
}

class HttpException implements Exception {
  const HttpException({
    required this.statusCode,
    required this.message,
    required this.url,
    this.body,
  });

  final int statusCode;
  final String message;
  final String url;
  final String? body;

  @override
  String toString() {
    final b = (body ?? '').trim();
    final bb = b.isEmpty
        ? ''
        : ' body=${b.length > 300 ? b.substring(0, 300) : b}';
    return 'HttpException($statusCode) $message ($url)$bb';
  }
}
