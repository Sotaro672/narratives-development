// frontend/mall/lib/features/auth/application/shipping_address_service.dart
import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:firebase_auth/firebase_auth.dart';
import 'package:http/http.dart' as http;
import 'package:dio/dio.dart';

import '../../../app/config/api_base.dart';

import '../../user/infrastructure/user_repository_http.dart';
import '../../shippingAddress/infrastructure/repository_http.dart';

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

    /// ✅ baseUrl は「ルート」(SNS/Mall 名を含めない) を渡す想定
    /// 例: https://...run.app
    String? baseUrl,
  }) : _auth = auth ?? FirebaseAuth.instance,
       _http = httpClient ?? http.Client(),
       _baseUrl = _normalizeBaseUrl((baseUrl ?? resolveApiBase()).trim()),
       _userRepo = userRepo,
       _shipRepo = shipRepo {
    final b = _baseUrl;
    _userRepoInst = _userRepo ?? UserRepositoryHttp(baseUrl: b);
    _shipRepoInst = _shipRepo ?? ShippingAddressRepositoryHttp(baseUrl: b);
  }

  final FirebaseAuth _auth;
  final http.Client _http;

  /// ✅ ルートURL（末尾スラッシュなし）
  final String _baseUrl;

  final UserRepositoryHttp? _userRepo;
  final ShippingAddressRepositoryHttp? _shipRepo;

  late final UserRepositoryHttp _userRepoInst;
  late final ShippingAddressRepositoryHttp _shipRepoInst;

  void dispose() {
    try {
      _userRepoInst.dispose();
    } catch (_) {}
    try {
      _shipRepoInst.dispose();
    } catch (_) {}
    try {
      _http.close();
    } catch (_) {}
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

    final zip7 = normalizeZip(zip);

    return !saving &&
        isLoggedIn &&
        s(lastName).isNotEmpty &&
        s(firstName).isNotEmpty &&
        zip7.length == 7 &&
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

    // token取得（デバッグ用ログ）
    try {
      final rawToken = await fbUser.getIdToken(true);
      final token = rawToken.toString().trim();
      if (token.isEmpty) {
        return const SaveAddressResult.failure(
          '認証トークンが取得できませんでした。再ログインしてください。',
        );
      }
      log('[ShippingAddress] idToken(len)=${token.length}');
    } catch (e) {
      return SaveAddressResult.failure('認証トークン取得に失敗しました: $e');
    }

    final ln = s(lastName);
    final lnk = s(lastNameKana);
    final fn = s(firstName);
    final fnk = s(firstNameKana);

    final zip7 = normalizeZip(zip);
    if (zip7.length != 7) {
      return const SaveAddressResult.failure('郵便番号は7桁で入力してください。');
    }

    final st = s(pref);
    final ct = s(city);
    final a1 = s(addr1);
    final a2 = s(addr2);

    try {
      // ----------------------------
      // 1) create/update user (REQUIRED)
      //    ✅ me設計（idは送らない）
      // ----------------------------
      log(
        '[ShippingAddress] upsert user uid=$uid '
        'lastName="$ln" lastNameKana="$lnk" firstName="$fn" firstNameKana="$fnk"',
      );

      try {
        await _userRepoInst.create(
          CreateUserBody(
            firstName: fn,
            firstNameKana: fnk.isEmpty ? null : fnk,
            lastName: ln,
            lastNameKana: lnk.isEmpty ? null : lnk,
          ),
        );
        log('[ShippingAddress] user create: OK');
      } catch (e) {
        log('[ShippingAddress] user create failed -> try updateMe. err=$e');
        try {
          await _userRepoInst.updateMe(
            UpdateUserBody(
              firstName: fn,
              firstNameKana: fnk.isEmpty ? null : fnk,
              lastName: ln,
              lastNameKana: lnk.isEmpty ? null : lnk,
            ),
          );
          log('[ShippingAddress] user updateMe: OK');
        } catch (e2) {
          log('[ShippingAddress] user updateMe failed. err=$e2');
          return SaveAddressResult.failure('user保存に失敗しました: $e2');
        }
      }

      // ----------------------------
      // 2) create shipping address (REQUIRED)
      //    ✅ POSTは必ず /mall/me/shipping-addresses（idなし）
      //    ✅ 既存があるなら list -> patch で更新
      // ----------------------------
      log('[ShippingAddress] create/upsert shippingAddress uid=$uid');

      final saved = await upsertShippingAddress(
        zip7: zip7,
        pref: st,
        city: ct,
        addr1: a1,
        addr2: a2,
      );

      final sb = StringBuffer()
        ..writeln('配送先情報を保存しました。')
        ..writeln('shippingAddressId=${saved.id} userId=${saved.userId}');

      return SaveAddressResult.success(sb.toString().trim());
    } catch (e) {
      return SaveAddressResult.failure('配送先情報の保存に失敗しました: $e');
    }
  }

  /// ✅ POSTは必ず id なし
  /// - まず createMine()（POST /me/shipping-addresses）
  /// - 409/CONFLICT の場合は listMine() で既存 id を取り、updateById(PATCH) する
  Future<ShippingAddress> upsertShippingAddress({
    required String zip7,
    required String pref,
    required String city,
    required String addr1,
    required String addr2,
  }) async {
    if (zip7.trim().length != 7) {
      throw ArgumentError('zipCode must be 7 digits. got="$zip7"');
    }

    log(
      '[ShippingAddress] upsertShippingAddress '
      'zip=$zip7 state="$pref" city="$city" street="$addr1" street2="$addr2"',
    );

    try {
      // ✅ create (idなしPOST)
      return await _shipRepoInst.createMine(
        zipCode: zip7,
        state: pref,
        city: city,
        street: addr1,
        street2: addr2.isEmpty ? null : addr2,
        country: 'JP',
      );
    } on DioException catch (e) {
      final status = e.response?.statusCode;

      // 既存がある（重複）想定：409/CONFLICT を優先的に吸収
      if (status == 409) {
        log('[ShippingAddress] createMine conflict -> try updateById');
      } else {
        // backendが 400/409 以外で「すでにある」を返すケースもあるので、
        // "conflict" が含まれるなら同様に扱う
        final body = e.response?.data?.toString() ?? '';
        final msg = (e.message ?? '');
        final looksConflict =
            body.contains('conflict') || msg.contains('conflict');
        if (!looksConflict) {
          rethrow;
        }
        log('[ShippingAddress] createMine looksConflict -> try updateById');
      }

      final list = await _shipRepoInst.listMine();
      if (list.isEmpty) {
        throw Exception('shippingAddress exists but listMine returned empty');
      }

      final current = list.first;
      return await _shipRepoInst.updateById(
        current.id,
        UpdateShippingAddressInput(
          zipCode: zip7,
          state: pref,
          city: city,
          street: addr1,
          street2: addr2, // "" を渡せば消える仕様
          country: 'JP',
        ),
      );
    }
  }

  // ------------------------------------------------------------
  // helpers
  // ------------------------------------------------------------

  static String _normalizeBaseUrl(String v) {
    final s = v.trim();
    if (s.isEmpty) return '';
    return s.replaceAll(RegExp(r'\/+$'), '');
  }
}
