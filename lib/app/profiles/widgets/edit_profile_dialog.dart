import 'package:ema_educacion_medica_avanzada/app/profiles/profiles.dart';
import 'package:ema_educacion_medica_avanzada/config/styles/app_styles.dart';
import 'package:flutter/material.dart';
import 'package:get/get.dart';
import 'package:ema_educacion_medica_avanzada/core/core.dart';
import 'package:country_flags/country_flags.dart';

class EditProfileDialog extends StatefulWidget {
  final UserModel profile;
  final List<CountryModel> countries;

  const EditProfileDialog({
    super.key,
    required this.profile,
    required this.countries,
  });

  @override
  EditProfileDialogState createState() => EditProfileDialogState();
}

class EditProfileDialogState extends State<EditProfileDialog> {
  final _formKey = GlobalKey<FormState>();

  late TextEditingController _nameController;
  late TextEditingController _surnameController;
  late TextEditingController _professionController;
  late TextEditingController _ageController;
  late TextEditingController _cityController;

  int? _selectedCountryId;
  String? _selectedGender;

  final List<String> _genderOptions = ['Hombre', 'Mujer', 'Otro'];

  @override
  void initState() {
    super.initState();
    _nameController = TextEditingController(text: widget.profile.firstName);
    _surnameController = TextEditingController(text: widget.profile.lastName);
    _professionController = TextEditingController(
      text: widget.profile.profession,
    );
    _ageController = TextEditingController(
      text: widget.profile.age?.toString(),
    );
    _cityController = TextEditingController(text: widget.profile.city);
    _selectedCountryId = widget.profile.countryId;

    // Verificar que el género actual esté en las opciones válidas
    final currentGender = widget.profile.gender;

    if (currentGender != null && _genderOptions.contains(currentGender)) {
      _selectedGender = currentGender;
    } else {
      // Si el género actual no está en las opciones válidas, usar null
      _selectedGender = null;
    }
  }

  TextStyle _headerStyle(BuildContext context) {
    return Theme.of(context).textTheme.titleLarge!.copyWith(
      color: AppStyles.primaryColor,
      fontWeight: FontWeight.bold,
    );
  }

