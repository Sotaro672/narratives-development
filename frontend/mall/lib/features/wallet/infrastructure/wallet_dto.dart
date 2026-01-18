class WalletDTO {
  const WalletDTO({
    required this.avatarId,
    required this.walletAddress,
    required this.tokens,
  });

  final String avatarId;
  final String walletAddress;
  final List<String> tokens;

  static String _pickString(Map<String, dynamic> j, List<String> keys) {
    for (final k in keys) {
      if (!j.containsKey(k)) continue;
      final v = (j[k] ?? '').toString().trim();
      if (v.isNotEmpty) return v;
    }
    return '';
  }

  static List<String> _pickStringList(
    Map<String, dynamic> j,
    List<String> keys,
  ) {
    for (final k in keys) {
      if (!j.containsKey(k)) continue;
      final v = j[k];

      if (v is List) {
        // ["mint1","mint2"]
        final out = <String>[];
        for (final e in v) {
          final s = (e ?? '').toString().trim();
          if (s.isNotEmpty) out.add(s);
        }
        return out;
      }
    }
    return const <String>[];
  }

  factory WalletDTO.fromJson(Map<String, dynamic> j) {
    final avatarId = _pickString(j, const [
      'avatarId',
      'AvatarID',
      'AvatarId',
      'id',
      'ID',
    ]);
    final walletAddress = _pickString(j, const [
      'walletAddress',
      'WalletAddress',
      'address',
      'Address',
      'owner',
      'Owner',
    ]);
    final tokens = _pickStringList(j, const [
      'tokens',
      'Tokens',
      'tokenMints',
      'TokenMints',
      'mints',
      'Mints',
    ]);

    return WalletDTO(
      avatarId: avatarId,
      walletAddress: walletAddress,
      tokens: tokens,
    );
  }
}
