import 'dart:convert';
import 'dart:io';
import 'package:ema_educacion_medica_avanzada/config/constants/constants.dart';
import 'package:ema_educacion_medica_avanzada/app/profiles/profiles.dart';
import 'package:ema_educacion_medica_avanzada/core/core.dart';
import 'package:http/http.dart' as http;
import 'package:image_picker/image_picker.dart';
import 'package:mime/mime.dart';

class ApiProfileService extends ProfileService {
  @override
  Future<UserModel> updateProfile(UserModel profile) async {
    try {
      final url = Uri.parse('$apiUrl/user-detail/${profile.id}');
      final response = await http.post(
        url,
        headers: {
          'Authorization': 'Bearer ${profile.authToken}',
          'Content-Type': 'application/json',
          'Accept': 'application/json',
        },
        body: jsonEncode(profile.toUpdateMap()),
      );

      if (response.statusCode == 200) {
        final updatedData = jsonDecode(response.body)['data'];
        return UserModel.fromMap(updatedData);
      } else {
        throw Exception(
            'Error al actualizar el perfil: ${response.statusCode}');
      }
    } catch (e) {
      throw Exception('Error al actualizar el perfil: $e');
    }
  }

  @override
  Future<UserModel> updateProfileImage(
      UserModel profile, XFile imageFile) async {
    try {
      final url = Uri.parse('$apiUrl/user-detail/${profile.id}');
      
      print('🔄 Iniciando actualización de imagen de perfil');
      print('📡 URL: $url');
      print('👤 User ID: ${profile.id}');
      print('📁 Archivo: ${imageFile.path}');
      print('📏 Tamaño: ${await File(imageFile.path).length()} bytes');

      // Validar el archivo
      await _validateImageFile(imageFile);
      print('✅ Validación de archivo completada');

      // Crear la solicitud
      var request = http.MultipartRequest('POST', url)
        ..headers.addAll({
          'Authorization': 'Bearer ${profile.authToken}',
          'Accept': 'application/json',
        });

      print('🔑 Headers configurados: ${request.headers}');

      // Adjuntar el archivo
      final file = await http.MultipartFile.fromPath(
        'profile_image',
        imageFile.path,
        filename: imageFile.name,
      );
      request.files.add(file);
      print('📎 Archivo adjuntado: ${file.field} - ${file.filename}');

      // Enviar la solicitud
      print('🚀 Enviando solicitud al servidor...');
      final streamedResponse = await request.send();
      final responseStr = await streamedResponse.stream.bytesToString();
      
      print('📊 Respuesta del servidor:');
      print('   Status Code: ${streamedResponse.statusCode}');
      print('   Headers: ${streamedResponse.headers}');
      print('   Body: $responseStr');

      // Procesar la respuesta
      return _handleResponse(streamedResponse, responseStr);
    } on SocketException {
      print('❌ Error de conexión: No hay conexión a Internet');
      throw Exception(
          {'success': false, 'message': 'No hay conexión a Internet'});
    } on HttpException catch (e) {
      print('❌ Error HTTP: ${e.message}');
      throw Exception({'success': false, 'message': e.message});
    } catch (e) {
      print('❌ Error inesperado: ${e.toString()}');
      throw Exception({
        'success': false,
        'message': 'Error al actualizar la imagen: ${e.toString()}'
      });
    }
  }

  Future<void> _validateImageFile(XFile imageFile) async {
    print('🔍 Validando archivo de imagen...');
    
    // Verificar que el archivo existe
    if (!await File(imageFile.path).exists()) {
      print('❌ El archivo no existe: ${imageFile.path}');
      throw Exception(
          {'success': false, 'message': 'El archivo de imagen no existe'});
    }

    // Validar el tipo de archivo
    final allowedMimeTypes = ['image/jpeg', 'image/png', 'image/gif'];
    final mimeType = lookupMimeType(imageFile.path);
    print('📄 Tipo MIME detectado: $mimeType');

    if (mimeType == null || !allowedMimeTypes.contains(mimeType)) {
      print('❌ Tipo de archivo no permitido: $mimeType');
      throw Exception({
        'success': false,
        'message': 'El archivo no es una imagen válida (JPEG, PNG, GIF)'
      });
    }

    // Validar el tamaño del archivo
    final maxSize = 5 * 1024 * 1024; // 5 MB
    final fileSize = await File(imageFile.path).length();
    print('📏 Tamaño del archivo: $fileSize bytes (máximo: $maxSize bytes)');

    if (fileSize > maxSize) {
      print('❌ Archivo demasiado grande: $fileSize bytes');
      throw Exception(
          {'success': false, 'message': 'La imagen no puede ser mayor a 5 MB'});
    }
    
    print('✅ Validación completada exitosamente');
  }

  UserModel _handleResponse(
      http.StreamedResponse response, String responseStr) {
    print('🔧 Procesando respuesta del servidor...');
    
    if (response.statusCode == 200) {
      print('✅ Status 200 - Procesando datos...');
      final responseData = jsonDecode(responseStr);
      print('📋 Datos de respuesta: $responseData');

      if (responseData['data'] == null ||
          responseData['data'] is! Map<String, dynamic>) {
        print('❌ Respuesta no contiene datos válidos');
        throw Exception({
          'success': false,
          'message': 'La respuesta del servidor no contiene datos válidos'
        });
      }

      final data = responseData['data'] as Map<String, dynamic>;
      print('✅ Datos válidos encontrados, creando UserModel...');
      return UserModel.fromMap(data);
    } else {
      print('❌ Error del servidor: ${response.statusCode}');
      final responseData = jsonDecode(responseStr);
      final message = responseData['message'] ?? 'Error desconocido';
      print('📝 Mensaje de error: $message');
      throw HttpException(message, uri: response.request?.url);
    }
  }

  @override
  Future<UserModel> fetchDetailedProfile(UserModel profile) async {
    final url = Uri.parse('$apiUrl/user-detail/${profile.id}');
    final response = await http.get(
      url,
      headers: {
        'Authorization': 'Bearer ${profile.authToken}',
        'Accept': 'application/json',
      },
    );
    if (response.statusCode == 200) {
      final responseMap = jsonDecode(response.body) as Map<String, dynamic>;
      if (responseMap['data'] == null ||
          responseMap['data'] is! Map<String, dynamic>) {
        // If no detailed data, return the current profile
        return profile;
      }
      final data = responseMap['data'] as Map<String, dynamic>;
      return UserModel.fromMap(data);
    } else {
      throw Exception(
          'Error al obtener perfil detallado: ${response.statusCode}');
    }
  }
}
