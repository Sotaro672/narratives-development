// frontend\mall\lib\features\avatar\presentation\model\me_avatar.dart

class MeAvatar {
  const MeAvatar({
    required this.avatarId,
    this.walletAddress,
    this.userId,
    this.avatarName,
    this.avatarIcon,
    this.profile,
    this.externalLink,
    this.createdAt,
    this.updatedAt,
    this.deletedAt,
  });

  /// ✅ 必須: /mall/me/avatars のサーバ解決結果
  /// backend のレスポンスは原則 "avatarId" を返す想定
  final String avatarId;

  /// ✅ /mall/me/avatars のレスポンス形状が
  /// A) { avatarId, patch: {...} } でも
  /// B) { avatarId, walletAddress, ... } でも
  /// どちらでも読めるように optional にする
  final String? walletAddress;

  /// ✅ AvatarPatch 全体（サーバが返す場合のみ埋まる）
  final String? userId;
  final String? avatarName;
  final String? avatarIcon;
  final String? profile;
  final String? externalLink;

  final DateTime? createdAt;
  final DateTime? updatedAt;
  final DateTime? deletedAt;

  static String _s(Object? v) => (v ?? '').toString().trim();

  static String? _optString(Map<String, dynamic> j, String key) {
    if (!j.containsKey(key)) return null;
    final v = _s(j[key]);
    return v.isEmpty ? null : v;
  }

  static DateTime? _optDateTime(Map<String, dynamic> j, String key) {
    final s = _optString(j, key);
    if (s == null) return null;
    try {
      return DateTime.parse(s);
    } catch (_) {
      return null;
    }
  }

  /// patch 形状に寄せるための helper
  static Map<String, dynamic>? _optPatch(Map<String, dynamic> j) {
    final p = j['patch'];

    // patch が Map ならそれを使う
    if (p is Map) {
      return Map<String, dynamic>.from(p);
    }

    // patch が無い（=フラット）なら null
    return null;
  }

  factory MeAvatar.fromJson(Map<String, dynamic> j) {
    final avatarId = _s(j['avatarId']);
    if (avatarId.isEmpty) {
      throw const FormatException('MeAvatar: avatarId is required');
    }

    // ------------------------------------------------------------
    // ✅ 互換:
    // A) { avatarId, patch: {...} }
    // B) { avatarId, walletAddress, userId, avatarName, ... } (flat)
    // ------------------------------------------------------------
    final patch = _optPatch(j);

    // A) patch あり
    if (patch != null) {
      // walletAddress は patch にある想定だが、将来 absent でも落とさない
      final wa = _optString(patch, 'walletAddress');

      return MeAvatar(
        avatarId: avatarId,
        walletAddress: wa,
        userId: _optString(patch, 'userId'),
        avatarName: _optString(patch, 'avatarName'),
        avatarIcon: _optString(patch, 'avatarIcon'),
        profile: _optString(patch, 'profile'),
        externalLink: _optString(patch, 'externalLink'),
        createdAt: _optDateTime(patch, 'createdAt'),
        updatedAt: _optDateTime(patch, 'updatedAt'),
        deletedAt: _optDateTime(patch, 'deletedAt'),
      );
    }

    // B) flat
    return MeAvatar(
      avatarId: avatarId,
      walletAddress: _optString(j, 'walletAddress'),
      userId: _optString(j, 'userId'),
      avatarName: _optString(j, 'avatarName'),
      avatarIcon: _optString(j, 'avatarIcon'),
      profile: _optString(j, 'profile'),
      externalLink: _optString(j, 'externalLink'),
      createdAt: _optDateTime(j, 'createdAt'),
      updatedAt: _optDateTime(j, 'updatedAt'),
      deletedAt: _optDateTime(j, 'deletedAt'),
    );
  }

  /// ✅ できるだけ patch 形状で出す（既存コード互換）
  /// - walletAddress が null の場合でも patch 自体は出す（キーは入れない）
  Map<String, dynamic> toJson() {
    final patch = <String, dynamic>{};

    final wa = (walletAddress ?? '').trim();
    if (wa.isNotEmpty) patch['walletAddress'] = wa;

    if ((userId ?? '').trim().isNotEmpty) patch['userId'] = userId!.trim();
    if ((avatarName ?? '').trim().isNotEmpty) {
      patch['avatarName'] = avatarName!.trim();
    }
    if ((avatarIcon ?? '').trim().isNotEmpty) {
      patch['avatarIcon'] = avatarIcon!.trim();
    }
    if ((profile ?? '').trim().isNotEmpty) patch['profile'] = profile!.trim();
    if ((externalLink ?? '').trim().isNotEmpty) {
      patch['externalLink'] = externalLink!.trim();
    }

    if (createdAt != null) patch['createdAt'] = createdAt!.toIso8601String();
    if (updatedAt != null) patch['updatedAt'] = updatedAt!.toIso8601String();
    if (deletedAt != null) patch['deletedAt'] = deletedAt!.toIso8601String();

    return {'avatarId': avatarId, 'patch': patch};
  }
}
