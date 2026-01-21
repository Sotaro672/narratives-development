// frontend\mall\lib\features\avatar\presentation\page\avatar.dart
import 'package:firebase_auth/firebase_auth.dart';
import 'package:flutter/material.dart';
import 'package:flutter_hooks/flutter_hooks.dart';
import 'package:go_router/go_router.dart';

import '../../../../app/routing/navigation.dart';
import '../../../../app/routing/routes.dart';
import '../../../../app/shell/presentation/components/app_fixed_extent_grid.dart';
import '../../../wallet/infrastructure/token_metadata_dto.dart';
import '../../../wallet/infrastructure/token_resolve_dto.dart';
import '../../../wallet/presentation/component/token_card.dart';
import '../hook/use_avatar.dart';
import '../model/avatar_vm.dart';

class AvatarPage extends HookWidget {
  const AvatarPage({super.key});

  @override
  Widget build(BuildContext context) {
    // ✅ Pattern B: `from` は URL で受け取らない
    final vm = useAvatarVm(context);

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
                  onPressed: () {
                    // ✅ Pattern B: 戻り先を Store に保存して login へ
                    NavStore.I.setReturnTo(AppRoutePath.avatar);
                    context.go(AppRoutePath.login);
                  },
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

    // ✅ Missing 判定の前に error を表示できるようにする
    if (vm.meAvatarSnap.hasError) {
      return Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 560),
          child: Padding(
            padding: const EdgeInsets.all(16),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                const Icon(Icons.error_outline, size: 40),
                const SizedBox(height: 12),
                Text(
                  'アバター情報の取得でエラーが発生しました',
                  style: Theme.of(context).textTheme.titleLarge,
                  textAlign: TextAlign.center,
                ),
                const SizedBox(height: 8),
                Text(
                  '${vm.meAvatarSnap.error}',
                  textAlign: TextAlign.center,
                  style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                    color: Theme.of(context).colorScheme.error,
                  ),
                ),
                const SizedBox(height: 16),
                ElevatedButton.icon(
                  onPressed: () {
                    NavStore.I.setReturnTo(AppRoutePath.avatar);
                    context.go(AppRoutePath.avatarEdit);
                  },
                  icon: const Icon(Icons.edit_outlined),
                  label: const Text('アバター編集へ'),
                ),
              ],
            ),
          ),
        ),
      );
    }

    final me = vm.meAvatarSnap.data;
    if (me == null || me.avatarId.trim().isEmpty) {
      return _MissingMeAvatarView(
        backTo: AppRoutePath.avatar,
        onGoEdit: () {
          NavStore.I.setReturnTo(AppRoutePath.avatar);
          context.go(AppRoutePath.avatarEdit);
        },
      );
    }

    // -------------------------
    // Normal profile view
    // -------------------------
    final user = vm.user!;

    // ✅ 取得元を MeAvatar(=patch) に寄せる（互換なし）
    final icon = (me.avatarIcon ?? '').trim();
    final prof = (me.profile ?? '').trim();
    final link = (me.externalLink ?? '').trim();

    return Center(
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 560),
        child: SingleChildScrollView(
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 16),
          child: _AvatarProfileBody(
            user: user,
            avatarIcon: icon.isEmpty ? null : icon,
            profile: prof.isEmpty ? null : prof,
            externalLink: link.isEmpty ? null : link,
            counts: vm.counts,
            tab: vm.tab,
            onTabChange: vm.setTab,
            walletSnap: vm.walletSnap,
            tokens: vm.tokens,
            resolvedTokens: vm.resolvedTokens,
            tokenMetadatas: vm.tokenMetadatas,
            isTokensLoading: vm.isTokensLoading,
            tokenLoadingByMint: vm.tokenLoadingByMint,
            onEdit: () {
              NavStore.I.setReturnTo(AppRoutePath.avatar);
              vm.goToAvatarEdit();
            },
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
    required this.avatarIcon,
    required this.profile,
    required this.externalLink,
    required this.counts,
    required this.tab,
    required this.onTabChange,
    required this.walletSnap,
    required this.tokens,
    required this.resolvedTokens,
    required this.tokenMetadatas,
    required this.isTokensLoading,
    required this.tokenLoadingByMint,
    required this.onEdit,
  });

  final User user;

  /// ✅ Backend/Firestore 正規キー: avatarIcon（nullable）
  final String? avatarIcon;

  /// ✅ Backend/Firestore 正規キー: profile（nullable）
  final String? profile;

  /// ✅ Backend/Firestore 正規キー: externalLink（nullable）
  final String? externalLink;

  final ProfileCounts counts;
  final ProfileTab tab;
  final void Function(ProfileTab next) onTabChange;

  final AsyncSnapshot walletSnap;
  final List<String> tokens;

  final Map<String, TokenResolveDTO> resolvedTokens;
  final Map<String, TokenMetadataDTO> tokenMetadatas;

  final bool isTokensLoading;
  final Map<String, bool> tokenLoadingByMint;

  final VoidCallback onEdit;

  @override
  Widget build(BuildContext context) {
    final icon = (avatarIcon ?? '').trim();
    final prof = (profile ?? '').trim();
    final link = (externalLink ?? '').trim();

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
              backgroundImage: icon.isNotEmpty ? NetworkImage(icon) : null,
              child: icon.isEmpty
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

        // ✅ profile / externalLink を表示
        const SizedBox(height: 10),
        Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const SizedBox(width: 88 + 16),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  _ProfileBox(profile: prof),
                  const SizedBox(height: 8),
                  _ExternalLinkBox(url: link),
                ],
              ),
            ),
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
          if (walletSnap.hasError)
            Text(
              'トークンの取得に失敗しました: ${walletSnap.error}',
              style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                color: Theme.of(context).colorScheme.error,
              ),
            )
          else
            _TokenCards(
              tokens: tokens,
              resolvedTokens: resolvedTokens,
              tokenMetadatas: tokenMetadatas,
              isTokensLoading: isTokensLoading,
              tokenLoadingByMint: tokenLoadingByMint,
            ),
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

