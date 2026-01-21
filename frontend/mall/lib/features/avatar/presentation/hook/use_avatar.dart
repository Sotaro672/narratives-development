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
  String s(String? v) => (v ?? '').trim();

  // ---------------------------
  // Repository / Client lifecycle
  // ---------------------------
  final walletRepo = useMemoized(() => wallet_api.WalletRepositoryHttp());
  useEffect(() {
    return () => walletRepo.dispose();
  }, [walletRepo]);

  // ❌ const AvatarApiClient() は不可（MallAuthedApi を内部で new するため）
  final apiClient = useMemoized(() => AvatarApiClient(), const []);

  // ---------------------------
  // Auth stream
  // ---------------------------
  final authSnap = useStream<User?>(
    FirebaseAuth.instance.userChanges(),
    initialData: FirebaseAuth.instance.currentUser,
  );
  final user = FirebaseAuth.instance.currentUser ?? authSnap.data;

  // ---------------------------
  // Tab state
  // ---------------------------
  final tabState = useState<ProfileTab>(ProfileTab.tokens);

  // ---------------------------
  // Data loads
  // ---------------------------
  final meAvatarFuture = useMemoized(() => apiClient.fetchMeAvatar(), const []);
  final meAvatarSnap = useFuture<MeAvatar?>(meAvatarFuture);

  // ✅ Wallet は「自分のウォレット」をサーバ側で解決する前提
  // - URL avatarId は使わない
  // - meAvatar が取れない/空のときは wallet を読まない（無駄なI/O回避）
  final walletFuture = useMemoized(() async {
    final me = await meAvatarFuture;
    if (me == null) return null;
    if (me.avatarId.trim().isEmpty) return null;

    // ✅ sync → fetch
    return walletRepo.syncAndFetchMeWallet();
  }, [meAvatarFuture, walletRepo]);
  final walletSnap = useFuture<WalletDTO?>(walletFuture);

  // ---------------------------
  // ✅ Resolve tokens by mintAddress
  // ---------------------------
  final resolveFuture = useMemoized(() async {
    final w = await walletFuture;
    final mints = (w?.tokens ?? const <String>[])
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
  }, [walletFuture, walletRepo]);
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
  }, [resolveFuture, walletRepo]);
  final metadataSnap = useFuture<Map<String, TokenMetadataDTO>>(metadataFuture);

  // ---------------------------
  // Navigation handlers
  // ---------------------------
  void goToAvatarEditHandler() => goToAvatarEdit(context);

  // ---------------------------
  // View-derived fields
  // ---------------------------
  final tokens = walletSnap.data?.tokens ?? const <String>[];

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

  final photoUrl = user == null ? '' : s(user.photoURL);

  String profileBioFor(User u) {
    return '（プロフィール未設定）';
  }

  final bio = user == null ? '（プロフィール未設定）' : profileBioFor(user);

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
    photoUrl: photoUrl,
    bio: bio,
    goToAvatarEdit: goToAvatarEditHandler,
  );
}
