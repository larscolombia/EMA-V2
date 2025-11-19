import 'dart:io';
import 'dart:typed_data';
import 'package:ema_educacion_medica_avanzada/admin/features/books/models/vector_store.dart';
import 'package:ema_educacion_medica_avanzada/admin/features/books/models/vector_store_file.dart';
import 'package:ema_educacion_medica_avanzada/admin/features/books/services/vector_stores_service.dart';
import 'package:get/get.dart';

class VectorStoresController extends GetxController {
  final VectorStoresService _service = VectorStoresService();

  // Estado observable
  final vectorStores = <VectorStore>[].obs;
  final files = <VectorStoreFile>[].obs;
  final isLoading = false.obs;
  final isLoadingFiles = false.obs;
  final isUploading = false.obs;
  final error = ''.obs;
  final uploadProgress = 0.0.obs;

  // Vector store seleccionado
  final Rx<VectorStore?> selectedVectorStore = Rx<VectorStore?>(null);

  @override
  void onInit() {
    super.onInit();
    loadVectorStores();
  }

  // Cargar todos los vector stores
  Future<void> loadVectorStores() async {
    try {
      isLoading.value = true;
      error.value = '';
      final result = await _service.getVectorStores();
      vectorStores.value = result;

      // Si hay un vector store por defecto, seleccionarlo
      if (selectedVectorStore.value == null && result.isNotEmpty) {
        final defaultStore = result.firstWhere(
          (vs) => vs.isDefault,
          orElse: () => result.first,
        );
        await selectVectorStore(defaultStore);
      }
    } catch (e) {
      error.value = e.toString();
      Get.snackbar(
        'Error',
        'No se pudieron cargar los vector stores: ${e.toString()}',
        snackPosition: SnackPosition.BOTTOM,
      );
    } finally {
      isLoading.value = false;
    }
  }

  // Seleccionar un vector store y cargar sus archivos
  Future<void> selectVectorStore(VectorStore vectorStore) async {
    selectedVectorStore.value = vectorStore;
    await loadFiles(vectorStore.id);
  }

  // Cargar archivos de un vector store
  Future<void> loadFiles(int vectorStoreId) async {
    try {
      isLoadingFiles.value = true;
      error.value = '';
      final result = await _service.getFiles(vectorStoreId);
      files.value = result;
    } catch (e) {
      error.value = e.toString();
      Get.snackbar(
        'Error',
        'No se pudieron cargar los archivos: ${e.toString()}',
        snackPosition: SnackPosition.BOTTOM,
      );
    } finally {
      isLoadingFiles.value = false;
    }
  }

  // Crear nuevo vector store
  Future<bool> createVectorStore({
    required String name,
    required String description,
    required String category,
  }) async {
    try {
      isLoading.value = true;
      error.value = '';
      final newStore = await _service.createVectorStore(
        name: name,
        description: description,
        category: category,
      );
      await loadVectorStores();
      await selectVectorStore(newStore);
      Get.snackbar(
        'Éxito',
        'Vector store creado correctamente',
        snackPosition: SnackPosition.BOTTOM,
      );
      return true;
    } catch (e) {
      error.value = e.toString();
      Get.snackbar(
        'Error',
        'No se pudo crear el vector store: ${e.toString()}',
        snackPosition: SnackPosition.BOTTOM,
      );
      return false;
    } finally {
      isLoading.value = false;
    }
  }

  // Actualizar vector store
  Future<bool> updateVectorStore(
    int id, {
    required String name,
    required String description,
    required String category,
    bool? isDefault,
  }) async {
    try {
      isLoading.value = true;
      error.value = '';
      await _service.updateVectorStore(
        id,
        name: name,
        description: description,
        category: category,
        isDefault: isDefault,
      );
      await loadVectorStores();
      
      // Actualizar el vector store seleccionado si es el mismo
      if (selectedVectorStore.value?.id == id) {
        final updated = vectorStores.firstWhere((vs) => vs.id == id);
        selectedVectorStore.value = updated;
      }
      
      Get.snackbar(
        'Éxito',
        'Vector store actualizado correctamente',
        snackPosition: SnackPosition.BOTTOM,
      );
      return true;
    } catch (e) {
      error.value = e.toString();
      Get.snackbar(
        'Error',
        'No se pudo actualizar el vector store: ${e.toString()}',
        snackPosition: SnackPosition.BOTTOM,
      );
      return false;
    } finally {
      isLoading.value = false;
    }
  }

