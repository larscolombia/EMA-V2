class AdminUser {
  final int id;
  final String firstName;
  final String lastName;
  final String email;
  final String role;
  final String token;
  final DateTime createdAt;
  final DateTime updatedAt;

  AdminUser({
    required this.id,
    required this.firstName,
    required this.lastName,
    required this.email,
    required this.role,
    required this.token,
    required this.createdAt,
    required this.updatedAt,
  });

  String get fullName => '$firstName $lastName';
  bool get isSuperAdmin => role == 'super_admin';

  factory AdminUser.fromJson(Map<String, dynamic> json) {
    final user = json['user'] as Map<String, dynamic>;
    return AdminUser(
      id: user['id'] as int,
      firstName: user['first_name'] as String? ?? '',
      lastName: user['last_name'] as String? ?? '',
      email: user['email'] as String,
      role: user['role'] as String? ?? 'user',
      token: json['token'] as String,
      createdAt: DateTime.parse(user['created_at'] as String),
      updatedAt: DateTime.parse(user['updated_at'] as String),
    );
  }

  Map<String, dynamic> toJson() => {
        'id': id,
        'first_name': firstName,
        'last_name': lastName,
        'email': email,
        'role': role,
        'token': token,
        'created_at': createdAt.toIso8601String(),
        'updated_at': updatedAt.toIso8601String(),
      };
}
