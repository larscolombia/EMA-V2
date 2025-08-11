// ignore_for_file: public_member_api_docs, sort_constructors_first
import 'dart:convert';

import 'package:ema_educacion_medica_avanzada/core/token/token.dart';


class LaravelToken implements Token {
  final String token;
  final String tokenType;
  final DateTime? expiration;
  final dynamic expiresIn;

  LaravelToken({
    required this.token,
    this.expiresIn,
    this.expiration,
    this.tokenType = 'Bearer',
  });

  factory LaravelToken.fromApi(Map<String, dynamic> jsonData) {
    return LaravelToken.fromMap(jsonData['token'] as Map<String, dynamic>);
  }
  
  factory LaravelToken.fromJson(String source) {
    return LaravelToken.fromMap(json.decode(source) as Map<String, dynamic>);
  }

  factory LaravelToken.fromMap(Map<String, dynamic> map) {
    return LaravelToken(
      token: map['token'] as String,
      tokenType: map['tokenType'] as String,
      expiration: map['expiration'] != null ? DateTime.fromMillisecondsSinceEpoch(map['expiration'] as int) : null,
      expiresIn: map['expiresIn'] as dynamic,
    );
  }

  String toJson() => json.encode(toMap());

  Map<String, dynamic> toMap() {
    return <String, dynamic>{
      'token': token,
      'tokenType': tokenType,
      'expiration': expiration?.millisecondsSinceEpoch,
      'expiresIn': expiresIn,
    };
  }

  @override
  String value() => '$tokenType $token';
}
