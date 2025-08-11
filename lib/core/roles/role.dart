import 'dart:convert';


class Role {
  String name;
  
  Role({
    required this.name,
  });

  Map<String, dynamic> toMap() {
    return <String, dynamic>{
      'name': name,
    };
  }

  factory Role.fromJson(String source) {
    return Role.fromMap(json.decode(source) as Map<String, dynamic>);
  }

  factory Role.fromMap(Map<String, dynamic> map) {
    return Role(
      name: map['name'] as String,
    );
  }

  String toJson() => json.encode(toMap());
}