class _ProfileBox extends StatelessWidget {
  const _ProfileBox({required this.profile});

  final String profile;

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
        profile.isEmpty ? '（プロフィール未設定）' : profile,
        style: Theme.of(
          context,
        ).textTheme.bodyMedium?.copyWith(color: cs.onSurfaceVariant),
      ),
    );
  }
}

class _ExternalLinkBox extends StatelessWidget {
  const _ExternalLinkBox({required this.url});

  final String url;

  @override
  Widget build(BuildContext context) {
    final cs = Theme.of(context).colorScheme;

    if (url.trim().isEmpty) {
      return Container(
        width: double.infinity,
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
        decoration: BoxDecoration(
          color: cs.surfaceContainerHighest,
          borderRadius: BorderRadius.circular(12),
        ),
        child: Text(
          '（外部リンク未設定）',
          style: Theme.of(
            context,
          ).textTheme.bodyMedium?.copyWith(color: cs.onSurfaceVariant),
        ),
      );
    }

    return InkWell(
      onTap: () {
        // ✅ 本格的なリンク起動は url_launcher 等に寄せる想定。
        // ここでは「表示されること」を優先して UI に出す。
      },
      child: Container(
        width: double.infinity,
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
        decoration: BoxDecoration(
          color: cs.surfaceContainerHighest,
          borderRadius: BorderRadius.circular(12),
        ),
        child: Row(
          children: [
            Icon(Icons.link, size: 18, color: cs.onSurfaceVariant),
            const SizedBox(width: 8),
            Expanded(
              child: Text(
                url.trim(),
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: Theme.of(
                  context,
                ).textTheme.bodyMedium?.copyWith(color: cs.onSurfaceVariant),
              ),
            ),
          ],
        ),
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
            if (tab == target) {
              return;
            }
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
  const _TokenCards({
    required this.tokens,
    required this.resolvedTokens,
    required this.tokenMetadatas,
    required this.isTokensLoading,
    required this.tokenLoadingByMint,
  });

  final List<String> tokens;
  final Map<String, TokenResolveDTO> resolvedTokens;
  final Map<String, TokenMetadataDTO> tokenMetadatas;

  /// ✅ tokens が空の時でも skeleton を出すための全体ロード状態
  final bool isTokensLoading;

  /// ✅ mint ごとのロード状態
  final Map<String, bool> tokenLoadingByMint;

  @override
  Widget build(BuildContext context) {
    // ✅ tokens が空でも、ロード中なら skeleton を出す
    if (tokens.isEmpty) {
      if (isTokensLoading) {
        return AppFixedExtentGrid(
          crossAxisCount: 3,
          spacing: 10,
          childAspectRatio: 0.82,
          extraTextLines: 1,
          shrinkWrap: true,
          physics: const NeverScrollableScrollPhysics(),
          itemCount: 6,
          itemBuilder: (context, index) {
            return const TokenCard(mintAddress: '', isLoading: true);
          },
        );
      }

      return Text(
        'トークンはまだありません。',
        style: Theme.of(context).textTheme.bodyMedium?.copyWith(
          color: Theme.of(context).colorScheme.onSurfaceVariant,
        ),
      );
    }

    // ✅ 3列グリッド表示（カード高さ固定）
    return AppFixedExtentGrid(
      crossAxisCount: 3,
      spacing: 10,
      childAspectRatio: 0.82,
      extraTextLines: 1,
      shrinkWrap: true,
      physics: const NeverScrollableScrollPhysics(),
      itemCount: tokens.length,
      itemBuilder: (context, index) {
        final m = tokens[index].trim();

        final resolved = resolvedTokens[m];
        final meta = tokenMetadatas[m];

        // ✅ mint 単位のロード判定（案A）
        final loading = (tokenLoadingByMint[m] ?? isTokensLoading);

        return TokenCard(
          mintAddress: m,
          resolved: resolved,
          metadata: meta,
          isLoading: loading,
        );
      },
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
