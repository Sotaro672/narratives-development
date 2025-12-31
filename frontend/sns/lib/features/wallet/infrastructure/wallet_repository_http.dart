// frontend/sns/lib/features/wallet/infrastructure/wallet_repository_http.dart
import 'dart:convert';

import 'package:firebase_auth/firebase_auth.dart';
import 'package:http/http.dart' as http;

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
  final String lastUpdatedAt; // RFC3339 string (or empty)
  final String status;

  static String s(dynamic v) => (v ?? '').toString().trim();

  static List<String> _tokensFrom(dynamic v) {
    if (v is List) {
      return v.map((e) => s(e)).where((x) => x.isNotEmpty).toList();
    }
    return <String>[];
  }

  factory WalletDTO.fromJson(Map<String, dynamic> j) {
    return WalletDTO(
      avatarId: s(j['avatarId']),
      walletAddress: s(j['walletAddress']),
      tokens: _tokensFrom(j['tokens']),
      lastUpdatedAt: s(j['lastUpdatedAt']),
      status: s(j['status']).isEmpty ? 'active' : s(j['status']),
    );
  }
}

class WalletRepositoryHttp {
  WalletRepositoryHttp({
    http.Client? client,
    FirebaseAuth? auth,
    this.apiBase,
    this.logger,
  }) : _client = client ?? http.Client(),
       _auth = auth ?? FirebaseAuth.instance;

  final http.Client _client;
  final FirebaseAuth _auth;

  /// 例: https://narratives-backend-...run.app
  /// ここを渡さない場合は環境変数などの方式に合わせて差し替えてください。
  final String? apiBase;

  final void Function(String s)? logger;

  void dispose() {
    _client.close();
  }

  void _log(String s) => logger?.call(s);

  String _s(String? v) => (v ?? '').trim();

  Uri _u(String path, [Map<String, String>? qp]) {
    final base = _s(apiBase);
    if (base.isEmpty) {
      // apiBase が未設定なら相対パスとして扱う（同一オリジン想定）
      return Uri(path: path, queryParameters: qp);
    }
    return Uri.parse(base).replace(path: path, queryParameters: qp);
  }

  Future<Map<String, String>> _authHeaders() async {
    final user = _auth.currentUser;
    if (user == null) return {};
    final token = await user.getIdToken();
    return {'Authorization': 'Bearer $token'};
  }

  /// avatarId から Wallet を取得（トークン一覧表示用）
  ///
  /// ✅ エンドポイント未確定のため、候補を順に試します。
  /// - GET /sns/wallet?avatarId=...
  /// - GET /sns/wallets?avatarId=...
  /// - GET /sns/wallets/{avatarId}
  ///
  /// 成功条件: 2xx + JSON
  Future<WalletDTO?> fetchByAvatarId(String avatarId) async {
    final aid = _s(avatarId);
    if (aid.isEmpty) return null;

    final headers = {'Accept': 'application/json', ...await _authHeaders()};

    final candidates = <Uri>[
      _u('/sns/wallet', {'avatarId': aid}),
      _u('/sns/wallets', {'avatarId': aid}),
      _u('/sns/wallets/$aid'),
    ];

    for (final uri in candidates) {
      try {
        _log('[WalletRepositoryHttp] GET $uri');
        final res = await _client.get(uri, headers: headers);

        if (res.statusCode < 200 || res.statusCode >= 300) {
          _log(
            '[WalletRepositoryHttp] non-2xx status=${res.statusCode} url=$uri',
          );
          continue;
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
      } catch (e) {
        _log('[WalletRepositoryHttp] fetch error url=$uri err=$e');
        continue;
      }
    }

    return null;
  }
}
