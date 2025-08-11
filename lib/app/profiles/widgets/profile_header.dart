import 'dart:io';
import 'package:flutter/material.dart';
import 'package:image_picker/image_picker.dart';
import 'package:get/get.dart';
import 'package:ema_educacion_medica_avanzada/core/users/user_model.dart';
import 'package:ema_educacion_medica_avanzada/config/config.dart';
import 'package:ema_educacion_medica_avanzada/app/profiles/controllers/profile_controller.dart';

class ProfileHeader extends StatefulWidget {
  final UserModel profile;
  final bool isEditable;
  final bool inDrawer;

  const ProfileHeader({
    super.key,
    required this.profile,
    this.isEditable = false,
    this.inDrawer = false,
  });

  @override
  State<ProfileHeader> createState() => _ProfileHeaderState();
}

class _ProfileHeaderState extends State<ProfileHeader> {
  bool _showOverlay = false;

  Future<void> _pickImage() async {
    print('📱 [ProfileHeader] Iniciando selección de imagen');
    
    final picker = ImagePicker();
    final pickedFile = await picker.pickImage(
      source: ImageSource.gallery,
      imageQuality: 85,
    );

    if (pickedFile != null) {
      print('✅ [ProfileHeader] Imagen seleccionada: ${pickedFile.path}');
      print('📏 [ProfileHeader] Tamaño: ${await File(pickedFile.path).length()} bytes');
      
      final ProfileController profileController = Get.find<ProfileController>();
      try {
        print('🔄 [ProfileHeader] Llamando al controlador para actualizar imagen...');
        await profileController.updateProfileImage(pickedFile);
        print('✅ [ProfileHeader] Actualización completada exitosamente');
      } catch (e) {
        print('❌ [ProfileHeader] Error al actualizar la imagen: $e');
        if (!mounted) return;
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Error al actualizar la imagen: $e')),
        );
      }
    } else {
      print('❌ [ProfileHeader] No se seleccionó ninguna imagen');
    }
  }

  final EdgeInsets _paddingDrawer =
      EdgeInsets.only(top: 25, bottom: 25, right: 16, left: 16);
  final EdgeInsets _paddingProfile =
      EdgeInsets.symmetric(horizontal: 28, vertical: 25);

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: widget.inDrawer == true ? _paddingDrawer : _paddingProfile,
      decoration: BoxDecoration(color: AppStyles.primaryColor),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.spaceBetween,
        crossAxisAlignment: CrossAxisAlignment.center,
        children: [
          // Datos del usuario (nombre, apellido)
          Expanded(
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                FittedBox(
                  fit: BoxFit.scaleDown,
                  child: Text(
                    widget.profile.firstName,
                    style: AppStyles.profileName,
                  ),
                ),
                FittedBox(
                  fit: BoxFit.scaleDown,
                  child: Text(
                    widget.profile.lastName,
                    style: AppStyles.profileLasName,
                  ),
                ),
              ],
            ),
          ),
          const SizedBox(width: 16),
          // Avatar del usuario con opción de edición
          Stack(
            children: [
              Obx(() {
                final profileImage = Get.find<ProfileController>()
                    .currentProfile
                    .value
                    .profileImage;
                return Container(
                  width: 100,
                  height: 100,
                  decoration: BoxDecoration(
                    color: AppStyles.greyColor,
                    borderRadius: BorderRadius.circular(100),
                    border: Border.all(
                      color: AppStyles.primary100,
                      width: 7,
                    ),
                  ),
                  child: ClipOval(
                    child: profileImage.isNotEmpty
                        ? Image.network(
                            profileImage,
                            fit: BoxFit.cover,
                            width: 100,
                            height: 100,
                          )
                        : const Icon(Icons.person, size: 48),
                  ),
                );
              }),
              // Si el perfil es editable, se muestra el overlay para cambiar la imagen
              if (widget.isEditable)
                Positioned.fill(
                  child: GestureDetector(
                    onTap: () async {
                      setState(() => _showOverlay = true);
                      await _pickImage();
                      setState(() => _showOverlay = false);
                    },
                    onTapDown: (_) => setState(() => _showOverlay = true),
                    onTapCancel: () => setState(() => _showOverlay = false),
                    onTapUp: (_) => setState(() => _showOverlay = false),
                    child: ClipOval(
                      child: Container(
                        width: 100,
                        height: 100,
                        color: _showOverlay
                            ? Colors.black.withAlpha((0.3 * 255).toInt())
                            : Colors.transparent,
                        alignment: Alignment.center,
                        child: _showOverlay
                            ? const Icon(
                                Icons.camera_alt,
                                color: Colors.white,
                                size: 36,
                              )
                            : null,
                      ),
                    ),
                  ),
                ),
            ],
          ),
        ],
      ),
    );
  }
}
