import 'dart:convert';
import 'package:ema_educacion_medica_avanzada/app/subscriptions/model/subscription_model.dart';
import 'package:ema_educacion_medica_avanzada/core/roles/role.dart';

class UserModel {
  int id;
  String firstName;
  String lastName;
  String email;
  String? contact;
  String? regionCode;
  bool status;
  String language;
  bool darkMode;
  String? emailVerifiedAt;
  String? tenantId;
  DateTime createdAt;
  DateTime updatedAt;
  String fullName;
  String profileImage;
  String authToken;
  String? gender;
  int? age;
  int? countryId;
  String? countryName;
  String? city;
  String? profession;
  List<Role> roles;
  Subscription? subscription;
  List<dynamic> media;

  Subscription? get activeSubscription => subscription;

  UserModel({
    required this.id,
    required this.firstName,
    required this.lastName,
    required this.email,
    this.contact,
    this.regionCode,
    required this.status,
    required this.language,
    required this.darkMode,
    this.emailVerifiedAt,
    this.tenantId,
    required this.createdAt,
    required this.updatedAt,
    required this.fullName,
    required this.profileImage,
    required this.authToken,
    this.gender,
    this.age,
    this.countryId,
    this.countryName,
    this.city,
    this.profession,
    this.roles = const [],
    this.subscription,
    this.media = const [],
  });

  UserModel copyWith({
    String? firstName,
    String? lastName,
    String? profession,
    String? gender,
    int? age,
    String? city,
    String? profileImage,
    int? countryId,
    String? countryName,
    String? authToken,
    Subscription? subscription,
    String? email,
    // added this parameter
  }) {
    return UserModel(
      id: id,
      firstName: firstName ?? this.firstName,
      lastName: lastName ?? this.lastName,
      email: email ?? this.email,
      contact: contact,
      regionCode: regionCode,
      status: status,
      language: language,
      darkMode: darkMode,
      emailVerifiedAt: emailVerifiedAt,
      tenantId: tenantId,
      createdAt: createdAt,
      updatedAt: updatedAt,
      fullName: fullName,
      profileImage: profileImage ?? this.profileImage,
      authToken: authToken ?? this.authToken,
      gender: gender ?? this.gender,
      age: age ?? this.age,
      countryId: countryId ?? this.countryId,
      countryName: countryName ?? this.countryName,
      city: city ?? this.city,
      profession: profession ?? this.profession,
      roles: roles,
      subscription: subscription,
      media: media,
    );
  }

  factory UserModel.fromJson(String source) {
    return UserModel.fromMap(json.decode(source) as Map<String, dynamic>);
  }

  factory UserModel.fromMap(Map<String, dynamic> map) {
    print('🔧 [UserModel] Procesando datos del servidor...');
    print('📋 [UserModel] Datos recibidos: $map');

    int _asInt(dynamic v, {int def = 0}) {
      if (v == null) return def;
      if (v is int) return v;
      if (v is num) return v.toInt();
      return int.tryParse(v.toString()) ?? def;
    }

    int? _asNullableInt(dynamic v) {
      if (v == null) return null;
      if (v is int) return v;
      if (v is num) return v.toInt();
      return int.tryParse(v.toString());
    }

    bool _asBool(dynamic v, {bool def = false}) {
      if (v == null) return def;
      if (v is bool) return v;
      if (v is num) return v != 0;
      final s = v.toString().toLowerCase();
      if (s == 'true') return true;
      if (s == 'false') return false;
      final i = int.tryParse(s);
      return i != null ? i != 0 : def;
    }

    DateTime _asDate(dynamic v) {
      if (v == null) return DateTime.now();
      final s = v.toString();
      return DateTime.tryParse(s) ?? DateTime.now();
    }

    final userModel = UserModel(
      id: _asInt(map['id'], def: 0),
      firstName: map['first_name']?.toString() ?? '',
      lastName: map['last_name']?.toString() ?? '',
      email: map['email']?.toString() ?? '',
      contact: map['contact']?.toString() ?? '',
      regionCode: map['region_code']?.toString() ?? '',
      status: _asBool(map['status'], def: false),
      language: map['language']?.toString() ?? 'es',
      darkMode: _asBool(map['dark_mode']),
      emailVerifiedAt: map['email_verified_at']?.toString() ?? '',
      tenantId: map['tenant_id']?.toString() ?? '',
      createdAt: _asDate(map['created_at']),
      updatedAt: _asDate(map['updated_at']),
      fullName: map['full_name']?.toString() ?? '',
      profileImage: map['profile_image']?.toString() ?? '',
      authToken: map['token']?.toString() ?? '',
      gender: map['genero']?.toString() ?? '',
      age: _asNullableInt(map['edad']),
      countryId: _asNullableInt(map['country_id']),
      city: map['city']?.toString() ?? '',
      countryName: map['country_name']?.toString() ?? '',
      profession: map['profession']?.toString() ?? '',
      roles:
          map['roles'] != null
              ? List<Role>.from(
                (map['roles'] as List<dynamic>).map((x) => Role.fromMap(x)),
              )
              : [],
      subscription:
          map['active_subscription'] != null
              ? Subscription.fromJson(map['active_subscription'])
              : map['subscription'] != null
              ? Subscription.fromJson(map['subscription'])
              : null,
      media: map['media'] ?? [],
    );

    print('✅ [UserModel] Usuario creado exitosamente');
    print('🖼️ [UserModel] URL de imagen: ${userModel.profileImage}');

    return userModel;
  }

