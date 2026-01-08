// frontend\mall\lib\features\wallet\infrastructure\wallet_repository_http.dart
import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:firebase_auth/firebase_auth.dart';
import 'package:http/http.dart' as http;

// ✅ 共通 resolver（API_BASE / API_BASE_URL / fallback を吸収）
import '../../../app/config/api_base.dart';

class WalletDTO {
  WalletDTO({
    required this.avatarId,
    required this.walletAddress,
    required this.tokens,
    required this.lastUpdatedAt,
    required this.status,
  });

  final String avatarId;
  final String walletAddress;
  final List<String> tokens;

  /// RFC3339 string (or empty)
  final String lastUpdatedAt;

  final String status;

  static String s(dynamic v) => (v ?? '').toString().trim();

  static List<String> _tokensFrom(dynamic v) {
    if (v is List) {
      return v.map((e) => s(e)).where((x) => x.isNotEmpty).toList();
    }
    return <String>[];
  }

  /// ✅ backend が avatarId を返さない可能性があるので、
  ///    呼び出し側で補完できるようにする
  factory WalletDTO.fromJson(
    Map<String, dynamic> j, {
    String fallbackAvatarId = '',
  }) {
    dynamic pickAny(List<String> keys) {
      for (final k in keys) {
        if (j.containsKey(k)) return j[k];
      }
      return null;
    }

    var avatarId = s(pickAny(const ['avatarId', 'AvatarID', 'AvatarId']));
    if (avatarId.isEmpty) {
      avatarId = fallbackAvatarId.trim();
    }

    final walletAddress = s(
      pickAny(const ['walletAddress', 'WalletAddress', 'address', 'Address']),
    );

    final lastUpdatedAt = s(
      pickAny(const [
        'lastUpdatedAt',
        'LastUpdatedAt',
        'updatedAt',
        'UpdatedAt',
      ]),
    );

    final statusRaw = s(pickAny(const ['status', 'Status']));
    final status = statusRaw.isEmpty ? 'active' : statusRaw;

    return WalletDTO(
      avatarId: avatarId,
      walletAddress: walletAddress,
      tokens: _tokensFrom(pickAny(const ['tokens', 'Tokens'])),
      lastUpdatedAt: lastUpdatedAt,
      status: status,
    );
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

class WalletRepositoryHttp {
  WalletRepositoryHttp({
    http.Client? client,
    FirebaseAuth? auth,
    String? baseUrl,
    this.logger,
  }) : _client = client ?? http.Client(),
       _auth = auth ?? FirebaseAuth.instance,
       _base = _normalizeBase(
         (baseUrl ?? '').trim().isNotEmpty
             ? baseUrl!.trim()
             : resolveApiBase(), // ← もし resolveMallApiBase() があるなら差し替え推奨
       ) {
    if (_base.trim().isEmpty) {
      throw Exception(
        'API base URL is empty (API_BASE_URL / API_BASE / fallback).',
      );
    }
    _log('[WalletRepositoryHttp] init baseUrl=$_base');
  }

  final http.Client _client;
  final FirebaseAuth _auth;
  final String _base;

  final void Function(String s)? logger;

  static const bool _envHttpLog = bool.fromEnvironment(
    'ENABLE_HTTP_LOG',
    defaultValue: false,
  );

  bool get _logEnabled => kDebugMode || _envHttpLog;

  int? _lastStatusCode;

  void dispose() {
    _client.close();
  }

  void _log(String s) {
    if (!_logEnabled) return;
    if (logger != null) {
      logger!.call(s);
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

  Uri _uri(String path, [Map<String, String>? query]) {
    final p = path.startsWith('/') ? path : '/$path';
    return Uri.parse('$_base$p').replace(queryParameters: query);
  }

  Future<Map<String, String>> _authHeaders({bool forceRefresh = false}) async {
    final headers = <String, String>{};

    try {
      final user = _auth.currentUser;
      if (user != null) {
        final token = await user.getIdToken(forceRefresh);
        final t = (token ?? '').toString().trim();
        if (t.isNotEmpty) headers['Authorization'] = 'Bearer $t';
      }
    } catch (e) {
      _log('[WalletRepositoryHttp] token error: $e');
    }

    return headers;
  }

  // ------------------------------------------------------------
  // API (strict)
  // ------------------------------------------------------------

  /// ✅ 確定版:
  /// - GET /mall/wallets/{avatarId}
  /// - (optional) ?walletAddress=...  ※初回作成だけ
  Future<WalletDTO?> fetchByAvatarId(
    String avatarId, {
    String? walletAddressForCreate,
  }) async {
    final aid = avatarId.trim();
    if (aid.isEmpty) return null;

    final query = <String, String>{};
    final wa = (walletAddressForCreate ?? '').trim();
    if (wa.isNotEmpty) {
      query['walletAddress'] = wa;
    }

    final uri = _uri('/mall/wallets/$aid', query.isEmpty ? null : query);

    // first try
    final dto = await _tryGetWallet(
      uri,
      headers: <String, String>{
        'Accept': 'application/json',
        ...await _authHeaders(forceRefresh: false),
      },
      fallbackAvatarId: aid,
    );
    if (dto != null) return dto;

    // if 401 -> retry with refreshed token
    if (_lastStatusCode == 401) {
      final dto2 = await _tryGetWallet(
        uri,
        headers: <String, String>{
          'Accept': 'application/json',
          ...await _authHeaders(forceRefresh: true),
        },
        fallbackAvatarId: aid,
      );
      if (dto2 != null) return dto2;
    }

    return null;
  }

  Future<WalletDTO?> _tryGetWallet(
    Uri uri, {
    required Map<String, String> headers,
    required String fallbackAvatarId,
  }) async {
    try {
      _log('[WalletRepositoryHttp] GET $uri');

      final safeHeaders = <String, String>{};
      headers.forEach((k, v) {
        if (k.toLowerCase() == 'authorization') {
          safeHeaders[k] = 'Bearer ***';
        } else {
          safeHeaders[k] = v;
        }
      });
      _log('[WalletRepositoryHttp] headers=${jsonEncode(safeHeaders)}');

      final res = await _client.get(uri, headers: headers);
      _lastStatusCode = res.statusCode;

      if (res.statusCode < 200 || res.statusCode >= 300) {
        _log(
          '[WalletRepositoryHttp] non-2xx status=${res.statusCode} url=$uri bodyLen=${res.body.length}',
        );
        return null;
      }

      final body = res.body.trim();
      if (body.isEmpty) return null;

      final decoded = jsonDecode(body);

      if (decoded is Map<String, dynamic>) {
        // pattern A: { wallet: {...} }
        final w = decoded['wallet'];
        if (w is Map<String, dynamic>) {
          return WalletDTO.fromJson(w, fallbackAvatarId: fallbackAvatarId);
        }
        // pattern B: direct wallet object
        return WalletDTO.fromJson(decoded, fallbackAvatarId: fallbackAvatarId);
      }

      _log(
        '[WalletRepositoryHttp] unexpected json type=${decoded.runtimeType}',
      );
      return null;
    } catch (e) {
      _log('[WalletRepositoryHttp] fetch error url=$uri err=$e');
      return null;
    }
  }
}
