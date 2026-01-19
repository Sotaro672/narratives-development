// frontend/mall/lib/features/avatar/presentation/page/avatar.dart
import 'dart:convert';

import 'package:firebase_auth/firebase_auth.dart';
import 'package:flutter/material.dart';
import 'package:flutter_hooks/flutter_hooks.dart';
import 'package:go_router/go_router.dart';

import '../../../../app/routing/routes.dart';
import '../../../wallet/infrastructure/token_resolve_dto.dart';
import '../../../wallet/presentation/component/token_card.dart';
import '../hook/use_avatar.dart';
import '../model/avatar_vm.dart';

class AvatarPage extends HookWidget {
  const AvatarPage({super.key, this.from});

  /// router.dart から渡される「遷移元」
  final String? from;

  @override
  Widget build(BuildContext context) {
    final vm = useAvatarVm(context, from: from);

    // -------------------------
    // Signed-out view
    // -------------------------
    if (vm.user == null) {
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
                  onPressed: () => context.go(vm.loginUri.toString()),
                  child: const Text('Sign in'),
                ),
              ],
            ),
          ),
        ),
      );
    }

    // -------------------------
    // MeAvatar loading / error -> missing view
    // -------------------------
    if (vm.meAvatarSnap.connectionState == ConnectionState.waiting) {
      return const Center(child: CircularProgressIndicator());
    }

    final me = vm.meAvatarSnap.data;
    if (me == null || me.avatarId.trim().isEmpty) {
      return _MissingMeAvatarView(
        backTo: vm.backTo,
        onGoEdit: () {
          // router.dart と同じルールで from を encode
          final qp = <String, String>{
            AppQueryKey.from: base64UrlEncode(utf8.encode(vm.backTo.trim())),
          };
          final uri = Uri(path: AppRoutePath.avatarEdit, queryParameters: qp);
          context.go(uri.toString());
        },
      );
    }

    // -------------------------
    // Normal profile view
    // -------------------------
    final user = vm.user!;
    final photoUrl = vm.photoUrl;
    final bio = vm.bio;

    return Center(
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 560),
        child: SingleChildScrollView(
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 16),
          child: _AvatarProfileBody(
            user: user,
            photoUrl: photoUrl,
            bio: bio,
            counts: vm.counts,
            tab: vm.tab,
            onTabChange: vm.setTab,
            walletSnap: vm.walletSnap,
            tokens: vm.tokens,
            resolvedTokens: vm.resolvedTokens,
            onEdit: vm.goToAvatarEdit,
          ),
        ),
      ),
    );
  }
}

// ============================================================
// Style-only widgets
// ============================================================

class _AvatarProfileBody extends StatelessWidget {
  const _AvatarProfileBody({
    required this.user,
    required this.photoUrl,
    required this.bio,
    required this.counts,
    required this.tab,
    required this.onTabChange,
    required this.walletSnap,
    required this.tokens,
    required this.resolvedTokens,
    required this.onEdit,
  });

  final User user;
  final String photoUrl;
  final String bio;

  final ProfileCounts counts;
  final ProfileTab tab;
  final void Function(ProfileTab next) onTabChange;

  final AsyncSnapshot walletSnap;
  final List<String> tokens;

  /// mintAddress -> resolved info
  final Map<String, TokenResolveDTO> resolvedTokens;

  final VoidCallback onEdit;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Row(
          crossAxisAlignment: CrossAxisAlignment.start,
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
                      color: Theme.of(context).colorScheme.onSurfaceVariant,
                    )
                  : null,
            ),
            const SizedBox(width: 16),
            Expanded(
              child: Column(
                children: [
                  Row(
                    children: [
                      _StatItem(label: '投稿', value: counts.postCount),
                      _StatItem(label: 'トークン', value: counts.tokenCount),
                    ],
                  ),
                  const SizedBox(height: 8),
                  Row(
                    children: [
                      _StatItem(label: 'フォロー中', value: counts.followingCount),
                      _StatItem(label: 'フォロワー', value: counts.followerCount),
                    ],
                  ),
                ],
              ),
            ),
          ],
        ),
        const SizedBox(height: 10),
        Row(
          crossAxisAlignment: CrossAxisAlignment.center,
          children: [
            const SizedBox(width: 88 + 16),
            Expanded(child: _ProfileBioBox(bio: bio)),
            const SizedBox(width: 10),
            IconButton(
              tooltip: 'Edit Avatar',
              onPressed: onEdit,
              icon: const Icon(Icons.edit_outlined),
            ),
          ],
        ),
        const SizedBox(height: 14),
        _TabBar(tab: tab, onChange: onTabChange),
        const SizedBox(height: 12),
        if (tab == ProfileTab.tokens) ...[
          if (walletSnap.connectionState == ConnectionState.waiting)
            const Padding(
              padding: EdgeInsets.symmetric(vertical: 8),
              child: LinearProgressIndicator(),
            )
          else if (walletSnap.hasError)
            Text(
              'トークンの取得に失敗しました: ${walletSnap.error}',
              style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                color: Theme.of(context).colorScheme.error,
              ),
            )
          else
            _TokenCards(tokens: tokens, resolvedTokens: resolvedTokens),
        ] else ...[
          const _PostsPlaceholder(),
        ],
      ],
    );
  }
}

