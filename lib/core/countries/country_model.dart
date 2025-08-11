class CountryModel {
  int id;
  String name;
  String shortCode;
  int phoneCode;

  CountryModel({
    required this.id,
    required this.name,
    required this.shortCode,
    required this.phoneCode,
  });

  factory CountryModel.fromJson(Map<String, dynamic> json) {
    return CountryModel(
      id: json['id'] as int,
      name: json['name'] as String,
      shortCode: json['short_code'] as String,
      phoneCode: json['phone_code'] as int,
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'id': id,
      'name': name,
      'short_code': shortCode,
      'phone_code': phoneCode,
    };
  }
}
