class VectorStore {
  final int id;
  final String name;
  final String vectorStoreId;
  final String description;
  final String category;
  final bool isDefault;
  final int fileCount;
  final int totalBytes;
  final DateTime createdAt;
  final DateTime updatedAt;

  VectorStore({
    required this.id,
    required this.name,
    required this.vectorStoreId,
    required this.description,
    required this.category,
    required this.isDefault,
    required this.fileCount,
    required this.totalBytes,
    required this.createdAt,
    required this.updatedAt,
  });

  factory VectorStore.fromJson(Map<String, dynamic> json) {
    return VectorStore(
      id: json['id'] as int,
      name: json['name'] as String,
      vectorStoreId: json['vector_store_id'] as String,
      description: json['description'] as String? ?? '',
      category: json['category'] as String? ?? '',
      isDefault: json['is_default'] as bool? ?? false,
      fileCount: json['file_count'] as int? ?? 0,
      totalBytes: json['total_bytes'] as int? ?? 0,
      createdAt: DateTime.parse(json['created_at'] as String),
      updatedAt: DateTime.parse(json['updated_at'] as String),
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'id': id,
      'name': name,
      'vector_store_id': vectorStoreId,
      'description': description,
      'category': category,
      'is_default': isDefault,
      'file_count': fileCount,
      'total_bytes': totalBytes,
      'created_at': createdAt.toIso8601String(),
      'updated_at': updatedAt.toIso8601String(),
    };
  }

  String get formattedSize {
    if (totalBytes < 1024) return '$totalBytes B';
    if (totalBytes < 1024 * 1024) return '${(totalBytes / 1024).toStringAsFixed(1)} KB';
    if (totalBytes < 1024 * 1024 * 1024) {
      return '${(totalBytes / (1024 * 1024)).toStringAsFixed(1)} MB';
    }
    return '${(totalBytes / (1024 * 1024 * 1024)).toStringAsFixed(2)} GB';
  }
}
