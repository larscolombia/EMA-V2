import 'package:shared_preferences/shared_preferences.dart';

class StringPreference {
  // late SharedPreferences? _prefs;
  final String key;
  final String defaultValue;
  String? _value;

  StringPreference({
    required this.key,
    required this.defaultValue
  });

  Future<String> getValue() async {
    final prefs = await SharedPreferences.getInstance();
    _value = prefs.getString(key) ?? defaultValue;
    return _value!;
  }

  setValue(String value) async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(key, value);
    _value = value;
  }
}
