import 'package:ema_educacion_medica_avanzada/app/profiles/profiles.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:ema_educacion_medica_avanzada/config/config.dart';
import 'package:ema_educacion_medica_avanzada/core/core.dart';

class ProfileInformation extends StatelessWidget {
  final UserModel profile;
  final List<CountryModel> countries;

  const ProfileInformation({
    super.key,
    required this.countries,
    required this.profile,
  });

  TextStyle _headerStyle(BuildContext context) => Theme.of(context)
      .textTheme
      .titleLarge!
      .copyWith(color: AppStyles.primaryColor, fontWeight: FontWeight.bold);

  TextStyle _infoTitleStyle(BuildContext context) =>
      Theme.of(context).textTheme.bodySmall!.copyWith(
          color: AppStyles.primaryColor,
          fontWeight: FontWeight.w600,
          fontSize: 14);

  TextStyle _infoValueStyle(BuildContext context) =>
      Theme.of(context).textTheme.bodyMedium!.copyWith(
          color: AppStyles.primary900,
          fontWeight: FontWeight.w400,
          fontSize: 14);

  Widget _buildInfoColumn(BuildContext context, String title, String value) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(title, style: _infoTitleStyle(context)),
          const SizedBox(height: 4),
          Text(value.isNotEmpty ? value : 'Sin especificar',
              style: _infoValueStyle(context)),
        ],
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    // Usamos Obx para reaccionar a los cambios de currentProfile
    return Obx(() {
      final currentProfile = Get.find<ProfileController>().currentProfile.value;
      return Padding(
        padding: const EdgeInsets.symmetric(horizontal: 28),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Text(
              'PERFIL PROFESIONAL Y PERSONAL',
              textAlign: TextAlign.left,
              style: _headerStyle(context),
            ),
            const SizedBox(height: 24),
            Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                _buildInfoColumn(
                    context, 'Correo electrónico: ', currentProfile.email),
                _buildInfoColumn(context, 'Formación: ',
                    currentProfile.profession ?? 'Sin especificar'),
                /* _buildInfoColumn(
                    context, 'Cargo actual o futuro: ', 'Sin especificar'),
                _buildInfoColumn(
                    context, 'Temas de Interes: ', 'Sin especificar'),*/
                const SizedBox(height: 16),
                Row(
                  children: [
                    Expanded(
                      child: _buildInfoColumn(
                          context, 'Nombre: ', currentProfile.firstName),
                    ),
                    const SizedBox(width: 12),
                    Expanded(
                      child: _buildInfoColumn(
                          context, 'Apellidos: ', currentProfile.lastName),
                    ),
                  ],
                ),
                const SizedBox(height: 16),
                Row(
                  children: [
                    Expanded(
                      child: _buildInfoColumn(context, 'Género: ',
                          currentProfile.gender ?? 'Sin especificar'),
                    ),
                    const SizedBox(width: 12),
                    Expanded(
                      child: _buildInfoColumn(
                          context,
                          'Edad: ',
                          currentProfile.age != null
                              ? '${currentProfile.age} años'
                              : 'Sin especificar'),
                    ),
                  ],
                ),
                const SizedBox(height: 16),
                Row(
                  children: [
                    Expanded(
                      child: _buildInfoColumn(context, 'País: ',
                          currentProfile.countryName ?? 'Sin especificar'),
                    ),
                    const SizedBox(width: 12),
                    Expanded(
                      child: _buildInfoColumn(context, 'Ciudad: ',
                          currentProfile.city ?? 'Sin especificar'),
                    ),
                  ],
                ),
                const SizedBox(height: 24),
                Center(
                  child: ElevatedButton(
                    onPressed: () async {
                      // Abrir el diálogo de edición
                      final updatedProfile = await showDialog<UserModel>(
                        context: context,
                        builder: (context) => EditProfileDialog(
                          profile: profile,
                          countries: countries,
                        ),
                      );

                      if (updatedProfile != null) {
                        try {
                          Get.find<ProfileController>().currentProfile.value =
                              updatedProfile;
                        } catch (e) {
                          if (context.mounted) {
                            ScaffoldMessenger.of(context).showSnackBar(
                              SnackBar(
                                content:
                                    Text('Error al actualizar la información'),
                              ),
                            );
                          }
                        }
                      }
                    },
                    style: ElevatedButton.styleFrom(
                      backgroundColor: AppStyles.primary900,
                      padding: const EdgeInsets.symmetric(
                          horizontal: 24, vertical: 12),
                      shape: RoundedRectangleBorder(
                        borderRadius: BorderRadius.circular(8),
                      ),
                    ),
                    child: const Text(
                      'Editar Información',
                      style: TextStyle(
                        color: Colors.white,
                        fontSize: 16,
                        fontWeight: FontWeight.bold,
                      ),
                    ),
                  ),
                ),
              ],
            ),
          ],
        ),
      );
    });
  }
}
