// frontend\mall\lib\features\avatar\application\avatar_service.dart
import 'package:firebase_auth/firebase_auth.dart';
import 'package:flutter/foundation.dart';

import 'package:mall/app/routing/avatar_name_store.dart';
import 'package:mall/features/avatar/infrastructure/avatar_api_client.dart';
import 'package:mall/features/avatar/presentation/model/me_avatar.dart';

import 'package:mall/features/wallet/infrastructure/repository_http.dart'
    as wallet_api;
import 'package:mall/features/wallet/infrastructure/token_metadata_dto.dart';
import 'package:mall/features/wallet/infrastructure/token_resolve_dto.dart';
import 'package:mall/features/wallet/infrastructure/wallet_dto.dart';

typedef AvatarServiceLogger = void Function(String message);

/// Application service for Avatar feature.
/// - Hides direct dependencies (FirebaseAuth, API clients, repositories) from hooks/UI.
/// - Centralizes side effects like AvatarNameStore updates.
/// - Provides simple methods used by presentation hooks/controllers.
class AvatarService {
  AvatarService({
    FirebaseAuth? auth,
    AvatarApiClient? apiClient,
    wallet_api.WalletRepositoryHttp? walletRepo,
    this.enableLogging = true,
    this.logger,
  }) : _auth = auth ?? FirebaseAuth.instance,
       _apiClient =
           apiClient ??
           AvatarApiClient(
             enableLogging: true,
             logger: (m) => logger?.call('[AvatarApiClient] $m'),
           ),
       _walletRepo = walletRepo ?? wallet_api.WalletRepositoryHttp();

  final FirebaseAuth _auth;
  final AvatarApiClient _apiClient;
  final wallet_api.WalletRepositoryHttp _walletRepo;

  final bool enableLogging;
  final AvatarServiceLogger? logger;

  void _log(String msg) {
    if (!enableLogging) return;
    final out = '[AvatarService] $msg';
    logger?.call(out);
    // ignore: avoid_print
    print(out);
    debugPrint(out);
  }

  String _s(Object? v) => (v ?? '').toString().trim();

  String _short(String v, {int max = 240}) {
    final t = _s(v);
    if (t.length <= max) return t;
    return '${t.substring(0, max)}...';
  }

  /// Current signed-in user (may be null).
  User? get currentUser => _auth.currentUser;

  /// Auth change stream.
  Stream<User?> userChanges() => _auth.userChanges();

  /// Fetch my avatar profile via backend.
  /// - If user is null: clears AvatarNameStore and returns null.
  /// - On success: updates AvatarNameStore from backend-derived avatarName.
  Future<MeAvatar?> fetchMyAvatarProfile({required User? user}) async {
    if (user == null) {
      _log('fetchMyAvatarProfile skipped (user=null)');
      AvatarNameStore.I.clear();
      return null;
    }

    _log('fetchMyAvatarProfile start uid="${_s(user.uid)}"');
    final res = await _apiClient.fetchMyAvatarProfile();

    _log(
      'fetchMyAvatarProfile done '
      'avatarId="${_s(res?.avatarId)}" '
      'avatarName="${_s(res?.avatarName)}" '
      'profile="${_short(_s(res?.profile))}" '
      'walletAddress="${_s(res?.walletAddress)}"',
    );

    _log('AvatarNameStore.setAvatarName("${_s(res?.avatarName)}")');
    AvatarNameStore.I.setAvatarName(res?.avatarName);

    return res;
  }

  /// Sync and fetch wallet for current user.
  /// - If user is null or avatarId empty: returns null.
  Future<WalletDTO?> syncAndFetchMeWallet({
    required User? user,
    required String meAvatarId,
  }) async {
    if (user == null) {
      _log('syncAndFetchMeWallet skipped (user=null)');
      return null;
    }
    final id = _s(meAvatarId);
    if (id.isEmpty) {
      _log('syncAndFetchMeWallet skipped (meAvatarId empty)');
      return null;
    }

    _log('syncAndFetchMeWallet start meAvatarId="$id"');
    final w = await _walletRepo.syncAndFetchMeWallet();
    _log('syncAndFetchMeWallet done tokensLen=${w?.tokens.length ?? 0}');
    return w;
  }

  /// Resolve tokens by mint addresses.
  Future<Map<String, TokenResolveDTO>> resolveTokensByMintAddresses(
    List<String> mintAddresses,
  ) async {
    final mints = mintAddresses
        .map((e) => _s(e))
        .where((e) => e.isNotEmpty)
        .toList();

    if (mints.isEmpty) {
      _log('resolveTokensByMintAddresses skipped (no mints)');
      return <String, TokenResolveDTO>{};
    }

    // uniq
    final seen = <String>{};
    final uniq = <String>[];
    for (final m in mints) {
      if (seen.add(m)) uniq.add(m);
    }

    _log('resolveTokensByMintAddresses start uniqLen=${uniq.length}');
    final results = await Future.wait<TokenResolveDTO?>(
      uniq.map((m) async {
        try {
          return await _walletRepo.resolveTokenByMintAddress(m);
        } catch (e) {
          _log('resolveTokenByMintAddress failed mint="$m" err="$e"');
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

    _log('resolveTokensByMintAddresses done okLen=${out.length}');
    return out;
  }

  /// Fetch token metadata via proxy (CORS avoid).
  Future<Map<String, TokenMetadataDTO>> fetchTokenMetadatasByResolved(
    Map<String, TokenResolveDTO> resolved,
  ) async {
    if (resolved.isEmpty) {
      _log('fetchTokenMetadatasByResolved skipped (resolved empty)');
      return <String, TokenMetadataDTO>{};
    }

    final entries = resolved.entries
        .map((e) => MapEntry(_s(e.key), e.value))
        .where((e) => e.key.isNotEmpty && _s(e.value.metadataUri).isNotEmpty)
        .toList();

    if (entries.isEmpty) {
      _log('fetchTokenMetadatasByResolved skipped (no metadataUri)');
      return <String, TokenMetadataDTO>{};
    }

    _log('fetchTokenMetadatasByResolved start entriesLen=${entries.length}');
    final results = await Future.wait<TokenMetadataDTO?>(
      entries.map((e) async {
        try {
          return await _walletRepo.fetchTokenMetadata(e.value.metadataUri);
        } catch (err) {
          _log(
            'fetchTokenMetadata failed mint="${e.key}" uri="${_s(e.value.metadataUri)}" err="$err"',
          );
          return null;
        }
      }),
    );

    final out = <String, TokenMetadataDTO>{};
    for (var i = 0; i < entries.length; i++) {
      final dto = results[i];
      if (dto == null) continue;
      out[entries[i].key] = dto;
    }

    _log('fetchTokenMetadatasByResolved done okLen=${out.length}');
    return out;
  }

  void dispose() {
    _log('dispose');
    _apiClient.dispose();
    _walletRepo.dispose();
  }
}
