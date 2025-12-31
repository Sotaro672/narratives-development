// frontend/sns/lib/features/avatar/presentation/page/avatar.dart
import 'dart:async';

import 'package:firebase_auth/firebase_auth.dart';
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../../wallet/infrastructure/wallet_repository_http.dart';

class _ProfileCounts {
  const _ProfileCounts({
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

class AvatarPage extends StatefulWidget {
  const AvatarPage({super.key, this.from});

  /// ✅ router.dart から渡される「遷移元」
  final String? from;

  @override
  State<AvatarPage> createState() => _AvatarPageState();
}

class _AvatarPageState extends State<AvatarPage> {
  late final WalletRepositoryHttp _walletRepo;

  Future<WalletDTO?>? _walletFuture;

  String s(String? v) => (v ?? '').trim();

  @override
  void initState() {
    super.initState();
    _walletRepo = WalletRepositoryHttp(
      // apiBase: 'https://narratives-backend-....run.app', // 必要なら注入
      // logger: (m) => debugPrint(m),
    );
  }

  @override
  void dispose() {
    _walletRepo.dispose();
    super.dispose();
  }

  String _displayNameFor(User u) {
    final dn = s(u.displayName);
    if (dn.isNotEmpty) return dn;

    final email = s(u.email);
    if (email.isNotEmpty) return email.split('@').first;

    return 'My Profile';
  }

  String _currentUri(BuildContext context) {
    return GoRouterState.of(context).uri.toString();
  }

  String _effectiveFrom(BuildContext context) {
    final v = s(widget.from);
    if (v.isNotEmpty) return v;
    return _currentUri(context);
  }

  // 暫定: avatarId = Firebase UID（本来は選択中 avatarId を使う）
  String _resolveAvatarId(User user) {
    return s(user.uid);
  }

  Widget _statItem(BuildContext context, String label, int value) {
    return Expanded(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Text(
            value.toString(),
            style: Theme.of(
              context,
            ).textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w700),
          ),
          const SizedBox(height: 2),
          Text(
            label,
            style: Theme.of(context).textTheme.bodySmall?.copyWith(
              color: Theme.of(context).colorScheme.onSurfaceVariant,
            ),
          ),
        ],
      ),
    );
  }

  Widget _tokenChips(BuildContext context, List<String> tokens) {
    if (tokens.isEmpty) {
      return Text(
        'トークンはまだありません。',
        style: Theme.of(context).textTheme.bodyMedium?.copyWith(
          color: Theme.of(context).colorScheme.onSurfaceVariant,
        ),
      );
    }

    String shortMint(String m) {
      final t = m.trim();
      if (t.length <= 12) return t;
      return '${t.substring(0, 6)}…${t.substring(t.length - 4)}';
    }

    return Wrap(
      spacing: 8,
      runSpacing: 8,
      children: tokens.map((m) {
        return Chip(
          label: Text(shortMint(m)),
          visualDensity: VisualDensity.compact,
        );
      }).toList(),
    );
  }

  void _kickoffLoads(User user) {
    final avatarId = _resolveAvatarId(user);
    _walletFuture ??= _walletRepo.fetchByAvatarId(avatarId);
  }

