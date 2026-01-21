// frontend\mall\lib\features\avatar\infrastructure\avatar_api_client.dart
import 'dart:convert';

import 'package:http/http.dart' as http;

import '../presentation/model/me_avatar.dart';
import 'api.dart';

/// ✅ Edit 画面プレフィル用 DTO
class MeAvatarProfileDto {
  const MeAvatarProfileDto({
    required this.avatarId,
    required this.name,
    required this.profile,
    required this.link,
    required this.iconUrl,
  });

  final String avatarId;
  final String name;
  final String profile;
  final String link;
  final String iconUrl;

  static String _s(Object? v) => (v ?? '').toString().trim();

  factory MeAvatarProfileDto.fromJson(Map<String, dynamic> j) {
    // server のキー揺れも吸収
    String pick(List<String> keys) {
      for (final k in keys) {
        if (!j.containsKey(k)) continue;
        final v = _s(j[k]);
        if (v.isNotEmpty) return v;
      }
      return '';
    }

    return MeAvatarProfileDto(
      avatarId: pick(const ['avatarId', 'AvatarID', 'AvatarId', 'id', 'ID']),
      name: pick(const ['name', 'displayName', 'avatarName', 'AvatarName']),
      profile: pick(const ['profile', 'bio', 'description', 'Profile', 'Bio']),
      link: pick(const ['link', 'url', 'externalLink', 'Link', 'URL']),
      iconUrl: pick(const ['iconUrl', 'iconURL', 'avatarIconUrl', 'photoUrl']),
    );
  }
}

class AvatarApiClient {
  AvatarApiClient({http.Client? client}) : _api = MallAuthedApi(client: client);

  final MallAuthedApi _api;

  String _s(Object? v) => (v ?? '').toString().trim();

  Map<String, dynamic> _decodeObject(String body) {
    final b = body.trim();
    if (b.isEmpty) throw const FormatException('Empty response body');
    final decoded = jsonDecode(b);
    if (decoded is Map<String, dynamic>) return decoded;
    if (decoded is Map) return Map<String, dynamic>.from(decoded);
    throw const FormatException('Invalid JSON shape (expected object)');
  }

  Map<String, dynamic> _unwrapData(Map<String, dynamic> decoded) {
    return _api.unwrapData(decoded);
  }

  String _pickString(Map<String, dynamic> j, List<String> keys) {
    for (final k in keys) {
      if (!j.containsKey(k)) continue;
      final v = _s(j[k]);
      if (v.isNotEmpty) return v;
    }
    return '';
  }

  Future<Map<String, dynamic>?> _getAuthedJson(String path) async {
    final uri = _api.uri(path);
    http.Response res;
    try {
      res = await _api.sendAuthed('GET', uri, jsonBody: null);
    } catch (_) {
      return null;
    }

    if (res.statusCode == 404) return null;
    if (res.statusCode < 200 || res.statusCode >= 300) return null;

    try {
      final decoded = _decodeObject(res.body);
      return _unwrapData(decoded);
    } catch (_) {
      return null;
    }
  }

  /// ✅ 既存：me avatar（avatarId + walletAddress）
  Future<MeAvatar?> fetchMeAvatar() async {
    final decoded = await _getAuthedJson('/mall/me/avatar');
    if (decoded == null) return null;

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

  /// ✅ NEW：edit プレフィル用（name/profile/link/iconUrl を取りたい）
  ///
  /// 前提:
  /// - まず /mall/me/avatar を叩き、レスポンス内に profile 情報があればそれを読む。
  /// - profile 情報が無ければ、同じデータソースで “追加フィールド” が返るように
  ///   backend を拡張するのが理想（このメソッドはそのまま使えます）。
  ///
  /// NOTE:
  /// - APIが avatarId しか返さない場合でも、このメソッドは avatarId だけ入った DTO を返すので、
  ///   VM側で「空なら上書きしない」実装と相性が良いです。
  Future<MeAvatarProfileDto?> fetchMyAvatarProfile() async {
    final decoded = await _getAuthedJson('/mall/me/avatar');
    if (decoded == null) return null;

    // allow multiple shapes: { avatar: {...} } / { me: {...} } / root object
    Map<String, dynamic> root = decoded;
    final avatarObj = decoded['avatar'];
    if (avatarObj is Map) root = Map<String, dynamic>.from(avatarObj);
    final meObj = decoded['me'];
    if (meObj is Map) root = Map<String, dynamic>.from(meObj);

    final dto = MeAvatarProfileDto.fromJson(root);
    if (dto.avatarId.trim().isEmpty) return null;

    return dto;
  }

  void dispose() {
    _api.dispose();
  }
}
