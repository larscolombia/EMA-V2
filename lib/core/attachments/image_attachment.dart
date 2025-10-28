import 'dart:convert';

class ImageAttachment {
  final String uid;
  final String fileName; // Nombre del archivo de imagen
  final String mimeType; // Tipo de archivo (image/jpeg, image/png, etc)
  final String filePath; // Ruta local del archivo de imagen (en el dispositivo)
  final int fileSize; // Tamaño del archivo en bytes

  ImageAttachment({
    required this.uid,
    required this.fileName,
    required this.mimeType,
    required this.filePath,
    required this.fileSize,
  });

  factory ImageAttachment.fromMap(Map<String, dynamic> map) {
    return ImageAttachment(
      uid: map['uid'] as String,
      fileName: map['fileName'] as String,
      mimeType: map['mimeType'] as String,
      filePath: map['filePath'] as String,
      fileSize: map['fileSize'] as int,
    );
  }

  factory ImageAttachment.fromJson(String source) =>
      ImageAttachment.fromMap(json.decode(source) as Map<String, dynamic>);

  Map<String, dynamic> toMap() {
    return <String, dynamic>{
      'uid': uid,
      'fileName': fileName,
      'mimeType': mimeType,
      'filePath': filePath,
      'fileSize': fileSize,
    };
  }

  String toJson() => json.encode(toMap());
}
