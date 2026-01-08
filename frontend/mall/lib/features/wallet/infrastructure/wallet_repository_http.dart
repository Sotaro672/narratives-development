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

  factory WalletDTO.fromJson(Map<String, dynamic> j) {
    // 名揺れ吸収を少しだけ（backend 側が変わっても落ちないように）
    dynamic pickAny(List<String> keys) {
      for (final k in keys) {
        if (j.containsKey(k)) return j[k];
      }
      return null;
    }

    final avatarId = s(pickAny(const ['avatarId', 'AvatarID', 'AvatarId']));
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
             : resolveSnsApiBase(),
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

  // ✅ release でもログを出したい場合: --dart-define=ENABLE_HTTP_LOG=true
  static const bool _envHttpLog = bool.fromEnvironment(
    'ENABLE_HTTP_LOG',
    defaultValue: false,
  );

  bool get _logEnabled => kDebugMode || _envHttpLog;

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

  Future<Map<String, String>> _authHeaders() async {
    final headers = <String, String>{};

    try {
      final user = _auth.currentUser;
      if (user != null) {
        // まずは通常トークン
        final token = await user.getIdToken(false);
        final t = (token ?? '').toString().trim();
        if (t.isNotEmpty) headers['Authorization'] = 'Bearer $t';
      }
    } catch (e) {
      _log('[WalletRepositoryHttp] token error: $e');
    }

    return headers;
  }

  Future<Map<String, String>> _authHeadersForceRefresh() async {
    final headers = <String, String>{};

    try {
      final user = _auth.currentUser;
      if (user != null) {
        final token = await user.getIdToken(true);
        final t = (token ?? '').toString().trim();
        if (t.isNotEmpty) headers['Authorization'] = 'Bearer $t';
      }
    } catch (e) {
      _log('[WalletRepositoryHttp] token refresh error: $e');
    }

    return headers;
  }

  // ------------------------------------------------------------
  // API
  // ------------------------------------------------------------

  /// avatarId から Wallet を取得（トークン一覧表示用）
  ///
  /// ✅ エンドポイント未確定のため、候補を順に試します。
  /// - GET /mall/wallet?avatarId=...
  /// - GET /mall/wallets?avatarId=...
  /// - GET /mall/wallets/{avatarId}
  ///
  /// 成功条件: 2xx + JSON
  Future<WalletDTO?> fetchByAvatarId(String avatarId) async {
    final aid = (avatarId).trim();
    if (aid.isEmpty) return null;

    final baseHeaders = <String, String>{
      'Accept': 'application/json',
      ...await _authHeaders(),
    };

    final candidates = <Uri>[
      _uri('/mall/wallet', {'avatarId': aid}),
      _uri('/mall/wallets', {'avatarId': aid}),
      _uri('/mall/wallets/$aid'),
    ];

    for (final uri in candidates) {
      // 1st try
      final dto = await _tryGetWallet(uri, headers: baseHeaders);
      if (dto != null) return dto;

      // If 401, retry once with refreshed token
      // （_tryGetWallet 内部で 401 を判定して null を返す）
      if (_lastStatusCode == 401) {
        final retryHeaders = <String, String>{
          'Accept': 'application/json',
          ...await _authHeadersForceRefresh(),
        };
        final dto2 = await _tryGetWallet(uri, headers: retryHeaders);
        if (dto2 != null) return dto2;
      }
    }

    return null;
  }

  int? _lastStatusCode;

  Future<WalletDTO?> _tryGetWallet(
    Uri uri, {
    required Map<String, String> headers,
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

      // パターンA: { wallet: {...} }
      if (decoded is Map<String, dynamic>) {
        final w = decoded['wallet'];
        if (w is Map<String, dynamic>) {
          return WalletDTO.fromJson(w);
        }
        // パターンB: 直で { avatarId, walletAddress, tokens... }
        return WalletDTO.fromJson(decoded);
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

  // （必要になったら）将来、確定エンドポイント向けの strict 実装を追加できます
  // Future<WalletDTO> getByAvatarIdStrict(...) ...
}
