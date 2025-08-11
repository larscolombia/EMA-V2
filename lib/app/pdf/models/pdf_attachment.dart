class PdfAttachment {
  final String uid;
  final String fileName; // Nombre del archivo PDF
  final String mimeType; // Tipo de archivo
  final String filePath; // Ruta local del archivo PDF (en el dispositivo)
  final int fileSize; // Tamaño del archivo en bytes o KB

  PdfAttachment({
    required this.uid,
    required this.fileName,
    required this.mimeType,
    required this.filePath,
    required this.fileSize,
  });
}