  // Eliminar vector store
  Future<bool> deleteVectorStore(int id) async {
    try {
      isLoading.value = true;
      error.value = '';
      await _service.deleteVectorStore(id);
      await loadVectorStores();
      
      // Si era el seleccionado, seleccionar el primero disponible
      if (selectedVectorStore.value?.id == id && vectorStores.isNotEmpty) {
        await selectVectorStore(vectorStores.first);
      }
      
      Get.snackbar(
        'Éxito',
        'Vector store eliminado correctamente',
        snackPosition: SnackPosition.BOTTOM,
      );
      return true;
    } catch (e) {
      error.value = e.toString();
      Get.snackbar(
        'Error',
        'No se pudo eliminar el vector store: ${e.toString()}',
        snackPosition: SnackPosition.BOTTOM,
      );
      return false;
    } finally {
      isLoading.value = false;
    }
  }

  // Subir archivo
  Future<bool> uploadFile(File file) async {
    if (selectedVectorStore.value == null) {
      Get.snackbar(
        'Error',
        'No hay un vector store seleccionado',
        snackPosition: SnackPosition.BOTTOM,
      );
      return false;
    }

    try {
      isUploading.value = true;
      uploadProgress.value = 0.0;
      error.value = '';

      await _service.uploadFile(selectedVectorStore.value!.id, file);
      
      uploadProgress.value = 1.0;
      
      // Recargar la lista de archivos después de subir
      await loadFiles(selectedVectorStore.value!.id);
      
      // Recargar vector stores para actualizar contadores
      await loadVectorStores();
      
      Get.snackbar(
        'Éxito',
        'Archivo subido correctamente. El indexado puede tomar unos minutos.',
        snackPosition: SnackPosition.BOTTOM,
      );
      return true;
    } catch (e) {
      error.value = e.toString();
      Get.snackbar(
        'Error',
        'No se pudo subir el archivo: ${e.toString()}',
        snackPosition: SnackPosition.BOTTOM,
      );
      return false;
    } finally {
      isUploading.value = false;
      uploadProgress.value = 0.0;
    }
  }

  // Subir archivo desde bytes (para web)
  Future<bool> uploadFileFromBytes(Uint8List bytes, String filename) async {
    if (selectedVectorStore.value == null) {
      Get.snackbar(
        'Error',
        'No hay un vector store seleccionado',
        snackPosition: SnackPosition.BOTTOM,
      );
      return false;
    }

    try {
      isUploading.value = true;
      uploadProgress.value = 0.0;
      error.value = '';

      await _service.uploadFileFromBytes(selectedVectorStore.value!.id, bytes, filename);
      
      uploadProgress.value = 1.0;
      
      // Recargar la lista de archivos después de subir
      await loadFiles(selectedVectorStore.value!.id);
      
      // Recargar vector stores para actualizar contadores
      await loadVectorStores();
      
      Get.snackbar(
        'Éxito',
        'Archivo subido correctamente. El indexado puede tomar unos minutos.',
        snackPosition: SnackPosition.BOTTOM,
      );
      return true;
    } catch (e) {
      error.value = e.toString();
      Get.snackbar(
        'Error',
        'No se pudo subir el archivo: ${e.toString()}',
        snackPosition: SnackPosition.BOTTOM,
      );
      return false;
    } finally {
      isUploading.value = false;
      uploadProgress.value = 0.0;
    }
  }

  // Eliminar archivo
  Future<bool> deleteFile(String fileId) async {
    if (selectedVectorStore.value == null) {
      Get.snackbar(
        'Error',
        'No hay un vector store seleccionado',
        snackPosition: SnackPosition.BOTTOM,
      );
      return false;
    }

    try {
      isLoadingFiles.value = true;
      error.value = '';
      
      await _service.deleteFile(selectedVectorStore.value!.id, fileId);
      
      // Recargar archivos y vector stores
      await loadFiles(selectedVectorStore.value!.id);
      await loadVectorStores();
      
      Get.snackbar(
        'Éxito',
        'Archivo eliminado correctamente',
        snackPosition: SnackPosition.BOTTOM,
      );
      return true;
    } catch (e) {
      error.value = e.toString();
      Get.snackbar(
        'Error',
        'No se pudo eliminar el archivo: ${e.toString()}',
        snackPosition: SnackPosition.BOTTOM,
      );
      return false;
    } finally {
      isLoadingFiles.value = false;
    }
  }

  // Refrescar archivos (para actualizar estados de indexado)
  Future<void> refreshFiles() async {
    if (selectedVectorStore.value != null) {
      await loadFiles(selectedVectorStore.value!.id);
    }
  }
}
