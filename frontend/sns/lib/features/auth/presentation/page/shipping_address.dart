// frontend/sns/lib/features/auth/presentation/page/shipping_address.dart
import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:firebase_auth/firebase_auth.dart';
import 'package:http/http.dart' as http;

import '../../../../app/shell/presentation/components/header.dart';

// ✅ repositories
import '../../../user/infrastructure/user_repository_http.dart';
import '../../../shippingAddress/infrastructure/shipping_address_repository_http.dart';

/// ✅ 認証メールリンクの着地点 + 配送先住所入力
class ShippingAddressPage extends StatefulWidget {
  const ShippingAddressPage({
    super.key,
    this.mode,
    this.oobCode,
    this.continueUrl,
    this.lang,
    this.from,
    this.intent,
  });

  /// Firebase action params
  final String? mode; // e.g. verifyEmail
  final String? oobCode;
  final String? continueUrl;
  final String? lang;

  /// optional app params
  final String? from;
  final String? intent;

  @override
  State<ShippingAddressPage> createState() => _ShippingAddressPageState();
}

class _ShippingAddressPageState extends State<ShippingAddressPage> {
  // ============================================================
  // API base (match list_repository_http.dart behavior)
  // ============================================================
  static const String _fallbackBaseUrl =
      'https://narratives-backend-871263659099.asia-northeast1.run.app';

  String _resolveApiBase() {
    const fromDefine = String.fromEnvironment('API_BASE_URL');
    final base = (fromDefine.isNotEmpty ? fromDefine : _fallbackBaseUrl).trim();
    return base.endsWith('/') ? base.substring(0, base.length - 1) : base;
  }

  // ============================================================
  // repositories
  // ============================================================
  late final UserRepositoryHttp _userRepo;
  late final ShippingAddressRepositoryHttp _shipRepo;

  bool _verifying = false;
  bool _verified = false;
  String? _verifyError;

  // ---- profile form ----
  final _lastNameCtrl = TextEditingController();
  final _lastNameKanaCtrl = TextEditingController();
  final _firstNameCtrl = TextEditingController();
  final _firstNameKanaCtrl = TextEditingController();

  // ---- address form ----
  final _zipCtrl = TextEditingController();
  final _prefCtrl = TextEditingController();
  final _cityCtrl = TextEditingController();
  final _addr1Ctrl = TextEditingController();
  final _addr2Ctrl = TextEditingController();

  bool _zipLoading = false;
  String? _zipError;

  bool _saving = false;
  String? _saveMsg;

  @override
  void initState() {
    super.initState();

    final baseUrl = _resolveApiBase();

    // ✅ Repos init
    _userRepo = UserRepositoryHttp(baseUrl: baseUrl);
    _shipRepo = ShippingAddressRepositoryHttp(baseUrl: baseUrl);

    _maybeApplyActionCode();

    // ✅ 郵便番号が変わったら自動で検索（7桁になったタイミング）
    _zipCtrl.addListener(_onZipChanged);

    // ✅ ボタン enable/disable を即時反映
    _lastNameCtrl.addListener(_onFormChanged);
    _lastNameKanaCtrl.addListener(_onFormChanged);
    _firstNameCtrl.addListener(_onFormChanged);
    _firstNameKanaCtrl.addListener(_onFormChanged);

    _prefCtrl.addListener(_onFormChanged);
    _cityCtrl.addListener(_onFormChanged);
    _addr1Ctrl.addListener(_onFormChanged);
    _addr2Ctrl.addListener(_onFormChanged);
  }

  @override
  void dispose() {
    _zipCtrl.removeListener(_onZipChanged);

    _lastNameCtrl.removeListener(_onFormChanged);
    _lastNameKanaCtrl.removeListener(_onFormChanged);
    _firstNameCtrl.removeListener(_onFormChanged);
    _firstNameKanaCtrl.removeListener(_onFormChanged);
    _prefCtrl.removeListener(_onFormChanged);
    _cityCtrl.removeListener(_onFormChanged);
    _addr1Ctrl.removeListener(_onFormChanged);
    _addr2Ctrl.removeListener(_onFormChanged);

    _lastNameCtrl.dispose();
    _lastNameKanaCtrl.dispose();
    _firstNameCtrl.dispose();
    _firstNameKanaCtrl.dispose();

    _zipCtrl.dispose();
    _prefCtrl.dispose();
    _cityCtrl.dispose();
    _addr1Ctrl.dispose();
    _addr2Ctrl.dispose();

    _userRepo.dispose();
    _shipRepo.dispose();

    super.dispose();
  }

