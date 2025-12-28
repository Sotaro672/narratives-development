// frontend/sns/lib/features/auth/application/shipping_address_service.dart
import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:firebase_auth/firebase_auth.dart';
import 'package:http/http.dart' as http;

import '../../user/infrastructure/user_repository_http.dart';
import '../../shippingAddress/infrastructure/shipping_address_repository_http.dart';

class VerifyEmailState {
  const VerifyEmailState({
    required this.verifying,
    required this.verified,
    required this.error,
  });

  final bool verifying;
  final bool verified;
  final String? error;

  static const idle = VerifyEmailState(
    verifying: false,
    verified: false,
    error: null,
  );

  VerifyEmailState copyWith({bool? verifying, bool? verified, String? error}) {
    return VerifyEmailState(
      verifying: verifying ?? this.verifying,
      verified: verified ?? this.verified,
      error: error,
    );
  }
}

class ZipLookupResult {
  const ZipLookupResult.success({
    required this.pref,
    required this.city,
    required this.town,
  }) : ok = true,
       error = null;

  const ZipLookupResult.failure(this.error)
    : ok = false,
      pref = '',
      city = '',
      town = '';

  final bool ok;
  final String pref;
  final String city;
  final String town;
  final String? error;
}

class SaveAddressResult {
  const SaveAddressResult.success(this.message) : ok = true, error = null;

  const SaveAddressResult.failure(this.error) : ok = false, message = null;

  final bool ok;
  final String? message;
  final String? error;
}

class ShippingAddressService {
  ShippingAddressService({
    FirebaseAuth? auth,
    http.Client? httpClient,
    UserRepositoryHttp? userRepo,
    ShippingAddressRepositoryHttp? shipRepo,
    String? baseUrl,
  }) : _auth = auth ?? FirebaseAuth.instance,
       _http = httpClient ?? http.Client(),
       _baseUrl = (baseUrl ?? _resolveApiBase()).trim(),
       _userRepo = userRepo,
       _shipRepo = shipRepo {
    // repo が未注入なら baseUrl で作る
    final b = _baseUrl.endsWith('/')
        ? _baseUrl.substring(0, _baseUrl.length - 1)
        : _baseUrl;
    _userRepoInst = _userRepo ?? UserRepositoryHttp(baseUrl: b);
    _shipRepoInst = _shipRepo ?? ShippingAddressRepositoryHttp(baseUrl: b);
  }

  final FirebaseAuth _auth;
  final http.Client _http;

  final String _baseUrl;

  final UserRepositoryHttp? _userRepo;
  final ShippingAddressRepositoryHttp? _shipRepo;

  late final UserRepositoryHttp _userRepoInst;
  late final ShippingAddressRepositoryHttp _shipRepoInst;

  static const String _fallbackBaseUrl =
      'https://narratives-backend-871263659099.asia-northeast1.run.app';

  static String _resolveApiBase() {
    const fromDefine = String.fromEnvironment('API_BASE_URL');
    final base = (fromDefine.isNotEmpty ? fromDefine : _fallbackBaseUrl).trim();
    return base.endsWith('/') ? base.substring(0, base.length - 1) : base;
  }

  void dispose() {
    // repo は自前で作った場合のみ dispose したいが、
    // 現状判定が難しいので “必ず dispose” 方針にする（repo 側が安全に実装されている前提）。
    _userRepoInst.dispose();
    _shipRepoInst.dispose();
    _http.close();
  }

  String s(String? v) => (v ?? '').trim();

  void log(String msg) {
    if (!kDebugMode) return;
    debugPrint(msg);
  }

  // ------------------------------------------------------------
  // routing helpers
  // ------------------------------------------------------------

  String backTo({required String? from, required String? continueUrl}) {
    final f = s(from);
    if (f.isNotEmpty) return f;

    final cu = s(continueUrl);
    if (cu.isNotEmpty) {
      final uri = Uri.tryParse(cu);
      if (uri != null) {
        final path = uri.path.isEmpty ? '/' : uri.path;
        final query = uri.query;
        return query.isEmpty ? path : '$path?$query';
      }
    }
    return '/';
  }

