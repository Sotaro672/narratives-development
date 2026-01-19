// frontend/mall/lib/features/avatar/presentation/hook/use_avatar.dart
import 'package:firebase_auth/firebase_auth.dart';
import 'package:flutter/material.dart';
import 'package:flutter_hooks/flutter_hooks.dart';

import 'package:mall/app/routing/routes.dart';

// ✅ prefix を付けて、WalletRepositoryHttp だけを参照する
import 'package:mall/features/wallet/infrastructure/repository_http.dart'
    as wallet_api;

import 'package:mall/features/wallet/infrastructure/token_metadata_dto.dart';
import 'package:mall/features/wallet/infrastructure/token_resolve_dto.dart';
import 'package:mall/features/wallet/infrastructure/wallet_dto.dart';

import 'package:mall/features/avatar/infrastructure/avatar_api_client.dart';
import 'package:mall/features/avatar/presentation/model/avatar_vm.dart';
import 'package:mall/features/avatar/presentation/navigation/avatar_navigation.dart';
import 'package:mall/features/avatar/presentation/model/me_avatar.dart';

AvatarVm useAvatarVm(BuildContext context, {String? from}) {
  String s(String? v) => (v ?? '').trim();

  // ---------------------------
  // Repository / Client lifecycle
  // ---------------------------
  final walletRepo = useMemoized(() => wallet_api.WalletRepositoryHttp());
  useEffect(() {
    return () => walletRepo.dispose();
  }, [walletRepo]);

  final apiClient = useMemoized(() => const AvatarApiClient(), const []);

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
  final urlAvatarId = resolveAvatarIdFromUrl(context);

  final meAvatarFuture = useMemoized(() => apiClient.fetchMeAvatar(), const []);
  final meAvatarSnap = useFuture<MeAvatar?>(meAvatarFuture);

  final walletFuture = useMemoized(() async {
    final me = await meAvatarFuture;
    if (me == null) return null;

    final urlAid = urlAvatarId.trim();
    final effectiveAid = urlAid.isNotEmpty ? urlAid : me.avatarId;
    if (effectiveAid.trim().isEmpty) return null;

    // ✅ sync → fetch
    return walletRepo.syncAndFetchMeWallet();
  }, [urlAvatarId, meAvatarFuture, walletRepo]);
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
  // URL normalize + auto-return
  // ---------------------------
  final normalizedUrlOnce = useRef(false);
  final returnedToFromOnce = useRef(false);

  useEffect(() {
    final me = meAvatarSnap.data;
    if (me == null || me.avatarId.trim().isEmpty) return null;

    final effectiveAid = urlAvatarId.trim().isNotEmpty
        ? urlAvatarId.trim()
        : me.avatarId;

    ensureAvatarIdInUrl(
      context: context,
      avatarId: effectiveAid,
      alreadyNormalized: normalizedUrlOnce.value,
      markNormalized: () => normalizedUrlOnce.value = true,
    );

    maybeReturnToFrom(
      context: context,
      avatarId: effectiveAid,
      from: from,
      alreadyReturned: returnedToFromOnce.value,
      markReturned: () => returnedToFromOnce.value = true,
    );

    return null;
  }, [meAvatarSnap.data?.avatarId, urlAvatarId, from]);

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

  final counts = ProfileCounts(
    postCount: 0,
    followerCount: 0,
    followingCount: 0,
    tokenCount: tokens.length,
  );

  final backTo = effectiveFrom(context, from: from);

  final loginUri = Uri(
    path: '/login',
    queryParameters: {AppQueryKey.from: backTo},
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
    counts: counts,
    tab: tabState.value,
    setTab: (next) => tabState.value = next,
    backTo: backTo,
    loginUri: loginUri,
    photoUrl: photoUrl,
    bio: bio,
    goToAvatarEdit: goToAvatarEditHandler,
  );
}
