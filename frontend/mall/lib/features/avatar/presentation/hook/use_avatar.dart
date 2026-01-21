// frontend\mall\lib\features\avatar\presentation\hook\use_avatar.dart
import 'package:firebase_auth/firebase_auth.dart';
import 'package:flutter/material.dart';
import 'package:flutter_hooks/flutter_hooks.dart';

// ✅ prefix を付けて、WalletRepositoryHttp だけを参照する
import 'package:mall/features/wallet/infrastructure/repository_http.dart'
    as wallet_api;

import 'package:mall/features/wallet/infrastructure/token_metadata_dto.dart';
import 'package:mall/features/wallet/infrastructure/token_resolve_dto.dart';
import 'package:mall/features/wallet/infrastructure/wallet_dto.dart';

import 'package:mall/features/avatar/infrastructure/avatar_api_client.dart';
import 'package:mall/features/avatar/presentation/model/avatar_vm.dart';
import 'package:mall/features/avatar/presentation/model/me_avatar.dart';

// ✅ Pattern B: navigation helpers are centralized here
import 'package:mall/app/routing/navigation.dart';

/// Pattern B:
/// - URL の `from` を廃止
/// - URL から avatarId を読まない / URL に avatarId を入れない
/// - 戻り先制御は NavStore 側（router.dart / 各ページのUI側）で行う
AvatarVm useAvatarVm(BuildContext context) {
  // ---------------------------
  // Repository / Client lifecycle
  // ---------------------------
  final walletRepo = useMemoized(() => wallet_api.WalletRepositoryHttp());
  useEffect(() {
    return () => walletRepo.dispose();
  }, [walletRepo]);

  // ---------------------------
  // Auth stream
  // ---------------------------
  final authSnap = useStream<User?>(
    FirebaseAuth.instance.userChanges(),
    initialData: FirebaseAuth.instance.currentUser,
  );
  final user = FirebaseAuth.instance.currentUser ?? authSnap.data;

  // ✅ user が変わったら apiClient を作り直す（内部で token を掴む/キャッシュする実装でも安全に）
  final apiClient = useMemoized(() => AvatarApiClient(), [user?.uid]);
  useEffect(() {
    return () => apiClient.dispose();
  }, [apiClient]);

  // ---------------------------
  // Tab state
  // ---------------------------
  final tabState = useState<ProfileTab>(ProfileTab.tokens);

  // ---------------------------
  // Data loads
  // ---------------------------
  // ✅ /mall/me/avatar（MeAvatar=patch全体）
  // ✅ user が変わったら取り直す
  final meAvatarFuture = useMemoized(() async {
    // サインアウト直後などで user が null のときは叩かない（無駄なI/O/不整合回避）
    if (user == null) return null;
    // ✅ /mall/me/avatar は「avatar patch 全体」を返す前提に統一
    return apiClient.fetchMyAvatarProfile();
  }, [apiClient, user?.uid]);
  final meAvatarSnap = useFuture<MeAvatar?>(meAvatarFuture);

  // ✅ Wallet は「自分のウォレット」をサーバ側で解決する前提
  // - URL avatarId は使わない
  // - meAvatar が取れない/空のときは wallet を読まない（無駄なI/O回避）
  //
  // 重要: meAvatarFuture を await する形だと、初回 Future 固定の影響を受けやすい。
  //       meAvatarSnap.data に基づいて walletFuture を作り直す。
  final meAvatarId = (meAvatarSnap.data?.avatarId ?? '').trim();

  final walletFuture = useMemoized(() async {
    if (user == null) return null;
    if (meAvatarId.isEmpty) return null;

    // ✅ sync → fetch
    return walletRepo.syncAndFetchMeWallet();
  }, [walletRepo, user?.uid, meAvatarId]);
  final walletSnap = useFuture<WalletDTO?>(walletFuture);

  // ---------------------------
  // ✅ Resolve tokens by mintAddress
  // ---------------------------
  // walletSnap.data を基準に作り直す（Future の固定参照を避ける）
  final tokensFromWallet = walletSnap.data?.tokens ?? const <String>[];

  final resolveFuture = useMemoized(() async {
    final mints = tokensFromWallet
        .map((e) => e.trim())
        .where((e) => e.isNotEmpty)
        .toList();

    if (mints.isEmpty) return <String, TokenResolveDTO>{};

    // 重複排除（順序維持）
    final seen = <String>{};
    final uniq = <String>[];
    for (final m in mints) {
      if (seen.add(m)) uniq.add(m);
    }

    final results = await Future.wait<TokenResolveDTO?>(
      uniq.map((m) async {
        try {
          return await walletRepo.resolveTokenByMintAddress(m);
        } catch (_) {
          return null;
        }
      }),
    );

    final out = <String, TokenResolveDTO>{};
    for (var i = 0; i < uniq.length; i++) {
      final dto = results[i];
      if (dto == null) continue;
      out[uniq[i]] = dto;
    }
    return out;
  }, [walletRepo, user?.uid, tokensFromWallet.join(',')]); // tokens の変化で再計算
  final resolvedSnap = useFuture<Map<String, TokenResolveDTO>>(resolveFuture);

  // ---------------------------
  // ✅ Fetch token metadata via proxy (CORS avoid)
  // ---------------------------
  final metadataFuture = useMemoized(() async {
    final resolved = await resolveFuture;
    if (resolved.isEmpty) return <String, TokenMetadataDTO>{};

    final entries = resolved.entries
        .map((e) => MapEntry(e.key.trim(), e.value))
        .where((e) => e.key.isNotEmpty && e.value.metadataUri.trim().isNotEmpty)
        .toList();

    if (entries.isEmpty) return <String, TokenMetadataDTO>{};

    final results = await Future.wait<TokenMetadataDTO?>(
      entries.map((e) async {
        try {
          return await walletRepo.fetchTokenMetadata(e.value.metadataUri);
        } catch (_) {
          return null;
        }
      }),
    );

    final out = <String, TokenMetadataDTO>{};
    for (var i = 0; i < entries.length; i++) {
      final dto = results[i];
      if (dto == null) continue;
      out[entries[i].key] = dto; // key = mintAddress
    }
    return out;
  }, [walletRepo, user?.uid, resolveFuture]);
  final metadataSnap = useFuture<Map<String, TokenMetadataDTO>>(metadataFuture);

  // ---------------------------
  // Navigation handlers
  // ---------------------------
  void goToAvatarEditHandler() => goToAvatarEdit(context);

  // ---------------------------
  // View-derived fields
  // ---------------------------
  final tokens = tokensFromWallet;

  final resolvedTokens = resolvedSnap.data ?? const <String, TokenResolveDTO>{};
  final tokenMetadatas =
      metadataSnap.data ?? const <String, TokenMetadataDTO>{};

  // ✅ 全体ロード状態（tokens が空の時の skeleton 用にも使う）
  final isTokensLoading =
      walletSnap.connectionState == ConnectionState.waiting ||
      resolvedSnap.connectionState == ConnectionState.waiting ||
      metadataSnap.connectionState == ConnectionState.waiting;

  // ✅ mint ごとのロード状態（案A）
  final tokenLoadingByMint = <String, bool>{};
  for (final raw in tokens) {
    final m = raw.trim();
    if (m.isEmpty) continue;

    final hasResolved = resolvedTokens.containsKey(m);
    final hasMeta = tokenMetadatas.containsKey(m);

    // まだ全体が進行中、または mint 単位で未取得なら loading 扱い
    tokenLoadingByMint[m] = isTokensLoading || !hasResolved || !hasMeta;
  }

  final counts = ProfileCounts(
    postCount: 0,
    followerCount: 0,
    followingCount: 0,
    tokenCount: tokens.length,
  );

  // ✅ 絶対正スキーマ: avatarIcon / profile / externalLink
  final me = meAvatarSnap.data;
  final avatarIcon = me?.avatarIcon;
  final profile = me?.profile;

  return AvatarVm(
    authSnap: authSnap,
    user: user,
    meAvatarSnap: meAvatarSnap,
    walletSnap: walletSnap,
    tokens: tokens,
    resolvedTokens: resolvedTokens,
    tokenMetadatas: tokenMetadatas,
    isTokensLoading: isTokensLoading,
    tokenLoadingByMint: tokenLoadingByMint,
    counts: counts,
    tab: tabState.value,
    setTab: (next) => tabState.value = next,
    avatarIcon: avatarIcon,
    profile: profile,
    goToAvatarEdit: goToAvatarEditHandler,
  );
}
