class VectorStoreFile {
  final int id;
  final String vectorStoreId;
  final String fileId;
  final String filename;
  final int fileSize;
  final String status; // processing, completed, failed
  final int? uploadedBy;
  final DateTime createdAt;
  final DateTime updatedAt;

  VectorStoreFile({
    required this.id,
    required this.vectorStoreId,
    required this.fileId,
    required this.filename,
    required this.fileSize,
    required this.status,
    this.uploadedBy,
    required this.createdAt,
    required this.updatedAt,
  });

  factory VectorStoreFile.fromJson(Map<String, dynamic> json) {
    return VectorStoreFile(
      id: json['id'] as int,
      vectorStoreId: json['vector_store_id'] as String,
      fileId: json['file_id'] as String,
      filename: json['filename'] as String,
      fileSize: json['file_size'] as int? ?? 0,
      status: json['status'] as String? ?? 'processing',
      uploadedBy: json['uploaded_by'] as int?,
      createdAt: DateTime.parse(json['created_at'] as String),
      updatedAt: DateTime.parse(json['updated_at'] as String),
    );
  }

  String get formattedSize {
    if (fileSize < 1024) return '$fileSize B';
    if (fileSize < 1024 * 1024) return '${(fileSize / 1024).toStringAsFixed(1)} KB';
    if (fileSize < 1024 * 1024 * 1024) {
      return '${(fileSize / (1024 * 1024)).toStringAsFixed(1)} MB';
    }
    return '${(fileSize / (1024 * 1024 * 1024)).toStringAsFixed(2)} GB';
  }

  double get uploadProgress {
    switch (status) {
      case 'completed':
        return 1.0;
      case 'failed':
        return 0.0;
      case 'processing':
      default:
        return 0.5; // Indeterminate
    }
  }
}