  bool cameFromEmailLink({required String? mode, required String? oobCode}) {
    final m = s(mode);
    final o = s(oobCode);
    return (m == 'verifyEmail' && o.isNotEmpty);
  }

  bool get loggedIn => _auth.currentUser != null;

  bool get emailVerified => _auth.currentUser?.emailVerified ?? false;

  // ------------------------------------------------------------
  // verify email action code
  // ------------------------------------------------------------

  Future<VerifyEmailState> applyActionCodeIfNeeded({
    required String? mode,
    required String? oobCode,
  }) async {
    final m = s(mode);
    final oob = s(oobCode);

    // ✅ verifyEmail 以外 / oobCode なしは何もしない
    if (m != 'verifyEmail' || oob.isEmpty) {
      return VerifyEmailState.idle;
    }

    try {
      await _auth.applyActionCode(oob);

      final user = _auth.currentUser;
      if (user != null) {
        await user.reload();
      }

      return const VerifyEmailState(
        verifying: false,
        verified: true,
        error: null,
      );
    } on FirebaseAuthException catch (e) {
      return VerifyEmailState(
        verifying: false,
        verified: false,
        error: friendlyActionError(e),
      );
    } catch (e) {
      return VerifyEmailState(
        verifying: false,
        verified: false,
        error: e.toString(),
      );
    }
  }

  String friendlyActionError(FirebaseAuthException e) {
    switch (e.code) {
      case 'expired-action-code':
        return 'この認証リンクは期限切れです。もう一度認証メールを送信してください。';
      case 'invalid-action-code':
        return 'この認証リンクは無効です。もう一度認証メールを送信してください。';
      case 'user-disabled':
        return 'このアカウントは無効化されています。';
      default:
        return e.message ?? 'メール認証に失敗しました。';
    }
  }

  // ------------------------------------------------------------
  // zip lookup
  // ------------------------------------------------------------

  String normalizeZip(String input) => input.replaceAll(RegExp(r'[^0-9]'), '');

  Future<ZipLookupResult> lookupZip(String zip7) async {
    try {
      final uri = Uri.parse(
        'https://zipcloud.ibsnet.co.jp/api/search?zipcode=$zip7',
      );
      final res = await _http.get(uri);

      if (res.statusCode != 200) {
        return ZipLookupResult.failure('住所検索に失敗しました（HTTP ${res.statusCode}）。');
      }

      final json = jsonDecode(res.body) as Map<String, dynamic>;
      final status = json['status'];
      if (status != 200) {
        final msg = (json['message'] ?? '住所検索に失敗しました。').toString();
        return ZipLookupResult.failure(msg);
      }

      final results = json['results'];
      if (results == null) {
        final msg = (json['message'] ?? '該当する住所が見つかりませんでした。').toString();
        return ZipLookupResult.failure(msg);
      }

      final list = results as List<dynamic>;
      if (list.isEmpty) {
        return const ZipLookupResult.failure('該当する住所が見つかりませんでした。');
      }

      final r0 = list.first as Map<String, dynamic>;
      final pref = (r0['address1'] ?? '').toString();
      final city = (r0['address2'] ?? '').toString();
      final town = (r0['address3'] ?? '').toString();

      return ZipLookupResult.success(pref: pref, city: city, town: town);
    } catch (e) {
      return ZipLookupResult.failure(e.toString());
    }
  }

  // ------------------------------------------------------------
  // save
  // ------------------------------------------------------------

  bool canSaveAddress({
    required bool saving,
    required String lastName,
    required String firstName,
    required String zip,
    required String pref,
    required String city,
    required String addr1,
  }) {
    final fbUser = _auth.currentUser;
    final isLoggedIn = fbUser != null;

    return !saving &&
        isLoggedIn &&
        s(lastName).isNotEmpty &&
        s(firstName).isNotEmpty &&
        normalizeZip(zip).isNotEmpty &&
        s(pref).isNotEmpty &&
        s(city).isNotEmpty &&
        s(addr1).isNotEmpty;
  }

