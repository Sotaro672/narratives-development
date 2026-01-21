// frontend\mall\lib\features\avatar\presentation\model\me_avatar.dart

class MeAvatar {
  const MeAvatar({
    required this.avatarId,
    required this.walletAddress,
    this.userId,
    this.avatarName,
    this.avatarIcon,
    this.profile,
    this.externalLink,
    this.deletedAt,
  });

  /// ✅ 必須: /mall/me/avatar のサーバ解決結果
  final String avatarId;
  final String walletAddress;

  /// ✅ AvatarPatch 全体（サーバが返す場合のみ埋まる）
  final String? userId;
  final String? avatarName;
  final String? avatarIcon;
  final String? profile;
  final String? externalLink;
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

  factory MeAvatar.fromJson(Map<String, dynamic> j) {
    final avatarId = _s(j['avatarId']);

    final p = j['patch'];
    if (p is! Map) {
      throw const FormatException('MeAvatar: missing "patch" object');
    }
    final patch = Map<String, dynamic>.from(p);

    final walletAddress = _s(patch['walletAddress']);
    if (avatarId.isEmpty) {
      throw const FormatException('MeAvatar: avatarId is required');
    }
    if (walletAddress.isEmpty) {
      throw const FormatException('MeAvatar: patch.walletAddress is required');
    }

    return MeAvatar(
      avatarId: avatarId,
      walletAddress: walletAddress,
      userId: _optString(patch, 'userId'),
      avatarName: _optString(patch, 'avatarName'),
      avatarIcon: _optString(patch, 'avatarIcon'),
      profile: _optString(patch, 'profile'),
      externalLink: _optString(patch, 'externalLink'),
      deletedAt: _optDateTime(patch, 'deletedAt'),
    );
  }

  Map<String, dynamic> toJson() => {
    'avatarId': avatarId,
    'patch': {
      'walletAddress': walletAddress,
      'userId': userId,
      'avatarName': avatarName,
      'avatarIcon': avatarIcon,
      'profile': profile,
      'externalLink': externalLink,
      'deletedAt': deletedAt?.toIso8601String(),
    },
  };
}