  Future<void> _saveProfile() async {
    if (!_formKey.currentState!.validate()) return;

    try {
      final selectedCountry = widget.countries.firstWhere(
        (c) => c.id == _selectedCountryId,
        orElse:
            () => CountryModel(
              id: 0,
              name: 'Sin especificar',
              shortCode: 'XX',
              phoneCode: 00,
            ),
      );

      final updatedProfile = widget.profile.copyWith(
        firstName: _nameController.text,
        lastName: _surnameController.text,
        profession: _professionController.text,
        gender: _selectedGender,
        age: int.tryParse(_ageController.text),
        city: _cityController.text,
        countryId: _selectedCountryId,
        countryName: selectedCountry.name,
      );

      final controller = Get.find<ProfileController>();
      await controller.updateProfile(updatedProfile);
      // No need to call Get.back() here as it's handled in the controller
    } catch (error) {
      Get.snackbar(
        'Error',
        'Error al actualizar la información',
        snackPosition: SnackPosition.TOP,
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    final keyboardHeight = MediaQuery.of(context).viewInsets.bottom;

    return LayoutBuilder(
      builder: (context, constraints) {
        return AnimatedPadding(
          padding: EdgeInsets.only(bottom: keyboardHeight),
          duration: const Duration(milliseconds: 300),
          curve: Curves.easeOut,
          child: Material(
            color: Colors.transparent,
            child: Container(
              height: constraints.maxHeight,
              decoration: const BoxDecoration(
                color: Colors.white,
                borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
                boxShadow: [
                  BoxShadow(
                    color: Colors.black26,
                    blurRadius: 10,
                    spreadRadius: 2,
                    offset: Offset(0, 4),
                  ),
                ],
              ),
              child: Column(
                children: [
                  // Header del modal con estilo acorde a ProfileInformation
                  Padding(
                    padding: const EdgeInsets.all(16.0),
                    child: Row(
                      mainAxisAlignment: MainAxisAlignment.spaceBetween,
                      children: [
                        Text(
                          'Editar Información',
                          style: _headerStyle(context),
                        ),
                        IconButton(
                          icon: const Icon(Icons.close),
                          onPressed: () => Navigator.of(context).pop(),
                        ),
                      ],
                    ),
                  ),
                  Expanded(
                    child: SingleChildScrollView(
                      padding: const EdgeInsets.symmetric(horizontal: 16),
                      child: Form(
                        key: _formKey,
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            _buildTextFormField(
                              controller: _nameController,
                              label: 'Nombre',
                              validator:
                                  (value) =>
                                      value!.isEmpty ? 'Campo requerido' : null,
                            ),
                            _buildTextFormField(
                              controller: _surnameController,
                              label: 'Apellidos',
                              validator:
                                  (value) =>
                                      value!.isEmpty ? 'Campo requerido' : null,
                            ),
                            _buildTextFormField(
                              controller: _professionController,
                              label: 'Formación',
                            ),
                            _buildTextFormField(
                              controller: _ageController,
                              label: 'Edad',
                              keyboardType: TextInputType.number,
                            ),
                            DropdownButtonFormField<String>(
                              value: _selectedGender,
                              decoration: const InputDecoration(
                                filled: true,
                                fillColor: Colors.white,
                                border: OutlineInputBorder(
                                  borderRadius: BorderRadius.all(
                                    Radius.circular(12),
                                  ),
                                ),
                                hintText: 'Género',
                              ),
                              items:
                                  _genderOptions.map((gender) {
                                    return DropdownMenuItem<String>(
                                      value: gender,
                                      child: Text(gender),
                                    );
                                  }).toList(),
                              onChanged: (value) {
                                setState(() {
                                  _selectedGender = value;
                                });
                              },
                            ),
                            const SizedBox(height: 16),
                            DropdownButtonFormField<int>(
                              value: _selectedCountryId,
                              decoration: const InputDecoration(
                                labelText: 'País',
                                filled: true,
                                fillColor: Colors.white,
                                border: OutlineInputBorder(
                                  borderRadius: BorderRadius.all(
                                    Radius.circular(12),
                                  ),
                                ),
                              ),
                              items:
                                  widget.countries.map((country) {
                                    return DropdownMenuItem<int>(
                                      value: country.id,
                                      child: Row(
                                        children: [
                                          CountryFlag.fromCountryCode(
                                            country.shortCode,
                                            height: 20,
                                            width: 30,
                                          ),
                                          const SizedBox(width: 8),
                                          Text(country.name),
                                        ],
                                      ),
                                    );
                                  }).toList(),
                              onChanged: (value) {
                                setState(() {
                                  _selectedCountryId = value;
                                  _cityController.clear();
                                });
                              },
                            ),
                            const SizedBox(height: 16),
                            _buildTextFormField(
                              controller: _cityController,
                              label: 'Ciudad',
                              enabled: _selectedCountryId != null,
                            ),
                          ],
                        ),
                      ),
                    ),
                  ),
                  // Botones de acción, adaptados a la proporción de colores y forma
                  Container(
                    padding: const EdgeInsets.all(16.0),
                    decoration: const BoxDecoration(
                      color: AppStyles.primary900,
                    ),
                    child: Row(
                      mainAxisAlignment: MainAxisAlignment.spaceBetween,
                      children: [
                        TextButton(
                          onPressed: () => Navigator.of(context).pop(),
                          style: TextButton.styleFrom(
                            backgroundColor: Colors.transparent,
                            foregroundColor: Colors.white,
                            shape: RoundedRectangleBorder(
                              borderRadius: BorderRadius.circular(8),
                              side: const BorderSide(color: Colors.white),
                            ),
                            padding: const EdgeInsets.symmetric(
                              horizontal: 20,
                              vertical: 12,
                            ),
                          ),
                          child: const Text('Cancelar'),
                        ),
                        ElevatedButton(
                          onPressed: _saveProfile,
                          style: ElevatedButton.styleFrom(
                            backgroundColor: Colors.white,
                            foregroundColor: AppStyles.primary900,
                            shape: RoundedRectangleBorder(
                              borderRadius: BorderRadius.circular(8),
                            ),
                            padding: const EdgeInsets.symmetric(
                              horizontal: 20,
                              vertical: 12,
                            ),
                          ),
                          child: const Text('Guardar'),
                        ),
                      ],
                    ),
                  ),
                ],
              ),
            ),
          ),
        );
      },
    );
  }

  Widget _buildTextFormField({
    required TextEditingController controller,
    required String label,
    TextInputType keyboardType = TextInputType.text,
    String? Function(String?)? validator,
    bool enabled = true,
  }) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 16),
      child: TextFormField(
        controller: controller,
        enabled: enabled,
        decoration: InputDecoration(
          labelText: label,
          filled: true,
          fillColor: Colors.white,
          border: OutlineInputBorder(borderRadius: BorderRadius.circular(12)),
        ),
        keyboardType: keyboardType,
        validator: validator,
      ),
    );
  }
}