  Future<SaveAddressResult> saveAddress({
    required String lastName,
    required String lastNameKana,
    required String firstName,
    required String firstNameKana,
    required String zip,
    required String pref,
    required String city,
    required String addr1,
    required String addr2,
  }) async {
    final fbUser = _auth.currentUser;
    if (fbUser == null) {
      return const SaveAddressResult.failure('サインインが必要です。');
    }

    final uid = fbUser.uid.trim();
    if (uid.isEmpty) {
      return const SaveAddressResult.failure('uid が取得できませんでした。');
    }

    final ln = s(lastName);
    final lnk = s(lastNameKana);
    final fn = s(firstName);
    final fnk = s(firstNameKana);

    final zip7 = normalizeZip(zip);
    final st = s(pref);
    final ct = s(city);
    final a1 = s(addr1);
    final a2 = s(addr2);

    bool userSaved = false;
    String? userErr;

    try {
      // ----------------------------
      // 1) upsert user (best-effort)
      // ----------------------------
      log(
        '[ShippingAddress] upsert user uid=$uid '
        'lastName="$ln" lastNameKana="$lnk" firstName="$fn" firstNameKana="$fnk"',
      );

      try {
        await _userRepoInst.create(
          CreateUserBody(
            id: uid,
            firstName: fn,
            firstNameKana: fnk.isEmpty ? null : fnk,
            lastName: ln,
            lastNameKana: lnk.isEmpty ? null : lnk,
          ),
        );
        userSaved = true;
      } catch (e) {
        log('[ShippingAddress] user create failed -> try update. err=$e');
        try {
          await _userRepoInst.update(
            uid,
            UpdateUserBody(
              firstName: fn,
              firstNameKana: fnk.isEmpty ? null : fnk,
              lastName: ln,
              lastNameKana: lnk.isEmpty ? null : lnk,
            ),
          );
          userSaved = true;
        } catch (e2) {
          userErr = e2.toString();
          log('[ShippingAddress] user update failed. err=$e2');
        }
      }

      // ----------------------------
      // 2) upsert shipping address (required)
      // ----------------------------
      log('[ShippingAddress] upsert shippingAddress userId=$uid');

      final created = await tryUpsertShippingAddress(
        uid: uid,
        zip7: zip7,
        pref: st,
        city: ct,
        addr1: a1,
        addr2: a2,
      );

      final sb = StringBuffer()
        ..writeln('配送先情報を保存しました。')
        ..writeln('shippingAddressId=${created.id} userId=${created.userId}');
      if (userSaved) {
        sb.writeln('user: saved');
      } else if (userErr != null) {
        sb.writeln('user: failed (non-blocking) $userErr');
      }

      return SaveAddressResult.success(sb.toString().trim());
    } catch (e) {
      return SaveAddressResult.failure(e.toString());
    }
  }

  Future<ShippingAddress> tryUpsertShippingAddress({
    required String uid,
    required String zip7,
    required String pref,
    required String city,
    required String addr1,
    required String addr2,
  }) async {
    try {
      return await _shipRepoInst.create(
        CreateShippingAddressInput(
          userId: uid,
          zipCode: zip7,
          state: pref,
          city: city,
          street: addr1,
          street2: addr2.isEmpty ? null : addr2,
          country: 'JP',
        ),
      );
    } catch (e) {
      log('[ShippingAddress] shipping create failed -> try update. err=$e');

      // NOTE: id=uid 前提（あなたの backend の設計に合わせて）
      return await _shipRepoInst.update(
        uid,
        UpdateShippingAddressInput(
          zipCode: zip7,
          state: pref,
          city: city,
          street: addr1,
          street2: addr2, // "" を渡すと消去扱いにできる
          country: 'JP',
        ),
      );
    }
  }
}
