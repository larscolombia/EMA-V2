import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';
import 'package:ema_educacion_medica_avanzada/admin/features/books/models/vector_store.dart';
import 'package:ema_educacion_medica_avanzada/admin/features/books/models/vector_store_file.dart';
import 'package:ema_educacion_medica_avanzada/config/constants/constants.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:http/http.dart' as http;
import 'package:http_parser/http_parser.dart';

class VectorStoresService {
  final _storage = const FlutterSecureStorage();

  Future<String?> _getToken() async {
    return await _storage.read(key: 'admin_token');
  }

  // Listar todos los vector stores
  Future<List<VectorStore>> getVectorStores() async {
    try {
      final token = await _getToken();
      final response = await http.get(
        Uri.parse('$apiUrl/admin/vectorstores'),
        headers: {
          'Content-Type': 'application/json',
          if (token != null) 'Authorization': 'Bearer $token',
        },
      );

      if (response.statusCode == 200) {
        final Map<String, dynamic> jsonResponse = json.decode(response.body);
        final List<dynamic> data = jsonResponse['data'] as List<dynamic>;
        return data.map((json) => VectorStore.fromJson(json)).toList();
      } else {
        throw Exception('Error al obtener vector stores: ${response.statusCode}');
      }
    } catch (e) {
      throw Exception('Error de conexión: $e');
    }
  }

  // Obtener detalles de un vector store
  Future<VectorStore> getVectorStore(int id) async {
    try {
      final token = await _getToken();
      final response = await http.get(
        Uri.parse('$apiUrl/admin/vectorstores/$id'),
        headers: {
          'Content-Type': 'application/json',
          if (token != null) 'Authorization': 'Bearer $token',
        },
      );

      if (response.statusCode == 200) {
        return VectorStore.fromJson(json.decode(response.body));
      } else {
        throw Exception('Error al obtener vector store');
      }
    } catch (e) {
      throw Exception('Error de conexión: $e');
    }
  }

  // Crear nuevo vector store
  Future<VectorStore> createVectorStore({
    required String name,
    required String description,
    required String category,
  }) async {
    try {
      final token = await _getToken();
      final response = await http.post(
        Uri.parse('$apiUrl/admin/vectorstores'),
        headers: {
          'Content-Type': 'application/json',
          if (token != null) 'Authorization': 'Bearer $token',
        },
        body: json.encode({
          'name': name,
          'description': description,
          'category': category,
        }),
      );

      if (response.statusCode == 201) {
        final data = json.decode(response.body);
        // Retornar vector store recién creado
        return await getVectorStore(data['id']);
      } else {
        final error = json.decode(response.body);
        throw Exception(error['error'] ?? 'Error al crear vector store');
      }
    } catch (e) {
      throw Exception('Error al crear vector store: $e');
    }
  }

  // Actualizar vector store
  Future<void> updateVectorStore(
    int id, {
    required String name,
    required String description,
    required String category,
    bool? isDefault,
  }) async {
    try {
      final token = await _getToken();
      final body = <String, dynamic>{
        'name': name,
        'description': description,
        'category': category,
      };
      if (isDefault != null) {
        body['is_default'] = isDefault;
      }

      final response = await http.put(
        Uri.parse('$apiUrl/admin/vectorstores/$id'),
        headers: {
          'Content-Type': 'application/json',
          if (token != null) 'Authorization': 'Bearer $token',
        },
        body: json.encode(body),
      );

      if (response.statusCode != 200) {
        final error = json.decode(response.body);
        throw Exception(error['error'] ?? 'Error al actualizar vector store');
      }
    } catch (e) {
      throw Exception('Error al actualizar vector store: $e');
    }
  }

  // Eliminar vector store
  Future<void> deleteVectorStore(int id) async {
    try {
      final token = await _getToken();
      final response = await http.delete(
        Uri.parse('$apiUrl/admin/vectorstores/$id'),
        headers: {
          'Content-Type': 'application/json',
          if (token != null) 'Authorization': 'Bearer $token',
        },
      );

      if (response.statusCode != 200) {
        final error = json.decode(response.body);
        throw Exception(error['error'] ?? 'Error al eliminar vector store');
      }
    } catch (e) {
      throw Exception('Error al eliminar vector store: $e');
    }
  }