class _StatItem extends StatelessWidget {
  const _StatItem({required this.label, required this.value});

  final String label;
  final int value;

  @override
  Widget build(BuildContext context) {
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
}

class _ProfileBioBox extends StatelessWidget {
  const _ProfileBioBox({required this.bio});

  final String bio;

  @override
  Widget build(BuildContext context) {
    final cs = Theme.of(context).colorScheme;
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
      decoration: BoxDecoration(
        color: cs.surfaceContainerHighest,
        borderRadius: BorderRadius.circular(12),
      ),
      child: Text(
        bio.isEmpty ? '（プロフィール未設定）' : bio,
        style: Theme.of(
          context,
        ).textTheme.bodyMedium?.copyWith(color: cs.onSurfaceVariant),
      ),
    );
  }
}

class _TabBar extends StatelessWidget {
  const _TabBar({required this.tab, required this.onChange});

  final ProfileTab tab;
  final void Function(ProfileTab next) onChange;

  @override
  Widget build(BuildContext context) {
    final cs = Theme.of(context).colorScheme;

    Widget tabButton({
      required ProfileTab target,
      required IconData icon,
      required String semanticsLabel,
    }) {
      final selected = tab == target;

      return Expanded(
        child: InkWell(
          onTap: () {
            if (tab == target) return;
            onChange(target);
          },
          child: Padding(
            padding: const EdgeInsets.symmetric(vertical: 10),
            child: Icon(
              icon,
              size: 22,
              color: selected ? cs.onSurface : cs.onSurfaceVariant,
              semanticLabel: semanticsLabel,
            ),
          ),
        ),
      );
    }

    return Container(
      decoration: BoxDecoration(
        border: Border(
          top: BorderSide(color: cs.outlineVariant.withValues(alpha: 0.6)),
          bottom: BorderSide(color: cs.outlineVariant.withValues(alpha: 0.6)),
        ),
      ),
      child: Row(
        children: [
          tabButton(
            target: ProfileTab.posts,
            icon: Icons.grid_on,
            semanticsLabel: 'Posts',
          ),
          Container(
            width: 1,
            height: 24,
            color: cs.outlineVariant.withValues(alpha: 0.6),
          ),
          tabButton(
            target: ProfileTab.tokens,
            icon: Icons.local_offer_outlined,
            semanticsLabel: 'Tokens',
          ),
        ],
      ),
    );
  }
}

class _TokenCards extends StatelessWidget {
  const _TokenCards({required this.tokens, required this.resolvedTokens});

  final List<String> tokens;
  final Map<String, TokenResolveDTO> resolvedTokens;

  @override
  Widget build(BuildContext context) {
    if (tokens.isEmpty) {
      return Text(
        'トークンはまだありません。',
        style: Theme.of(context).textTheme.bodyMedium?.copyWith(
          color: Theme.of(context).colorScheme.onSurfaceVariant,
        ),
      );
    }

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        for (final raw in tokens) ...[
          Builder(
            builder: (context) {
              final m = raw.trim();
              final resolved = resolvedTokens[m];
              return TokenCard(mintAddress: m, resolved: resolved);
            },
          ),
          const SizedBox(height: 10),
        ],
      ],
    );
  }
}

class _PostsPlaceholder extends StatelessWidget {
  const _PostsPlaceholder();

  @override
  Widget build(BuildContext context) {
    return Container(
      height: 240,
      alignment: Alignment.center,
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(12),
        color: Theme.of(context).colorScheme.surfaceContainerHighest,
      ),
      child: Text(
        '（次）ここに投稿グリッドを表示します',
        style: Theme.of(context).textTheme.bodyMedium?.copyWith(
          color: Theme.of(context).colorScheme.onSurfaceVariant,
        ),
      ),
    );
  }
}

class _MissingMeAvatarView extends StatelessWidget {
  const _MissingMeAvatarView({required this.backTo, required this.onGoEdit});

  final String backTo;
  final VoidCallback onGoEdit;

  @override
  Widget build(BuildContext context) {
    return Center(
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 560),
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              const Icon(Icons.info_outline, size: 40),
              const SizedBox(height: 12),
              Text(
                'アバター情報を取得できませんでした',
                style: Theme.of(context).textTheme.titleLarge,
                textAlign: TextAlign.center,
              ),
              const SizedBox(height: 8),
              const Text(
                'サインイン状態、または /mall/me/avatar の応答（avatarId・walletAddress）を確認してください。',
                textAlign: TextAlign.center,
              ),
              const SizedBox(height: 16),
              ElevatedButton.icon(
                onPressed: onGoEdit,
                icon: const Icon(Icons.edit_outlined),
                label: const Text('アバター編集へ'),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