  void _onFormChanged() {
    if (!mounted) return;
    setState(() {});
  }

  String _s(String? v) => (v ?? '').trim();

  void _log(String msg) {
    if (!kDebugMode) return;
    debugPrint(msg);
  }

  /// ✅ “戻る” の遷移先
  String _backTo() {
    final from = _s(widget.from);
    if (from.isNotEmpty) return from;

    final cu = _s(widget.continueUrl);
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

  Future<void> _maybeApplyActionCode() async {
    final mode = _s(widget.mode);
    final oob = _s(widget.oobCode);

    // ✅ verifyEmail 以外 / oobCode なしは何もしない
    if (mode != 'verifyEmail' || oob.isEmpty) {
      return;
    }

    if (mounted) {
      setState(() {
        _verifying = true;
        _verified = false;
        _verifyError = null;
      });
    }

    try {
      await FirebaseAuth.instance.applyActionCode(oob);

      final user = FirebaseAuth.instance.currentUser;
      if (user != null) {
        await user.reload();
      }

      if (mounted) {
        setState(() {
          _verified = true;
        });
      }
    } on FirebaseAuthException catch (e) {
      if (mounted) {
        setState(() {
          _verifyError = _friendlyActionError(e);
        });
      }
    } catch (e) {
      if (mounted) {
        setState(() {
          _verifyError = e.toString();
        });
      }
    } finally {
      if (mounted) {
        setState(() {
          _verifying = false;
        });
      }
    }
  }

  String _friendlyActionError(FirebaseAuthException e) {
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

  // ----------------------------------------------------------------
  // 郵便番号 → 住所自動入力
  // ----------------------------------------------------------------

  String _normalizeZip(String s) {
    return s.replaceAll(RegExp(r'[^0-9]'), '');
  }

  String? _lastResolvedZip;

  void _onZipChanged() {
    final zip = _normalizeZip(_zipCtrl.text);

    if (zip.length == 7 && zip != _lastResolvedZip) {
      _lastResolvedZip = zip;
      _lookupZipAndFill(zip);
    } else {
      if (_zipError != null) {
        setState(() => _zipError = null);
      }
    }

    if (mounted) {
      setState(() {});
    }
  }

  Future<void> _lookupZipAndFill(String zip7) async {
    if (_zipLoading) return;

    if (mounted) {
      setState(() {
        _zipLoading = true;
        _zipError = null;
      });
    }

    try {
      final uri = Uri.parse(
        'https://zipcloud.ibsnet.co.jp/api/search?zipcode=$zip7',
      );
      final res = await http.get(uri);

      if (res.statusCode != 200) {
        throw StateError('住所検索に失敗しました（HTTP ${res.statusCode}）。');
      }

      final json = jsonDecode(res.body) as Map<String, dynamic>;

      final status = json['status'];
      if (status != 200) {
        final msg = (json['message'] ?? '住所検索に失敗しました。').toString();
        throw StateError(msg);
      }

      final results = json['results'];
      if (results == null) {
        final msg = (json['message'] ?? '該当する住所が見つかりませんでした。').toString();
        throw StateError(msg);
      }

      final list = results as List<dynamic>;
      if (list.isEmpty) {
        throw StateError('該当する住所が見つかりませんでした。');
      }

      final r0 = list.first as Map<String, dynamic>;
      final pref = (r0['address1'] ?? '').toString();
      final city = (r0['address2'] ?? '').toString();
      final town = (r0['address3'] ?? '').toString();

      _prefCtrl.text = pref;
      _cityCtrl.text = city;
      _addr1Ctrl.text = town;

      if (mounted) {
        setState(() {
          _zipError = null;
        });
      }
    } catch (e) {
      if (mounted) {
        setState(() {
          _zipError = e.toString();
        });
      }
    } finally {
      if (mounted) {
        setState(() {
          _zipLoading = false;
        });
      }
    }
  }

  // ----------------------------------------------------------------
  // Save (USER + SHIPPING_ADDRESS)
  // ----------------------------------------------------------------

  bool get _canSaveAddress {
    final fbUser = FirebaseAuth.instance.currentUser;
    final loggedIn = fbUser != null;

    return !_saving &&
        loggedIn &&
        _s(_lastNameCtrl.text).isNotEmpty &&
        _s(_firstNameCtrl.text).isNotEmpty &&
        _normalizeZip(_zipCtrl.text).isNotEmpty &&
        _s(_prefCtrl.text).isNotEmpty &&
        _s(_cityCtrl.text).isNotEmpty &&
        _s(_addr1Ctrl.text).isNotEmpty;
  }

  Future<void> _saveAddressToBackend() async {
    final fbUser = FirebaseAuth.instance.currentUser;
    if (fbUser == null) {
      setState(() {
        _saveMsg = 'サインインが必要です。';
      });
      return;
    }

    final uid = fbUser.uid.trim();
    if (uid.isEmpty) {
      setState(() {
        _saveMsg = 'uid が取得できませんでした。';
      });
      return;
    }

    // ✅ 4つの入力値
    final lastName = _s(_lastNameCtrl.text);
    final lastNameKana = _s(_lastNameKanaCtrl.text);
    final firstName = _s(_firstNameCtrl.text);
    final firstNameKana = _s(_firstNameKanaCtrl.text);

    final zip7 = _normalizeZip(_zipCtrl.text);
    final pref = _s(_prefCtrl.text);
    final city = _s(_cityCtrl.text);
    final addr1 = _s(_addr1Ctrl.text);
    final addr2 = _s(_addr2Ctrl.text);

    if (mounted) {
      setState(() {
        _saveMsg = null;
        _saving = true;
      });
    }

    bool userSaved = false;
    String? userErr;

    try {
      // ----------------------------
      // 1) upsert user (best-effort)
      // ----------------------------
      _log(
        '[ShippingAddressPage] upsert user uid=$uid '
        'lastName="$lastName" lastNameKana="$lastNameKana" '
        'firstName="$firstName" firstNameKana="$firstNameKana"',
      );

      try {
        await _userRepo.create(
          CreateUserBody(
            id: uid,
            firstName: firstName,
            firstNameKana: firstNameKana.isEmpty ? null : firstNameKana,
            lastName: lastName,
            lastNameKana: lastNameKana.isEmpty ? null : lastNameKana,
          ),
        );
        userSaved = true;
      } catch (e) {
        _log('[ShippingAddressPage] user create failed -> try update. err=$e');
        try {
          await _userRepo.update(
            uid,
            UpdateUserBody(
              firstName: firstName,
              firstNameKana: firstNameKana.isEmpty ? null : firstNameKana,
              lastName: lastName,
              lastNameKana: lastNameKana.isEmpty ? null : lastNameKana,
            ),
          );
          userSaved = true;
        } catch (e2) {
          userErr = e2.toString();
          _log('[ShippingAddressPage] user update failed. err=$e2');
        }
      }

      // ----------------------------
      // 2) upsert shipping address (required)
      // ----------------------------
      _log('[ShippingAddressPage] upsert shippingAddress userId=$uid');

      final created = await _tryUpsertShippingAddress(
        uid: uid,
        zip7: zip7,
        pref: pref,
        city: city,
        addr1: addr1,
        addr2: addr2,
      );

      if (!mounted) return;

      final msg = StringBuffer()
        ..writeln('配送先情報を保存しました。')
        ..writeln('shippingAddressId=${created.id} userId=${created.userId}');
      if (userSaved) {
        msg.writeln('user: saved');
      } else if (userErr != null) {
        msg.writeln('user: failed (non-blocking) $userErr');
      }

      setState(() {
        _saveMsg = msg.toString().trim();
      });

      context.go('/billing-address');
    } catch (e) {
      if (mounted) {
        setState(() {
          _saveMsg = e.toString();
        });
      }
    } finally {
      if (mounted) {
        setState(() {
          _saving = false;
        });
      }
    }
  }

  Future<ShippingAddress> _tryUpsertShippingAddress({
    required String uid,
    required String zip7,
    required String pref,
    required String city,
    required String addr1,
    required String addr2,
  }) async {
    try {
      return await _shipRepo.create(
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
      _log(
        '[ShippingAddressPage] shipping create failed -> try update. err=$e',
      );

      // NOTE: id=uid 前提（あなたの backend の設計に合わせて）
      return await _shipRepo.update(
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

  @override
  Widget build(BuildContext context) {
    final backTo = _backTo();

    final user = FirebaseAuth.instance.currentUser;
    final loggedIn = user != null;
    final emailVerified = user?.emailVerified ?? false;

    final mode = _s(widget.mode);
    final oob = _s(widget.oobCode);
    final cameFromEmailLink = (mode == 'verifyEmail' && oob.isNotEmpty);

    return Scaffold(
      body: SafeArea(
        child: Column(
          children: [
            AppHeader(
              title: '配送先住所',
              showBack: true,
              backTo: backTo,
              actions: const [],
              onTapTitle: () => context.go('/'),
            ),
            Expanded(
              child: Center(
                child: ConstrainedBox(
                  constraints: const BoxConstraints(maxWidth: 520),
                  child: SingleChildScrollView(
                    padding: const EdgeInsets.all(16),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.stretch,
                      children: [
                        if (cameFromEmailLink) ...[
                          Text(
                            'メール認証の確認',
                            style: Theme.of(context).textTheme.titleLarge,
                          ),
                          const SizedBox(height: 8),
                          if (_verifying) ...[
                            const Text('確認中です…'),
                            const SizedBox(height: 12),
                            const LinearProgressIndicator(),
                            const SizedBox(height: 16),
                          ] else if (_verifyError != null) ...[
                            _InfoBox(
                              kind: _InfoKind.error,
                              text: _verifyError!,
                            ),
                            const SizedBox(height: 12),
                          ] else if (_verified) ...[
                            const _InfoBox(
                              kind: _InfoKind.ok,
                              text: 'メール認証が完了しました。続けて配送先情報を入力してください。',
                            ),
                            const SizedBox(height: 12),
                          ] else ...[
                            const _InfoBox(
                              kind: _InfoKind.info,
                              text: '認証リンクを確認します。',
                            ),
                            const SizedBox(height: 12),
                          ],
                        ] else ...[
                          const _InfoBox(
                            kind: _InfoKind.info,
                            text: '配送先情報を入力してください。',
                          ),
                          const SizedBox(height: 12),
                        ],
                        if (!loggedIn) ...[
                          const _InfoBox(
                            kind: _InfoKind.info,
                            text: '※ 住所の保存にはサインインが必要です。',
                          ),
                          const SizedBox(height: 8),
                          OutlinedButton(
                            onPressed: () {
                              final from = Uri.encodeComponent(
                                GoRouterState.of(context).uri.toString(),
                              );
                              context.go('/login?from=$from');
                            },
                            child: const Text('サインインへ'),
                          ),
                          const SizedBox(height: 16),
                        ] else if (!emailVerified) ...[
                          const _InfoBox(
                            kind: _InfoKind.info,
                            text: '※ 現在サインイン中ですが、メール認証が未完了の可能性があります。',
                          ),
                          const SizedBox(height: 16),
                        ],

                        Text(
                          'お届け先氏名',
                          style: Theme.of(context).textTheme.titleMedium,
                        ),
                        const SizedBox(height: 8),

                        // ✅ 並び替え：
                        // 1行目：苗字｜苗字かな
                        Row(
                          children: [
                            Expanded(
                              child: TextField(
                                controller: _lastNameCtrl,
                                textInputAction: TextInputAction.next,
                                decoration: const InputDecoration(
                                  labelText: '苗字',
                                  border: OutlineInputBorder(),
                                ),
                              ),
                            ),
                            const SizedBox(width: 12),
                            Expanded(
                              child: TextField(
                                controller: _lastNameKanaCtrl,
                                textInputAction: TextInputAction.next,
                                decoration: const InputDecoration(
                                  labelText: '苗字かな',
                                  border: OutlineInputBorder(),
                                ),
                              ),
                            ),
                          ],
                        ),
                        const SizedBox(height: 12),

                        // 2行目：名前｜名前かな
                        Row(
                          children: [
                            Expanded(
                              child: TextField(
                                controller: _firstNameCtrl,
                                textInputAction: TextInputAction.next,
                                decoration: const InputDecoration(
                                  labelText: '名前',
                                  border: OutlineInputBorder(),
                                ),
                              ),
                            ),
                            const SizedBox(width: 12),
                            Expanded(
                              child: TextField(
                                controller: _firstNameKanaCtrl,
                                textInputAction: TextInputAction.next,
                                decoration: const InputDecoration(
                                  labelText: '名前かな',
                                  border: OutlineInputBorder(),
                                ),
                              ),
                            ),
                          ],
                        ),

                        const SizedBox(height: 16),

                        Text(
                          '配送先住所',
                          style: Theme.of(context).textTheme.titleMedium,
                        ),
                        const SizedBox(height: 8),

                        TextField(
                          controller: _zipCtrl,
                          keyboardType: TextInputType.number,
                          decoration: InputDecoration(
                            labelText: '郵便番号（7桁）',
                            border: const OutlineInputBorder(),
                            helperText: '例: 1000001（ハイフン不要）',
                            suffixIcon: _zipLoading
                                ? const Padding(
                                    padding: EdgeInsets.all(12),
                                    child: SizedBox(
                                      width: 16,
                                      height: 16,
                                      child: CircularProgressIndicator(
                                        strokeWidth: 2,
                                      ),
                                    ),
                                  )
                                : IconButton(
                                    tooltip: '住所を自動入力',
                                    onPressed: () {
                                      final zip = _normalizeZip(_zipCtrl.text);
                                      if (zip.length == 7) {
                                        _lastResolvedZip = zip;
                                        _lookupZipAndFill(zip);
                                      } else {
                                        setState(() {
                                          _zipError = '郵便番号は7桁で入力してください。';
                                        });
                                      }
                                    },
                                    icon: const Icon(Icons.search),
                                  ),
                            errorText:
                                (_zipError == null || _zipError!.trim().isEmpty)
                                ? null
                                : _zipError,
                          ),
                        ),
                        const SizedBox(height: 12),

                        TextField(
                          controller: _prefCtrl,
                          decoration: const InputDecoration(
                            labelText: '都道府県',
                            border: OutlineInputBorder(),
                          ),
                        ),
                        const SizedBox(height: 12),

                        TextField(
                          controller: _cityCtrl,
                          decoration: const InputDecoration(
                            labelText: '市区町村',
                            border: OutlineInputBorder(),
                          ),
                        ),
                        const SizedBox(height: 12),

                        TextField(
                          controller: _addr1Ctrl,
                          decoration: const InputDecoration(
                            labelText: '住所１（町名・番地など）',
                            border: OutlineInputBorder(),
                          ),
                        ),
                        const SizedBox(height: 12),

                        TextField(
                          controller: _addr2Ctrl,
                          decoration: const InputDecoration(
                            labelText: '住所２（建物名・部屋番号など）',
                            border: OutlineInputBorder(),
                          ),
                        ),
                        const SizedBox(height: 16),

                        ElevatedButton(
                          onPressed: _canSaveAddress
                              ? _saveAddressToBackend
                              : null,
                          child: _saving
                              ? const SizedBox(
                                  width: 18,
                                  height: 18,
                                  child: CircularProgressIndicator(
                                    strokeWidth: 2,
                                  ),
                                )
                              : const Text('この住所を保存する'),
                        ),

                        if (_saveMsg != null) ...[
                          const SizedBox(height: 12),
                          _InfoBox(
                            kind: _saveMsg!.contains('保存しました')
                                ? _InfoKind.ok
                                : _InfoKind.info,
                            text: _saveMsg!,
                          ),
                        ],
                      ],
                    ),
                  ),
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

enum _InfoKind { info, ok, error }

class _InfoBox extends StatelessWidget {
  const _InfoBox({required this.kind, required this.text});

  final _InfoKind kind;
  final String text;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;

    late final Color bg;
    switch (kind) {
      case _InfoKind.ok:
        bg = scheme.primaryContainer.withValues(alpha: 0.55);
        break;
      case _InfoKind.error:
        bg = scheme.errorContainer.withValues(alpha: 0.55);
        break;
      case _InfoKind.info:
        bg = scheme.surfaceContainerHighest.withValues(alpha: 0.55);
        break;
    }

    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: bg,
        borderRadius: BorderRadius.circular(12),
      ),
      child: Text(text, style: Theme.of(context).textTheme.bodyMedium),
    );
  }
}
