import 'dart:convert';

class PdfAttachment {
  final String uid;
  final String fileName; // Nombre del archivo PDF
  final String mimeType; // Tipo de archivo
  final String filePath; // Ruta local del archivo PDF (en el dispositivo)
  final int fileSize;    // Tamaño del archivo en bytes o KB

  PdfAttachment({
    required this.uid,
    required this.fileName,
    required this.mimeType,
    required this.filePath,
    required this.fileSize,
  });

  factory PdfAttachment.fromMap(Map<String, dynamic> map) {
    return PdfAttachment(
      uid: map['uid'] as String,
      fileName: map['fileName'] as String,
      mimeType: map['mimeType'] as String,
      filePath: map['filePath'] as String,
      fileSize: map['fileSize'] as int,
    );
  }

  factory PdfAttachment.fromJson(String source) => PdfAttachment.fromMap(json.decode(source) as Map<String, dynamic>);

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