  @override
  Widget build(BuildContext context) {
    return StreamBuilder<User?>(
      stream: FirebaseAuth.instance.userChanges(),
      builder: (context, snap) {
        final user = FirebaseAuth.instance.currentUser ?? snap.data;

        if (user == null) {
          final backTo = _effectiveFrom(context);
          final loginUri = Uri(
            path: '/login',
            queryParameters: {'from': backTo},
          );
          return Center(
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 560),
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    const Icon(Icons.lock_outline, size: 40),
                    const SizedBox(height: 12),
                    Text(
                      'Sign in required',
                      style: Theme.of(context).textTheme.titleLarge,
                      textAlign: TextAlign.center,
                    ),
                    const SizedBox(height: 8),
                    const Text(
                      'プロフィールを表示するにはサインインが必要です。',
                      textAlign: TextAlign.center,
                    ),
                    const SizedBox(height: 16),
                    ElevatedButton(
                      onPressed: () => context.go(loginUri.toString()),
                      child: const Text('Sign in'),
                    ),
                  ],
                ),
              ),
            ),
          );
        }

        _kickoffLoads(user);

        final photoUrl = s(user.photoURL);
        final name = _displayNameFor(user);

        return Center(
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 560),
            child: SingleChildScrollView(
              padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 16),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Row(
                    crossAxisAlignment: CrossAxisAlignment.center,
                    children: [
                      CircleAvatar(
                        radius: 44,
                        backgroundColor: Theme.of(
                          context,
                        ).colorScheme.surfaceContainerHighest,
                        backgroundImage: photoUrl.isNotEmpty
                            ? NetworkImage(photoUrl)
                            : null,
                        child: photoUrl.isEmpty
                            ? Icon(
                                Icons.person,
                                size: 44,
                                color: Theme.of(
                                  context,
                                ).colorScheme.onSurfaceVariant,
                              )
                            : null,
                      ),
                      const SizedBox(width: 16),
                      Expanded(
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text(
                              name,
                              style: Theme.of(context).textTheme.titleLarge,
                              maxLines: 1,
                              overflow: TextOverflow.ellipsis,
                            ),
                            const SizedBox(height: 6),
                            Text(
                              'Profile',
                              style: Theme.of(context).textTheme.bodySmall
                                  ?.copyWith(
                                    color: Theme.of(
                                      context,
                                    ).colorScheme.onSurfaceVariant,
                                  ),
                            ),
                          ],
                        ),
                      ),
                    ],
                  ),
                  const SizedBox(height: 14),

                  FutureBuilder<WalletDTO?>(
                    future: _walletFuture,
                    builder: (context, wsnap) {
                      final wallet = wsnap.data;
                      final tokens = wallet?.tokens ?? const <String>[];

                      // 現状は未接続のため 0（AvatarState 連携時に置換）
                      final postCount = 0;
                      final followerCount = 0;
                      final followingCount = 0;

                      final counts = _ProfileCounts(
                        postCount: postCount,
                        followerCount: followerCount,
                        followingCount: followingCount,
                        tokenCount: tokens.length,
                      );

                      return Row(
                        children: [
                          _statItem(context, '投稿', counts.postCount),
                          _statItem(context, 'フォロワー', counts.followerCount),
                          _statItem(context, 'フォロー中', counts.followingCount),
                          _statItem(context, 'トークン', counts.tokenCount),
                        ],
                      );
                    },
                  ),

                  const SizedBox(height: 12),
                  const Divider(height: 1),
                  const SizedBox(height: 12),

                  Text(
                    'Tokens',
                    style: Theme.of(context).textTheme.titleMedium?.copyWith(
                      fontWeight: FontWeight.w700,
                    ),
                  ),
                  const SizedBox(height: 8),

                  FutureBuilder<WalletDTO?>(
                    future: _walletFuture,
                    builder: (context, wsnap) {
                      if (wsnap.connectionState == ConnectionState.waiting) {
                        return const Padding(
                          padding: EdgeInsets.symmetric(vertical: 8),
                          child: LinearProgressIndicator(),
                        );
                      }
                      if (wsnap.hasError) {
                        return Text(
                          'トークンの取得に失敗しました: ${wsnap.error}',
                          style: Theme.of(context).textTheme.bodyMedium
                              ?.copyWith(
                                color: Theme.of(context).colorScheme.error,
                              ),
                        );
                      }
                      final wallet = wsnap.data;
                      final tokens = wallet?.tokens ?? const <String>[];
                      return _tokenChips(context, tokens);
                    },
                  ),

                  const SizedBox(height: 18),

                  Text(
                    'Posts',
                    style: Theme.of(context).textTheme.titleMedium?.copyWith(
                      fontWeight: FontWeight.w700,
                    ),
                  ),
                  const SizedBox(height: 8),
                  Container(
                    height: 180,
                    alignment: Alignment.center,
                    decoration: BoxDecoration(
                      borderRadius: BorderRadius.circular(12),
                      color: Theme.of(
                        context,
                      ).colorScheme.surfaceContainerHighest,
                    ),
                    child: Text(
                      '（次）ここに投稿グリッドを表示します',
                      style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                        color: Theme.of(context).colorScheme.onSurfaceVariant,
                      ),
                    ),
                  ),
                ],
              ),
            ),
          ),
        );
      },
    );
  }
}
