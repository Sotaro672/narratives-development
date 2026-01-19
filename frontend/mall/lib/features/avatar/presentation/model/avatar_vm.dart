// frontend/mall/lib/features/avatar/presentation/model/avatar_vm.dart
import 'package:firebase_auth/firebase_auth.dart';
import 'package:flutter/material.dart';

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

class AvatarVm {
  const AvatarVm({
    required this.authSnap,
    required this.user,
    required this.meAvatarSnap,
    required this.walletSnap,
    required this.tokens,
    required this.counts,
    required this.tab,
    required this.setTab,
    required this.backTo,
    required this.loginUri,
    required this.photoUrl,
    required this.bio,
    required this.goToAvatarEdit,
  });

  final AsyncSnapshot<User?> authSnap;
  final User? user;

  final AsyncSnapshot<MeAvatar?> meAvatarSnap;
  final AsyncSnapshot<WalletDTO?> walletSnap;

  final List<String> tokens;
  final ProfileCounts counts;

  final ProfileTab tab;
  final void Function(ProfileTab next) setTab;

  final String backTo;
  final Uri loginUri;

  final String photoUrl;
  final String bio;

  final VoidCallback goToAvatarEdit;
}
