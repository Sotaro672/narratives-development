// frontend/mall/lib/features/wallet/infrastructure/repository_http.dart

import 'package:flutter/foundation.dart';
import 'package:http/http.dart' as http;
import 'package:firebase_auth/firebase_auth.dart';

import 'api.dart';

class WalletDTO {
  const WalletDTO({
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

  /// ✅ backend が avatarId / walletAddress を返さない可能性があるので、
  ///    呼び出し側で fallback を渡せるようにする
  factory WalletDTO.fromJson(
    Map<String, dynamic> j, {
    String fallbackAvatarId = '',
    String fallbackWalletAddress = '',
  }) {
    dynamic pickAny(List<String> keys) {
      for (final k in keys) {
        if (j.containsKey(k)) return j[k];
      }
      return null;
    }

    // avatarId
    var avatarId = s(
      pickAny(const ['avatarId', 'AvatarID', 'AvatarId', 'id', 'ID']),
    );
    if (avatarId.isEmpty) avatarId = fallbackAvatarId.trim();

    // walletAddress
    var walletAddress = s(
      pickAny(const ['walletAddress', 'WalletAddress', 'address', 'Address']),
    );
    if (walletAddress.isEmpty) {
      walletAddress = fallbackWalletAddress.trim();
    }

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

class WalletRepositoryHttp {
  WalletRepositoryHttp({
    http.Client? client,
    FirebaseAuth? auth,
    String? baseUrl,
    this.logger,
  }) : _api = MallAuthedApi(
         client: client,
         auth: auth,
         baseUrl: baseUrl,
         logger: logger,
       ) {
    _log('[WalletRepositoryHttp] init');
  }

  final MallAuthedApi _api;

  final void Function(String s)? logger;

  static const bool _envHttpLog = bool.fromEnvironment(
    'ENABLE_HTTP_LOG',
    defaultValue: false,
  );

  bool get _logEnabled => kDebugMode || _envHttpLog;

  void dispose() {
    _api.dispose();
  }

  void _log(String s) {
    if (!_logEnabled) return;
    if (logger != null) {
      logger!.call(s);
    } else {
      debugPrint(s);
    }
  }

  // ------------------------------------------------------------
  // API
  // ------------------------------------------------------------

  /// ✅ canonical (new):
  /// GET /mall/me/wallets/{walletAddress}?avatarId={avatarId}
  ///
  /// NOTE:
  /// - walletAddress は呼び出し側が解決して渡す（/mall/me/avatar 等）
  /// - avatarId は URL query で渡す（usecase / handler 側で必要なら利用）
  Future<WalletDTO?> fetchByWalletAddress({
    required String avatarId,
    required String walletAddress,
  }) async {
    final aid = avatarId.trim();
    final addr = walletAddress.trim();
    if (aid.isEmpty || addr.isEmpty) return null;

    final uri = _api.uri('/mall/me/wallets/$addr', <String, String>{
      'avatarId': aid,
    });

    http.Response res;
    try {
      res = await _api.getAuthed(uri);
    } catch (e) {
      _log('[WalletRepositoryHttp] network error url=$uri err=$e');
      return null;
    }

    if (res.statusCode >= 500) {
      // ✅ 500 は “存在しない” ではなく “落ちている”
      throw HttpException(
        statusCode: res.statusCode,
        message: _api.extractError(res.body) ?? 'server_error',
        url: uri.toString(),
        body: res.body,
      );
    }

    if (res.statusCode == 401) {
      _log('[WalletRepositoryHttp] 401 unauthorized url=$uri');
      return null;
    }

    if (res.statusCode == 404) {
      _log('[WalletRepositoryHttp] 404 url=$uri');
      return null;
    }

    if (res.statusCode < 200 || res.statusCode >= 300) {
      _log(
        '[WalletRepositoryHttp] non-2xx status=${res.statusCode} url=$uri bodyLen=${res.body.length}',
      );
      return null;
    }

    final body = res.body.trim();
    if (body.isEmpty) return null;

    final decoded = _api.unwrapData(_api.decodeObject(body));

    // pattern A: { wallet: {...} }
    final w = decoded['wallet'];
    if (w is Map<String, dynamic>) {
      return WalletDTO.fromJson(
        w,
        fallbackAvatarId: aid,
        fallbackWalletAddress: addr,
      );
    }
    if (w is Map) {
      return WalletDTO.fromJson(
        Map<String, dynamic>.from(w),
        fallbackAvatarId: aid,
        fallbackWalletAddress: addr,
      );
    }

    // pattern B: direct wallet object
    return WalletDTO.fromJson(
      decoded,
      fallbackAvatarId: aid,
      fallbackWalletAddress: addr,
    );
  }

  /// Backward-compatible alias:
  /// - 旧コードが fetchMeWallet() などを呼ぶ場合のために残す（必要なければ削除可）
  Future<WalletDTO?> fetchMeWallet({
    required String avatarId,
    required String walletAddress,
  }) {
    return fetchByWalletAddress(
      avatarId: avatarId,
      walletAddress: walletAddress,
    );
  }
}