  Map<String, dynamic> toMap() {
    return {
      'id': id,
      'first_name': firstName,
      'last_name': lastName,
      'email': email,
      'contact': contact,
      'region_code': regionCode,
      'status': status,
      'language': language,
      'dark_mode': darkMode ? 1 : 0,
      'email_verified_at': emailVerifiedAt,
      'tenant_id': tenantId,
      'created_at': createdAt.toIso8601String(),
      'updated_at': updatedAt.toIso8601String(),
      'full_name': fullName,
      'profile_image': profileImage,
      'token': authToken,
      'genero': gender,
      'edad': age,
      'country_id': countryId,
      'country_name': countryName,
      'city': city,
      'profession': profession,
      'roles': roles.map((role) => role.toMap()).toList(),
      'subscription': subscription?.toJson(),
      'media': media,
    };
  }

  factory UserModel.unknow() {
    return UserModel(
      id: 0,
      firstName: 'Usuario',
      lastName: 'Invitado',
      email: 'invitado@ema.com',
      status: false,
      language: 'es',
      darkMode: false,
      createdAt: DateTime.now(),
      updatedAt: DateTime.now(),
      fullName: 'Usuario Invitado',
      profileImage: '',
      authToken: '',
      media: [],
      countryName: '',
    );
  }

  factory UserModel.fromLaravelApi(Map<String, dynamic> jsonData) {
    final userData = jsonData['user'] as Map<String, dynamic>;

    int _asInt(dynamic v, {int def = 0}) {
      if (v == null) return def;
      if (v is int) return v;
      if (v is num) return v.toInt();
      return int.tryParse(v.toString()) ?? def;
    }

    int? _asNullableInt(dynamic v) {
      if (v == null) return null;
      if (v is int) return v;
      if (v is num) return v.toInt();
      return int.tryParse(v.toString());
    }

    bool _asBool(dynamic v, {bool def = false}) {
      if (v == null) return def;
      if (v is bool) return v;
      if (v is num) return v != 0;
      final s = v.toString().toLowerCase();
      if (s == 'true') return true;
      if (s == 'false') return false;
      final i = int.tryParse(s);
      return i != null ? i != 0 : def;
    }

    DateTime _asDate(dynamic v) {
      if (v == null) return DateTime.now();
      final s = v.toString();
      return DateTime.tryParse(s) ?? DateTime.now();
    }

    return UserModel(
      id: _asInt(userData['id']),
      firstName: userData['first_name']?.toString() ?? '',
      lastName: userData['last_name']?.toString() ?? '',
      email: userData['email']?.toString() ?? '',
      status: _asBool(userData['status'], def: true),
      language: userData['language']?.toString() ?? 'es',
      darkMode: _asBool(userData['dark_mode']),
      createdAt: _asDate(userData['created_at']),
      updatedAt: _asDate(userData['updated_at']),
      fullName: userData['full_name']?.toString() ?? '',
      profileImage: userData['profile_image']?.toString() ?? '',
      authToken: jsonData['token']?.toString() ?? '',
      gender: userData['genero']?.toString(),
      age: _asNullableInt(userData['edad']),
      countryId: _asNullableInt(userData['country_id']),
      countryName: userData['country_name']?.toString(),
      city: userData['city']?.toString(),
      profession: userData['profession']?.toString(),
      media: userData['media'] ?? [],
      tenantId: userData['tenant_id']?.toString(),
    );
  }

  Map<String, dynamic> toUpdateMap() {
    final data = {
      'first_name': firstName,
      'last_name': lastName,
      'genero': (gender?.trim().isEmpty ?? true) ? null : gender,
      'edad': age,
      'country_id': countryId,
      'country_name':
          (countryName?.trim().isEmpty ?? true) ? null : countryName,
      'city': (city?.trim().isEmpty ?? true) ? null : city,
      'profession': (profession?.trim().isEmpty ?? true) ? null : profession,
    };

    data.removeWhere((key, value) => value == null);
    return data;
  }

  String toJson() => json.encode(toMap());
}