  // Listar archivos de un vector store
  Future<List<VectorStoreFile>> getFiles(int vectorStoreId) async {
    try {
      final token = await _getToken();
      final response = await http.get(
        Uri.parse('$apiUrl/admin/vectorstores/$vectorStoreId/files'),
        headers: {
          'Content-Type': 'application/json',
          if (token != null) 'Authorization': 'Bearer $token',
        },
      );

      if (response.statusCode == 200) {
        final Map<String, dynamic> jsonResponse = json.decode(response.body);
        final List<dynamic> data = jsonResponse['data'] as List<dynamic>;
        return data.map((json) => VectorStoreFile.fromJson(json)).toList();
      } else {
        throw Exception('Error al obtener archivos');
      }
    } catch (e) {
      throw Exception('Error de conexión: $e');
    }
  }

  // Subir archivo a un vector store
  Future<void> uploadFile(int vectorStoreId, File file) async {
    try {
      final token = await _getToken();
      final request = http.MultipartRequest(
        'POST',
        Uri.parse('$apiUrl/admin/vectorstores/$vectorStoreId/upload'),
      );

      if (token != null) {
        request.headers['Authorization'] = 'Bearer $token';
      }

      // Determinar content type basado en la extensión
      String contentType = 'application/octet-stream';
      if (file.path.endsWith('.pdf')) {
        contentType = 'application/pdf';
      } else if (file.path.endsWith('.txt')) {
        contentType = 'text/plain';
      } else if (file.path.endsWith('.md')) {
        contentType = 'text/markdown';
      }

      request.files.add(
        await http.MultipartFile.fromPath(
          'file',
          file.path,
          contentType: MediaType.parse(contentType),
        ),
      );

      final streamedResponse = await request.send();
      final response = await http.Response.fromStream(streamedResponse);

      if (response.statusCode != 200) {
        final error = json.decode(response.body);
        throw Exception(error['error'] ?? 'Error al subir archivo');
      }
    } catch (e) {
      throw Exception('Error al subir archivo: $e');
    }
  }

  // Subir archivo desde bytes (para web)
  Future<void> uploadFileFromBytes(int vectorStoreId, Uint8List bytes, String filename) async {
    try {
      final token = await _getToken();
      final request = http.MultipartRequest(
        'POST',
        Uri.parse('$apiUrl/admin/vectorstores/$vectorStoreId/upload'),
      );

      if (token != null) {
        request.headers['Authorization'] = 'Bearer $token';
      }

      // Determinar content type basado en la extensión
      String contentType = 'application/octet-stream';
      if (filename.endsWith('.pdf')) {
        contentType = 'application/pdf';
      } else if (filename.endsWith('.txt')) {
        contentType = 'text/plain';
      } else if (filename.endsWith('.md')) {
        contentType = 'text/markdown';
      }

      request.files.add(
        http.MultipartFile.fromBytes(
          'file',
          bytes,
          filename: filename,
          contentType: MediaType.parse(contentType),
        ),
      );

      final streamedResponse = await request.send();
      final response = await http.Response.fromStream(streamedResponse);

      if (response.statusCode != 200) {
        final error = json.decode(response.body);
        throw Exception(error['error'] ?? 'Error al subir archivo');
      }
    } catch (e) {
      throw Exception('Error al subir archivo: $e');
    }
  }

  // Eliminar archivo de un vector store
  Future<void> deleteFile(int vectorStoreId, String fileId) async {
    try {
      final token = await _getToken();
      final response = await http.delete(
        Uri.parse('$apiUrl/admin/vectorstores/$vectorStoreId/files/$fileId'),
        headers: {
          'Content-Type': 'application/json',
          if (token != null) 'Authorization': 'Bearer $token',
        },
      );

      if (response.statusCode != 200) {
        final error = json.decode(response.body);
        throw Exception(error['error'] ?? 'Error al eliminar archivo');
      }
    } catch (e) {
      throw Exception('Error al eliminar archivo: $e');
    }
  }
}
