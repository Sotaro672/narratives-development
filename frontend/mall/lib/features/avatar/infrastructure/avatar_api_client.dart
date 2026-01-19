// frontend\mall\lib\features\avatar\infrastructure\avatar_api_client.dart
import 'dart:convert';

import 'package:firebase_auth/firebase_auth.dart';
import 'package:http/http.dart' as http;

import '../../../app/config/api_base.dart';
import '../presentation/model/me_avatar.dart';

class AvatarApiClient {
  const AvatarApiClient({http.Client? client}) : _client = client;

  final http.Client? _client;

  http.Client get _http => _client ?? http.Client();

  String _s(String? v) => (v ?? '').trim();

  String _normalizeBase(String base) {
    var b = base.trim();
    while (b.endsWith('/')) {
      b = b.substring(0, b.length - 1);
    }
    return b;
  }

  Uri _uri(String path, [Map<String, String>? query]) {
    final base = _normalizeBase(resolveMallApiBase());
    final p = path.startsWith('/') ? path : '/$path';
    return Uri.parse('$base$p').replace(queryParameters: query);
  }

  Future<String?> _getIdToken({bool forceRefresh = false}) async {
    final u = FirebaseAuth.instance.currentUser;
    if (u == null) return null;
    final t = await u.getIdToken(forceRefresh);
    final token = _s(t?.toString());
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

  String _pickString(Map<String, dynamic> j, List<String> keys) {
    for (final k in keys) {
      if (!j.containsKey(k)) continue;
      final v = _s(j[k]?.toString());
      if (v.isNotEmpty) return v;
    }
    return '';
  }

  Future<MeAvatar?> fetchMeAvatar() async {
    final endpoint = _uri('/mall/me/avatar');

    // 1st attempt (cached token)
    final token1 = await _getIdToken(forceRefresh: false);
    final headers1 = <String, String>{'Accept': 'application/json'};
    if (token1 != null) headers1['Authorization'] = 'Bearer $token1';

    http.Response res;
    try {
      res = await _http.get(endpoint, headers: headers1);
    } catch (_) {
      return null;
    }

    // retry once on 401 (force refresh)
    if (res.statusCode == 401) {
      final token2 = await _getIdToken(forceRefresh: true);
      final headers2 = <String, String>{'Accept': 'application/json'};
      if (token2 != null) headers2['Authorization'] = 'Bearer $token2';

      try {
        res = await _http.get(endpoint, headers: headers2);
      } catch (_) {
        return null;
      }
    }

    if (res.statusCode < 200 || res.statusCode >= 300) return null;

    final decoded = _unwrapData(_decodeObject(res.body));

    // allow multiple shapes: { avatar: {...} } / { me: {...} } / root object
    Map<String, dynamic> root = decoded;
    final avatarObj = decoded['avatar'];
    if (avatarObj is Map) root = Map<String, dynamic>.from(avatarObj);
    final meObj = decoded['me'];
    if (meObj is Map) root = Map<String, dynamic>.from(meObj);

    final avatarId = _pickString(root, const [
      'avatarId',
      'AvatarID',
      'AvatarId',
      'id',
      'ID',
    ]);
    final walletAddress = _pickString(root, const [
      'walletAddress',
      'WalletAddress',
      'address',
      'Address',
    ]);

    if (avatarId.isEmpty) return null;
    // 現状方針に合わせて walletAddress も必須
    if (walletAddress.isEmpty) return null;

    return MeAvatar(avatarId: avatarId, walletAddress: walletAddress);
  }
}
