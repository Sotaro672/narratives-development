// frontend/mall/lib/features/wallet/infrastructure/repository_http.dart
import 'dart:convert';

import 'package:firebase_auth/firebase_auth.dart';
import 'package:http/http.dart' as http;

import '../../../app/config/api_base.dart';
import 'token_resolve_dto.dart';
import 'wallet_dto.dart';

class WalletRepositoryHttp {
  WalletRepositoryHttp();

  void dispose() {}

  String _normalizeBase(String base) {
    var b = base.trim();
    while (b.endsWith('/')) {
      b = b.substring(0, b.length - 1);
    }
    return b;
  }

  Uri _uri(String path) {
    final base = _normalizeBase(resolveMallApiBase());
    final p = path.startsWith('/') ? path : '/$path';
    return Uri.parse('$base$p');
  }

  Uri _uriWithQuery(String path, Map<String, String> queryParameters) {
    final base = _normalizeBase(resolveMallApiBase());
    final p = path.startsWith('/') ? path : '/$path';
    final u = Uri.parse('$base$p');
    return u.replace(queryParameters: queryParameters);
  }

  Future<String?> _getIdToken({bool forceRefresh = false}) async {
    final u = FirebaseAuth.instance.currentUser;
    if (u == null) return null;
    final t = await u.getIdToken(forceRefresh);
    final token = (t ?? '').toString().trim();
    return token.isEmpty ? null : token;
  }

  Map<String, dynamic> _decodeObject(String body) {
    final b = body.trim();
    if (b.isEmpty) throw const FormatException('Empty response body');
    final decoded = jsonDecode(b);
    if (decoded is Map<String, dynamic>) return decoded;
    if (decoded is Map) return Map<String, dynamic>.from(decoded);
    throw const FormatException('Invalid JSON shape (expected object)');
  }

  Map<String, dynamic> _unwrapData(Map<String, dynamic> decoded) {
    final data = decoded['data'];
    if (data is Map<String, dynamic>) return data;
    if (data is Map) return Map<String, dynamic>.from(data);
    return decoded;
  }

  WalletDTO? _extractWallet(Map<String, dynamic> decoded) {
    // 1) {"wallets":[{...}]}
    final wallets = decoded['wallets'];
    if (wallets is List && wallets.isNotEmpty) {
      final first = wallets.first;
      if (first is Map<String, dynamic>) {
        return WalletDTO.fromJson(first);
      }
      if (first is Map) {
        return WalletDTO.fromJson(Map<String, dynamic>.from(first));
      }
    }

    // 2) {"wallet":{...}}
    final w = decoded['wallet'];
    if (w is Map<String, dynamic>) {
      return WalletDTO.fromJson(w);
    }
    if (w is Map) {
      return WalletDTO.fromJson(Map<String, dynamic>.from(w));
    }

    // 3) 直下が wallet オブジェクト
    // （avatarId or walletAddress or tokens があれば wallet とみなす）
    final hasAnyKey =
        decoded.containsKey('walletAddress') ||
        decoded.containsKey('WalletAddress') ||
        decoded.containsKey('tokens') ||
        decoded.containsKey('tokenMints');

    if (hasAnyKey) {
      return WalletDTO.fromJson(decoded);
    }

    return null;
  }

  Future<WalletDTO?> fetchMeWallet() async {
    final uri = _uri('/mall/me/wallets');

    // 1st try
    final token1 = await _getIdToken(forceRefresh: false);
    final headers1 = <String, String>{'Accept': 'application/json'};
    if (token1 != null) {
      headers1['Authorization'] = 'Bearer $token1';
    }

    http.Response res = await http.get(uri, headers: headers1);

    // retry 401
    if (res.statusCode == 401) {
      final token2 = await _getIdToken(forceRefresh: true);
      final headers2 = <String, String>{'Accept': 'application/json'};
      if (token2 != null) {
        headers2['Authorization'] = 'Bearer $token2';
      }
      res = await http.get(uri, headers: headers2);
    }

    if (res.statusCode < 200 || res.statusCode >= 300) return null;

    final decoded = _unwrapData(_decodeObject(res.body));
    return _extractWallet(decoded);
  }

  Future<void> syncMeWallet() async {
    final uri = _uri('/mall/me/wallets/sync');

    final token1 = await _getIdToken(forceRefresh: false);
    final headers1 = <String, String>{
      'Accept': 'application/json',
      'Content-Type': 'application/json',
    };
    if (token1 != null) {
      headers1['Authorization'] = 'Bearer $token1';
    }

    http.Response res = await http.post(
      uri,
      headers: headers1,
      body: jsonEncode({}),
    );

    if (res.statusCode == 401) {
      final token2 = await _getIdToken(forceRefresh: true);
      final headers2 = <String, String>{
        'Accept': 'application/json',
        'Content-Type': 'application/json',
      };
      if (token2 != null) {
        headers2['Authorization'] = 'Bearer $token2';
      }
      res = await http.post(uri, headers: headers2, body: jsonEncode({}));
    }

    // 失敗時は呼び出し側で再取得してエラー表示したいので例外にする
    if (res.statusCode < 200 || res.statusCode >= 300) {
      throw Exception('sync failed: ${res.statusCode} ${res.body}');
    }
  }

  Future<WalletDTO?> syncAndFetchMeWallet() async {
    try {
      await syncMeWallet();
    } catch (_) {
      // sync が失敗しても、現状の wallet を表示したいので握りつぶす
    }
    return fetchMeWallet();
  }

  /// Resolve token information from a mint address.
  ///
  /// Backend:
  /// - GET /mall/me/wallets/tokens/resolve?mintAddress=...
  ///
  /// Response (expected):
  /// {
  ///   "productId": "...",     // docId in tokens collection
  ///   "brandId": "...",
  ///   "metadataUri": "...",
  ///   "mintAddress": "..."
  /// }
  Future<TokenResolveDTO?> resolveTokenByMintAddress(String mintAddress) async {
    final m = mintAddress.trim();
    if (m.isEmpty) return null;

    final uri = _uriWithQuery('/mall/me/wallets/tokens/resolve', {
      'mintAddress': m,
    });

    // 1st try
    final token1 = await _getIdToken(forceRefresh: false);
    final headers1 = <String, String>{'Accept': 'application/json'};
    if (token1 != null) {
      headers1['Authorization'] = 'Bearer $token1';
    }

    http.Response res = await http.get(uri, headers: headers1);

    // retry 401
    if (res.statusCode == 401) {
      final token2 = await _getIdToken(forceRefresh: true);
      final headers2 = <String, String>{'Accept': 'application/json'};
      if (token2 != null) {
        headers2['Authorization'] = 'Bearer $token2';
      }
      res = await http.get(uri, headers: headers2);
    }

    if (res.statusCode < 200 || res.statusCode >= 300) {
      return null;
    }

    final decoded = _unwrapData(_decodeObject(res.body));
    return TokenResolveDTO.fromJson(decoded);
  }
}
