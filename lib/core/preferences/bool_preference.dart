import 'package:shared_preferences/shared_preferences.dart';

class BoolPreference {
  // late SharedPreferences? _prefs;
  final String key;
  final bool defaultValue;
  bool? _value;

  BoolPreference({
    required this.key,
    required this.defaultValue
  });

  Future<bool> getValue() async {
    final prefs = await SharedPreferences.getInstance();
    _value = prefs.getBool(key)?? defaultValue;
    return _value!;
  }

  setValue(bool value) async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setBool(key, value);
    _value = value;
  }
}
