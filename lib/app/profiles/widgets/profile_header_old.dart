import 'package:ema_educacion_medica_avanzada/config/config.dart';
import 'package:ema_educacion_medica_avanzada/core/users/user_model.dart';
import 'package:flutter/material.dart';
import 'package:image_picker/image_picker.dart';


class ProfileHeaderOld extends StatefulWidget {
  final UserModel profile;
  final bool isEditable;
  final bool inDrawer;

  const ProfileHeaderOld({
    super.key,
    required this.profile,
    this.isEditable = false,
    this.inDrawer = false,
  });

  @override
  State<ProfileHeaderOld> createState() => _ProfileHeaderOldState();
}

class _ProfileHeaderOldState extends State<ProfileHeaderOld> {
  bool _showOverlay = false;

  Future<void> _pickImage() async {
    final picker = ImagePicker();
    final image = await picker.pickImage(source: ImageSource.gallery);

    if (image != null) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Imagen actualizada con éxito')),
      );
    }
  }

  final EdgeInsets _paddingDrawer = EdgeInsets.only(top: 25, bottom: 25, right: 28, left: 0);
  final EdgeInsets _paddingProfile = EdgeInsets.symmetric(horizontal: 28, vertical: 25);
  
  @override
  Widget build(BuildContext context) {
    return Container(
      padding: widget.inDrawer == true
       ? _paddingDrawer
       : _paddingProfile,
      decoration: BoxDecoration(color: AppStyles.primaryColor),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.spaceBetween,
        crossAxisAlignment: CrossAxisAlignment.end,
        children: [
          Expanded(
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                FittedBox(
                  fit: BoxFit.scaleDown,
                  child: Text(
                    widget.profile.firstName,
                    style: AppStyles.profileName,
                  ),
                ),
                // const SizedBox(height: 0),
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
              Container(
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
                  child: widget.profile.profileImage.isNotEmpty
                      ? Image.network(
                          widget.profile.profileImage,
                          fit: BoxFit.cover,
                          width: 100,
                          height: 100,
                        )
                      : const Icon(Icons.person, size: 48),
                ),
              ),
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
