// frontend/inspector/lib/screens/inspection_detail/utils/inspection_formatters.dart
String formatInspectionResultLabel(String? raw) {
  switch (raw) {
    case 'passed':
      return '合格';
    case 'failed':
      return '不合格';
    case 'notYet':
    case null:
    default:
      return '未検査';
  }
}
