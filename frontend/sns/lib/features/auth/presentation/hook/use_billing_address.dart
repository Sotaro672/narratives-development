// frontend/sns/lib/features/auth/presentation/hook/use_billing_address.dart
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

/// BillingAddressPage の状態/処理をまとめる（Flutter では ChangeNotifier で hook 風にする）
class UseBillingAddress extends ChangeNotifier {
  UseBillingAddress({required this.from});

  /// optional back route
  final String? from;

  final cardNumberCtrl = TextEditingController();
  final cardHolderCtrl = TextEditingController();
  final cvcCtrl = TextEditingController();

  bool saving = false;
  String? msg;

  @override
  void dispose() {
    cardNumberCtrl.dispose();
    cardHolderCtrl.dispose();
    cvcCtrl.dispose();
    super.dispose();
  }

  String _s(String? v) => (v ?? '').trim();

  String backTo() {
    final f = _s(from);
    if (f.isNotEmpty) return f;
    return '/shipping-address';
  }

  // 全角数字を半角に寄せる（日本語IME対策）
  String _toAsciiDigits(String s) {
    return s.replaceAllMapped(RegExp(r'[０-９]'), (m) {
      final code = m[0]!.codeUnitAt(0);
      return String.fromCharCode(code - 0xFEE0);
    });
  }

  String normalizeDigits(String s) {
    final ascii = _toAsciiDigits(s);
    return ascii.replaceAll(RegExp(r'[^0-9]'), '');
  }

  String normalizeCardNumber(String s) {
    // ハイフン/スペース/全角数字などを除去して数字だけに
    final ascii = _toAsciiDigits(s);
    return ascii.replaceAll(RegExp(r'[^0-9]'), '');
  }

  String formatCardNumberForDisplay(String s) {
    final digits = normalizeCardNumber(s);
    final buf = StringBuffer();
    for (var i = 0; i < digits.length; i++) {
      if (i != 0 && i % 4 == 0) buf.write(' ');
      buf.write(digits[i]);
    }
    return buf.toString();
  }

  bool get canSave {
    if (saving) return false;

    final card = normalizeCardNumber(cardNumberCtrl.text);
    final holder = _s(cardHolderCtrl.text);
    final cvc = normalizeDigits(cvcCtrl.text);

    // 雛形: ざっくり必須チェック（Luhn等は後で）
    if (card.length < 12) return false; // AMEX等も考慮し ">=12" 程度に
    if (holder.isEmpty) return false;
    if (cvc.length != 3) return false;
    return true;
  }

  void onCardNumberChanged() {
    // 入力中の見た目を整える（強制しすぎない程度）
    final current = cardNumberCtrl.text;
    final formatted = formatCardNumberForDisplay(current);
    if (formatted == current) return;

    final sel = cardNumberCtrl.selection;
    cardNumberCtrl.value = TextEditingValue(
      text: formatted,
      selection: TextSelection.collapsed(
        offset: (sel.baseOffset + (formatted.length - current.length)).clamp(
          0,
          formatted.length,
        ),
      ),
    );

    notifyListeners();
  }

  void onFormChanged() {
    // holder/cvc など、単にボタン enable 更新のため
    notifyListeners();
  }

  Future<void> saveDummy(BuildContext context) async {
    saving = true;
    msg = null;
    notifyListeners();

    try {
      await Future<void>.delayed(const Duration(milliseconds: 600));

      msg = '請求情報を保存しました（ダミー）。';
      notifyListeners();

      if (!context.mounted) return;
      // ✅ 保存後に avatar_create へ遷移
      context.go('/avatar-create');
    } catch (e) {
      msg = e.toString();
      notifyListeners();
    } finally {
      saving = false;
      notifyListeners();
    }
  }
}
