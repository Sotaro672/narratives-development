// frontend\mall\lib\features\avatar\presentation\model\avatar_vm.dart
import 'package:firebase_auth/firebase_auth.dart';
import 'package:flutter/material.dart';

import '../../../wallet/infrastructure/token_metadata_dto.dart';
import '../../../wallet/infrastructure/token_resolve_dto.dart';
import '../../../wallet/infrastructure/wallet_dto.dart';
import 'me_avatar.dart';

class ProfileCounts {
  const ProfileCounts({
    required this.postCount,
    required this.followerCount,
    required this.followingCount,
    required this.tokenCount,
  });

  final int postCount;
  final int followerCount;
  final int followingCount;
  final int tokenCount;
}

enum ProfileTab { posts, tokens }

/// Pattern B:
/// - URL `from` を廃止し、戻り先は NavStore で管理する
/// - よって `backTo` / `loginUri` は ViewModel から削除する
class AvatarVm {
  const AvatarVm({
    required this.authSnap,
    required this.user,
    required this.meAvatarSnap,
    required this.walletSnap,
    required this.tokens,
    required this.resolvedTokens,
    required this.tokenMetadatas,
    required this.isTokensLoading,
    required this.tokenLoadingByMint,
    required this.counts,
    required this.tab,
    required this.setTab,
    required this.avatarIcon,
    required this.profile,
    required this.goToAvatarEdit,
  });

  final AsyncSnapshot<User?> authSnap;
  final User? user;

  final AsyncSnapshot<MeAvatar?> meAvatarSnap;
  final AsyncSnapshot<WalletDTO?> walletSnap;

  /// wallet.tokens (mintAddress list)
  final List<String> tokens;

  /// mintAddress -> resolved info (productId/docId, brandId, metadataUri, etc.)
  /// - resolve 未完了/失敗は entry が無い
  final Map<String, TokenResolveDTO> resolvedTokens;

  /// mintAddress -> token metadata (proxy fetched)
  /// - metadata 未取得/失敗は entry が無い
  final Map<String, TokenMetadataDTO> tokenMetadatas;

  /// ✅ tokens タブ全体としてロード中か（tokens が空の時にも skeleton を出す用途）
  final bool isTokensLoading;

  /// ✅ mintAddress -> loading
  /// - true の間は TokenCard が failure を出さず skeleton を出す
  final Map<String, bool> tokenLoadingByMint;

  final ProfileCounts counts;

  final ProfileTab tab;
  final void Function(ProfileTab next) setTab;

  /// ✅ Backend/Firestore 正規キー: avatarIcon
  /// - 例: "https://..." / "gs://bucket/..." / "avatars/..."
  /// - 未設定なら null
  final String? avatarIcon;

  /// ✅ Backend/Firestore 正規キー: profile
  /// - 未設定なら null
  final String? profile;

  final VoidCallback goToAvatarEdit;
}
